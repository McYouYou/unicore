package app

import (
	"encoding/json"
	"fmt"
	unicore "github.com/mcyouyou/unicore/api/deployer/v1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/controller"
	"strconv"
	"strings"
	"time"
)

// labels
const (
	LabelPodOwnerApp = "unicore.mcyou.cn/app-name"
	LabelPodOrdinal  = "unicore.mcyou.cn/pod-ordinal"
)

var patchCodec = scheme.Codecs.LegacyCodec(unicore.SchemeGroupVersion)

// get pvc to-create from app.Spec.VolumeClaimTemplates
func getPVCFromApp(app *unicore.App, pod *v1.Pod) map[string]*v1.PersistentVolumeClaim {
	_, ordinal := GetPodAppNameAndOrdinal(pod)
	pvcs := make(map[string]*v1.PersistentVolumeClaim, len(app.Spec.VolumeClaimTemplates))
	for i := range app.Spec.VolumeClaimTemplates {
		// set pvc name as pvc-app-ordinal
		claim := app.Spec.VolumeClaimTemplates[i]
		claim.Name = getPVCOutName(app, &claim, ordinal)
		claim.Namespace = app.Namespace
		claim.Labels = app.Spec.Selector.MatchLabels
		pvcs[app.Spec.VolumeClaimTemplates[i].Name] = &claim
	}
	return pvcs
}

// get pod's app name and the pod's ordinal from a real pod
func GetPodAppNameAndOrdinal(pod *v1.Pod) (string, int) {
	parts := strings.Split(pod.Name, "-")
	if len(parts) < 2 {
		return "", -1
	}
	owner := strings.Join(parts[:len(parts)-1], "-")
	if i, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
		return owner, i
	}
	return "", -1
}

// get pvc's to-create name for given ordinal
func getPVCOutName(app *unicore.App, pvc *v1.PersistentVolumeClaim, ordinal int) string {
	return fmt.Sprintf("%s-%s-%d", app.Name, pvc.Name, ordinal)
}

func getPodOutName(app *unicore.App, ordinal int) string {
	return fmt.Sprintf("%s-%d", app.Name, ordinal)
}

// check if the pod is a member of app
func matchAppAndPod(app *unicore.App, pod *v1.Pod) bool {
	appName, ordinal := GetPodAppNameAndOrdinal(pod)
	if ordinal < 0 {
		return false
	}
	return app.Name == appName && pod.Name == getPodOutName(app, ordinal) && pod.Namespace == app.Namespace &&
		pod.Labels[LabelPodOwnerApp] == app.Name
}

// set pod's identity to match app
func updatePodIdentity(app *unicore.App, pod *v1.Pod) {
	_, ordinal := GetPodAppNameAndOrdinal(pod)
	pod.Name = getPodOutName(app, ordinal)
	pod.Namespace = app.Namespace
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[LabelPodOwnerApp] = app.Name
	pod.Labels[LabelPodOrdinal] = strconv.Itoa(ordinal)
}

// check if the pod's current volumes match the app's requirement
func matchAppPVC(app *unicore.App, pod *v1.Pod) bool {
	_, ordinal := GetPodAppNameAndOrdinal(pod)
	if ordinal < 0 {
		return false
	}
	podVolumes := make(map[string]v1.Volume, len(pod.Spec.Volumes))
	for _, vol := range pod.Spec.Volumes {
		podVolumes[vol.Name] = vol
	}
	for _, pvc := range app.Spec.VolumeClaimTemplates {
		volume, ok := podVolumes[pvc.Name]
		if !ok || volume.VolumeSource.PersistentVolumeClaim == nil ||
			volume.VolumeSource.PersistentVolumeClaim.ClaimName != getPVCOutName(app, &pvc, ordinal) {
			return false
		}
	}
	return true
}

// update pod volumes to match the app. pod volume with the same name will be overwritten
func updatePodVolume(app *unicore.App, pod *v1.Pod) {
	toCreatePVC := getPVCFromApp(app, pod)
	newVolumes := make([]v1.Volume, 0, len(toCreatePVC))
	for name, pvc := range toCreatePVC {
		newVolumes = append(newVolumes, v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: pvc.Name, ReadOnly: false},
			},
		})
	}
	for _, v := range pod.Spec.Volumes {
		if _, ok := toCreatePVC[v.Name]; !ok {
			newVolumes = append(newVolumes, v)
		}
	}
	pod.Spec.Volumes = newVolumes
}

