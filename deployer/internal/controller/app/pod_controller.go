package app

import (
	"context"
	"fmt"
	unicore "github.com/mcyouyou/unicore/api/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	storagelisterv1 "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

type PodController struct {
	client    clientset.Interface
	podLister listerv1.PodLister
	pvcLister listerv1.PersistentVolumeClaimLister
	scLister  storagelisterv1.StorageClassLister
	recorder  record.EventRecorder
}

func NewPodController(client clientset.Interface, podLister listerv1.PodLister, pvcLister listerv1.PersistentVolumeClaimLister,
	scLister storagelisterv1.StorageClassLister, recorder record.EventRecorder) *PodController {
	return &PodController{
		client:    client,
		podLister: podLister,
		pvcLister: pvcLister,
		scLister:  scLister,
		recorder:  recorder,
	}
}

func (c *PodController) CreateStatefulPod(ctx context.Context, app *unicore.App, pod *v1.Pod) error {
	// create pvc before creating pod
	err := c.createPVC(app, pod)
	if err != nil {
		c.recordPodEvent("create", app, pod, err)
		return err
	}
	_, err = c.client.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		// this pod of its name has been created
		return err
	}
	c.recordPodEvent("create", app, pod, err)
	return err
}

func (c *PodController) UpdateStatefulPod(ctx context.Context, app *unicore.App, pod *v1.Pod) error {
	triedUpdate := false
	// use this retry func to update to avoid conflict
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		needUpdate := false
		// check if pod belong to app
		if !matchAppAndPod(app, pod) {
			updatePodIdentity(app, pod)
			needUpdate = true
		}
		// create pvc and update pod's volume if pod's volume dont match the app's requirement
		if matchAppPVC(app, pod) {
			updatePodVolume(app, pod)
			needUpdate = true
			if err := c.createPVC(app, pod); err != nil {
				c.recordPodEvent("update", app, pod, err)
				return err
			}
		}

		if needUpdate {
			triedUpdate = true
			_, err := c.client.CoreV1().Pods(pod.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
			if err != nil {
				// may be conflict, re-get the pod
				newPod, err2 := c.client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
				if err2 != nil {
					log.Errorf("get to-update pod %s err: %v", pod.Name, err2)
				} else {
					// use deep copy to avoid affecting cached pod data
					pod = newPod.DeepCopy()
				}
				return err
			}
		}
		return nil
	})
	if triedUpdate {
		c.recordPodEvent("update", app, pod, err)
	}
	return err
}

func (c *PodController) DeleteStatefulPod(ctx context.Context, app *unicore.App, pod *v1.Pod) error {
	err := c.client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	c.recordPodEvent("delete", app, pod, err)
	return err
}

// create app-specified pvcs for its pod, do nothing if all pvc's created
func (c *PodController) createPVC(app *unicore.App, pod *v1.Pod) error {
	for _, pvcTemplate := range getPVCFromApp(app, pod) {
		pvc, err := c.pvcLister.PersistentVolumeClaims(pvcTemplate.Namespace).Get(pvcTemplate.Name)
		if errors.IsNotFound(err) {
			_, err := c.client.CoreV1().PersistentVolumeClaims(pvcTemplate.Namespace).Create(context.TODO(),
				pvcTemplate, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("create pvc %s err:%v", pvcTemplate.Name, err)
			}
		} else if err != nil {
			return fmt.Errorf("get pvc %s err:%v", pvcTemplate.Name, err)
		} else if pvc.DeletionTimestamp != nil {
			// this pvc is set to be graceful deleted
			return fmt.Errorf("pvc %s is to be deleted", pvcTemplate.Name)
		}
	}
	return nil
}

func (c *PodController) recordPodEvent(verb string, app *unicore.App, pod *v1.Pod, err error) {
	if err == nil {
		reason := fmt.Sprintf("Succ %s", verb)
		msg := fmt.Sprintf("%s pod %s of app %s successful", verb, pod.Name, app.Name)
		c.recorder.Event(app, v1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", verb)
		msg := fmt.Sprintf("%s pod %s of app %s failed: %v", verb, pod.Name, app.Name, err)
		c.recorder.Event(app, v1.EventTypeWarning, reason, msg)
	}
}
