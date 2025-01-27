package app

import (
	"context"
	"encoding/json"
	"fmt"
	unicore "github.com/mcyouyou/unicore/api/deployer/v1"
	"github.com/mcyouyou/unicore/api/deployer/v1/inplace"
	"gomodules.xyz/jsonpatch/v2"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var containerImagePatchRex = regexp.MustCompile("^/spec/containers/([0-9]+)/image$")

const InPlaceUpdateReady corev1.PodConditionType = "InPlaceUpdateReady"

// InPlaceUpdatePod try in-place update the pod. returns err if the change can't use in-place update; otherwise returns the grace-second if set
func (c *StateController) InPlaceUpdatePod(app *unicore.App, pod *corev1.Pod, updateRevision *apps.ControllerRevision,
	revisions []*apps.ControllerRevision) (error, int) {

	// find the pod's revision raw json data, create a json patch of updateRevision
	var oldRevision *apps.ControllerRevision
	for _, r := range revisions {
		if r.Name == getPodRevision(pod) {
			oldRevision = r
			break
		}
	}
	if oldRevision == nil {
		return fmt.Errorf("pod's revision not found, can't use in-place update: %s", getPodRevision(pod)), 0
	}
	patches, err := jsonpatch.CreatePatch(oldRevision.Data.Raw, updateRevision.Data.Raw)
	if err != nil {
		return err, 0
	}
	oldTemplate, err := GetTemplateFromRevision(oldRevision)
	if err != nil {
		return err, 0
	}

	// get image-to-update from patches
	newImage := make(map[string]string)
	for _, op := range patches {
		op.Path = strings.Replace(op.Path, "/spec/template", "", 1)
		// check if match /spec/containers/xxx/image
		if op.Operation != "replace" || !containerImagePatchRex.MatchString(op.Path) {
			return fmt.Errorf("patch cannot use inplace-update: %s", op.Path), 0
		}
		index, _ := strconv.Atoi(strings.Split(op.Path, "/")[3])
		if len(oldTemplate.Spec.Containers) <= index {
			return fmt.Errorf("container index out of range: %s", op.Path), 0
		}
		newImage[oldTemplate.Spec.Containers[index].Name] = op.Value.(string)
	}

	// set ready condition to false if not set yet
	statusBytes := []byte(pod.Annotations[inplace.StatusKey])
	if ContainsReadinessGate(pod) && !IfConditionFalse(pod, InPlaceUpdateReady) {
		condition := corev1.PodCondition{
			Type:               InPlaceUpdateReady,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "StartInPlaceUpdate",
		}

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			origin, err := c.podController.podLister.Pods(pod.Namespace).Get(pod.Name)
			if err != nil {
				return err
			}
			clone := origin.DeepCopy()
			SetPodCondition(clone, condition)
			_, updateErr := c.podController.client.CoreV1().Pods(clone.Namespace).UpdateStatus(context.TODO(), clone, metav1.UpdateOptions{})
			return updateErr
		})
		return err, 0
	}

	graceSeconds := 0
	if app.Spec.UpdateStrategy.Type == apps.RollingUpdateStatefulSetStrategyType {
		if app.Spec.UpdateStrategy.RollingUpdate != nil && app.Spec.UpdateStrategy.RollingUpdate.InPlaceUpdateStrategy != nil {
			graceSeconds = int(app.Spec.UpdateStrategy.RollingUpdate.InPlaceUpdateStrategy.GracePeriodSeconds)
		}
	}

	if graceSeconds > 0 {
		status := inplace.Status{}
		waitSeconds := 0
		if len(statusBytes) == 0 {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				origin, err := c.podController.podLister.Pods(pod.Namespace).Get(pod.Name)
				if err != nil {
					return err
				}
				clone := origin.DeepCopy()
				inPlaceStatus := inplace.Status{
					Revision:        updateRevision.Name,
					UpdateTimeStamp: metav1.Now(),
				}
				statusJson, _ := json.Marshal(inPlaceStatus)
				if clone.Annotations == nil {
					clone.Annotations = make(map[string]string)
				}
				clone.Annotations[inplace.StatusKey] = string(statusJson)

				_, err = c.podController.client.CoreV1().Pods(clone.Namespace).UpdateStatus(context.TODO(), clone, metav1.UpdateOptions{})
				return err
			})
			if err != nil {
				return err, 0
			}
			waitSeconds = graceSeconds
		} else {
			_ = json.Unmarshal(statusBytes, &status)
			waitSeconds = int(status.UpdateTimeStamp.Add(time.Duration(graceSeconds) * time.Second).Sub(time.Now()).Seconds())
		}

		if waitSeconds > 0 {
			// first time without status, or other time within the grace period
			klog.Infof("wait for %d seconds before in-place update pod %s", waitSeconds+waitSeconds%2, pod.Name)
			return nil, waitSeconds + waitSeconds%2
		}
	}

	// change img and update pod, record old img ids
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		origin, err := c.podController.podLister.Pods(pod.Namespace).Get(pod.Name)
		if err != nil {
			return err
		}
		clone := origin.DeepCopy()

		if clone.GetLabels() == nil {
			clone.SetLabels(make(map[string]string))
		}
		clone.GetLabels()[apps.ControllerRevisionHashLabelKey] = app.Status.UpdateRevision
		if clone.Annotations == nil {
			clone.Annotations = make(map[string]string)
		}

		oldContainerImgId := make(map[string]string)
		mapContainerSpec2Status := make(map[string]*corev1.ContainerStatus)

		for _, v := range clone.Status.ContainerStatuses {
			mapContainerSpec2Status[v.Name] = &v
		}

		for i := range clone.Spec.Containers {
			c := &clone.Spec.Containers[i]
			image2change, ok := newImage[c.Name]
			if ok {
				clone.Spec.Containers[i].Image = image2change
				oldContainerImgId[c.Name] = mapContainerSpec2Status[c.Name].ContainerID
			}
		}

		inPlaceStatus := inplace.Status{
			Revision:        updateRevision.Name,
			UpdateTimeStamp: metav1.Now(),
			OldImgIds:       oldContainerImgId,
		}
		statusJson, _ := json.Marshal(inPlaceStatus)
		clone.Annotations[inplace.StatusKey] = string(statusJson)

		_, err = c.podController.client.CoreV1().Pods(clone.Namespace).Update(context.TODO(), clone, metav1.UpdateOptions{})
		return err
	})
	return retryErr, 0
}