// return json marshaled patch of app.spec.podSpecTemplate
func getAppPatch(app *unicore.App) ([]byte, error) {
	str, err := runtime.Encode(patchCodec, app)
	if err != nil {
		klog.Info("err:" + err.Error())
		return nil, err
	}
	var raw map[string]interface{}
	err = json.Unmarshal(str, &raw)
	if err != nil {
		return nil, err
	}
	objCopy := make(map[string]interface{})
	specCopy := make(map[string]interface{})
	spec := raw["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	specCopy["template"] = template
	template["$patch"] = "replace"
	objCopy["spec"] = specCopy
	patch, err := json.Marshal(objCopy)
	return patch, err
}

// ApplyRevision merge the origin app and the revision's data to create a new one
func ApplyRevision(app *unicore.App, revision *apps.ControllerRevision) (*unicore.App, error) {
	clone := app.DeepCopy()
	patched, err := strategicpatch.StrategicMergePatch([]byte(runtime.EncodeOrDie(patchCodec, clone)), revision.Data.Raw, clone)
	if err != nil {
		return nil, err
	}
	restoredApp := &unicore.App{}
	if err = json.Unmarshal(patched, restoredApp); err != nil {
		return nil, err
	}
	return restoredApp, nil
}

// getMinReadySeconds returns the minReadySeconds set in the rollingUpdate, default is 0
func getMinReadySeconds(app *unicore.App) int32 {
	if app.Spec.UpdateStrategy.RollingUpdate == nil ||
		app.Spec.UpdateStrategy.RollingUpdate.MinReadySeconds == nil {
		return 0
	}
	return *app.Spec.UpdateStrategy.RollingUpdate.MinReadySeconds
}

func getPodReady(pod *v1.Pod) bool {
	if pod == nil || pod.Status.Conditions == nil {
		return false
	}
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == v1.PodReady {
			return true
		}
	}
	return false
}

func getPodReadyCondition(pod *v1.Pod) *v1.PodCondition {
	if pod == nil {
		return nil
	}
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == v1.PodReady {
			return &pod.Status.Conditions[i]
		}
	}
	return nil
}

// return true if pod has been ready for the specified seconds
func getPodAvailable(pod *v1.Pod, minReadySeconds int32) bool {
	readyCondition := getPodReadyCondition(pod)
	if minReadySeconds == 0 || readyCondition == nil || readyCondition.LastTransitionTime.IsZero() {
		return false
	}
	if readyCondition.LastTransitionTime.Time.Add(time.Duration(minReadySeconds) * time.Second).Before(time.Now()) {
		return true
	}
	return false
}

// get pod's revision by label
func getPodRevision(pod *v1.Pod) string {
	if pod.Labels == nil {
		return ""
	}
	return pod.Labels[apps.StatefulSetRevisionLabel]
}

// creat pod for app from its template
func newAppPod(app *unicore.App, ordinal int) *v1.Pod {
	pod, _ := controller.GetPodFromTemplate(&app.Spec.Template, app, metav1.NewControllerRef(app, controllerKind))
	pod.Name = getPodOutName(app, ordinal)
	updatePodIdentity(app, pod)
	pod.Spec.Hostname = pod.Name
	pod.Spec.Subdomain = app.Spec.ServiceName
	updateVolume(app, pod)
	return pod
}

// update pod's volume spec to match with its template pvc
func updateVolume(app *unicore.App, pod *v1.Pod) {
	currentVolumes := pod.Spec.Volumes
	claims := getPVCFromApp(app, pod)
	newVolumes := make([]v1.Volume, 0, len(claims))
	for name, claim := range claims {
		newVolumes = append(newVolumes, v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: claim.Name, ReadOnly: false},
			},
		})
	}
	for i := range currentVolumes {
		if _, ok := claims[currentVolumes[i].Name]; !ok {
			newVolumes = append(newVolumes, currentVolumes[i])
		}
	}
	pod.Spec.Volumes = newVolumes
}

type descendingOrdinal []*v1.Pod

func (do descendingOrdinal) Len() int {
	return len(do)
}

func (do descendingOrdinal) Swap(i, j int) {
	do[i], do[j] = do[j], do[i]
}

func (do descendingOrdinal) Less(i, j int) bool {
	_, o1 := GetPodAppNameAndOrdinal(do[i])
	_, o2 := GetPodAppNameAndOrdinal(do[j])
	return o1 > o2
}

// check if pod is ready and available. if it's ready but unavailable, returns a time that we should retry after witch
func getPodAvailableAndNextCheckInterval(pod *v1.Pod, minReadySeconds int32) (bool, time.Duration) {
	if pod.Status.Phase != v1.PodRunning || !getPodReady(pod) {
		return false, 0
	}
	c := getPodReadyCondition(pod)
	minReadyDuration := time.Duration(minReadySeconds) * time.Second
	if minReadyDuration == 0 {
		return true, 0
	}
	if c.LastTransitionTime.IsZero() {
		return false, minReadyDuration
	}
	interval := c.LastTransitionTime.Time.Add(minReadyDuration).Sub(time.Now())
	if interval > 0 {
		return false, interval
	}
	return true, 0
}

func GetAppKey(app *unicore.App) string {
	return app.ObjectMeta.GetNamespace() + "/" + app.ObjectMeta.GetName()
}