// CleanUpInPlaceUpdate clean the annotation and condition after IsInPlaceUpdateDone returns true
func (c *StateController) CleanUpInPlaceUpdate(pod *corev1.Pod) error {
	condition := corev1.PodCondition{
		Type:               InPlaceUpdateReady,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "InPlaceUpdateDone",
	}
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		origin, err := c.podController.podLister.Pods(pod.Namespace).Get(pod.Name)
		if err != nil {
			return err
		}
		clone := origin.DeepCopy()
		SetPodCondition(clone, condition)
		delete(clone.Annotations, inplace.StatusKey)
		_, updateErr := c.podController.client.CoreV1().Pods(clone.Namespace).UpdateStatus(context.TODO(), clone, metav1.UpdateOptions{})
		return updateErr
	})
	return err
}

func IsInPlaceUpdateDone(pod *corev1.Pod) (bool, error) {
	var status inplace.Status
	err := json.Unmarshal([]byte(pod.Annotations[inplace.StatusKey]), &status)
	if err != nil {
		return false, err
	}
	for _, v := range pod.Status.ContainerStatuses {
		if imgId, ok := status.OldImgIds[v.Name]; ok {
			if imgId == v.ImageID {
				return false, nil
			} else {
				if !v.Ready || v.State.Running == nil || v.State.Running.StartedAt.Time.Before(status.UpdateTimeStamp.Time) {
					return false, nil
				}
			}
		}
	}
	return true, nil
}

func GetTemplateFromRevision(revision *apps.ControllerRevision) (*corev1.PodTemplateSpec, error) {
	var patchObj *struct {
		Spec struct {
			Template corev1.PodTemplateSpec `json:"template"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(revision.Data.Raw, &patchObj); err != nil {
		return nil, err
	}
	return &patchObj.Spec.Template, nil
}
func InjectReadinessGate(pod *corev1.Pod) {
	for _, v := range pod.Spec.ReadinessGates {
		if v.ConditionType == InPlaceUpdateReady {
			return
		}
	}
	pod.Spec.ReadinessGates = append(pod.Spec.ReadinessGates, corev1.PodReadinessGate{ConditionType: InPlaceUpdateReady})
}

func ContainsReadinessGate(pod *corev1.Pod) bool {
	for _, v := range pod.Spec.ReadinessGates {
		if v.ConditionType == InPlaceUpdateReady {
			return true
		}
	}
	return false
}

func SetPodCondition(pod *corev1.Pod, condition corev1.PodCondition) {
	for i, c := range pod.Status.Conditions {
		if c.Type == condition.Type {
			if c.Status != condition.Status {
				pod.Status.Conditions[i] = condition
			}
			return
		}
	}
	pod.Status.Conditions = append(pod.Status.Conditions, condition)
}

func IfConditionFalse(pod *corev1.Pod, conditionType corev1.PodConditionType) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == conditionType && c.Status == corev1.ConditionFalse {
			return true
		}
	}
	return false
}
