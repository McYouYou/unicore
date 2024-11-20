package app

import (
	"context"
	"fmt"
	unicore "github.com/mcyouyou/unicore/api/v1"
	"github.com/mcyouyou/unicore/internal/controller/requeue_duration"
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/controller/history"
	"k8s.io/utils/ptr"
	"math"
	"sort"
)

var (
	controllerKind = unicore.GroupVersion.WithKind("App")
)

type StateController struct {
	podController *PodController
	recorder      record.EventRecorder
	history       history.Interface
}

// NewStateController new a stateController to manage revision and state
func NewStateController(podController *PodController, recorder record.EventRecorder, history history.Interface) *StateController {
	return &StateController{
		podController: podController,
		recorder:      recorder,
		history:       history,
	}
}

func (c *StateController) UpdateApp(ctx context.Context, app *unicore.App, pods []*v1.Pod) error {
	err := c.updateApp(ctx, app, pods)

}

func (c *StateController) updateApp(ctx context.Context, app *unicore.App, pods []*v1.Pod) error {
	if app == nil {
		return fmt.Errorf("app is nil")
	}
	// do a copy since it may be modified
	app = app.DeepCopy()
	// list and sort revisions from apiserver
	selector, err := metav1.LabelSelectorAsSelector(app.Spec.Selector)
	if err != nil {
		return err
	}
	revisions, err := c.history.ListControllerRevisions(app, selector)
	if err != nil {
		return err
	}
	history.SortControllerRevisions(revisions)
	collisionCount := int32(0)
	if app.Status.CollisionCount != nil {
		collisionCount = *app.Status.CollisionCount
	}

	// create new revision from to-update app(marshaling)
	patch, err := getAppPatch(app)
	if err != nil {
		return err
	}
	nextRevisionId := int64(1)
	if len(revisions) > 0 {
		nextRevisionId = revisions[len(revisions)-1].Revision + 1
	}
	cr, err := history.NewControllerRevision(app, controllerKind, app.Spec.Template.Labels,
		runtime.RawExtension{Raw: patch}, nextRevisionId, &collisionCount)
	if err != nil {
		return err
	}
	if cr.ObjectMeta.Annotations == nil {
		cr.ObjectMeta.Annotations = make(map[string]string)
	}
	for k, v := range app.Annotations {
		cr.ObjectMeta.Annotations[k] = v
	}

	// check if equivalent revision exists, then decide whether to issue creating or updating(rolling back) to apiserver
	equalRevisions := history.FindEqualRevisions(revisions, cr)
	equalCnt := len(equalRevisions)
	if equalCnt != 0 {
		if history.EqualRevision(revisions[len(revisions)-1], equalRevisions[equalCnt-1]) {
			// newest revision is up to date, nothing changed
			cr = revisions[len(revisions)-1]
		} else {
			// same revision found in history, rollback to it by updating the old revision's RevisionID
			cr, err = c.history.UpdateControllerRevision(equalRevisions[equalCnt-1], cr.Revision)
			if err != nil {
				return err
			}
		}
	} else {
		// issue a creat req to apiserver, adding the collisionCount if hash collision
		cr, err = c.history.CreateControllerRevision(app, cr, &collisionCount)
		if err != nil {
			return err
		}
	}

	// try finding the corresponding revision to the app.status
	updateRevision := cr
	var currentRevision *apps.ControllerRevision
	for i := range revisions {
		if revisions[i].Name == app.Status.CurrentRevision {
			currentRevision = revisions[i]
			break
		}
	}
	if currentRevision == nil {
		currentRevision = updateRevision
	}

	// TODO: refresh update expectation

}

// core procedure of updating the status and pods
func (c *StateController) applyUpdate(ctx context.Context,
	app *unicore.App,
	currentRevision *apps.ControllerRevision,
	updateRevision *apps.ControllerRevision,
	collisionCount int32,
	pods []*v1.Pod,
	revisions *apps.ControllerRevision) (*unicore.AppStatus, error) {
	selector, err := metav1.LabelSelectorAsSelector(app.Spec.Selector)
	if err != nil {
		return app.Status.DeepCopy(), err
	}

	currentSet, err := ApplyRevision(app, currentRevision)
	if err != nil {
		return app.Status.DeepCopy(), err
	}
	updateSet, err := ApplyRevision(app, updateRevision)
	if err != nil {
		return app.Status.DeepCopy(), err
	}

	status := unicore.AppStatus{}
	status.CurrentRevision = currentRevision.Name
	status.UpdateRevision = updateRevision.Name
	status.ObservedGeneration = app.Generation
	status.CollisionCount = ptr.To[int32](collisionCount)
	status.LabelSelector = selector.String()
	minReadySeconds := getMinReadySeconds(app)
	// update App.Status
	updateStatus(&status, minReadySeconds, currentRevision, updateRevision, pods)

	// the new replica list to maintain
	replicaCnt := int(*app.Spec.Replicas)
	replicas := make([]*v1.Pod, replicaCnt)
	firstUnhealthyOrdinal := math.MaxInt32
	var firstUnhealthyPod *v1.Pod
	invalidPods := make([]*v1.Pod, 0)

	// filter invalid ordinal pod, fill the pod to replica list with their index
	burstable := app.Spec.PodManagementPolicy == apps.ParallelPodManagement
	for i := range pods {
		if _, ord := getPodAppNameAndOrdinal(pods[i]); ord < replicaCnt {
			replicas[ord] = pods[i]
		} else if ord >= 0 {
			invalidPods = append(invalidPods, pods[i])
		}
	}

	// create extra pod to fill the empty index of replica list
	for i := 0; i < replicaCnt; i++ {
		if replicas[i] == nil {
			replicas[i] = newVersionedPodForApp(currentSet, updateSet, currentRevision.Name, updateRevision.Name,
				i, replicas)
		}
	}
	// sort invalid pods by their ordinals, descending
	sort.Sort(descendingOrdinal(invalidPods))

	unhealthy := 0
	for i := range replicas {
		if replicas[i] == nil {
			continue
		}
		if !getPodReady(replicas[i]) || replicas[i].DeletionTimestamp != nil {
			unhealthy++
			if _, ord := getPodAppNameAndOrdinal(replicas[i]); ord < firstUnhealthyOrdinal {
				firstUnhealthyOrdinal = ord
				firstUnhealthyPod = replicas[i]
			}
		}
	}

	for i := len(invalidPods) - 1; i >= 0; i-- {
		if !getPodReady(invalidPods[i]) || invalidPods[i].DeletionTimestamp != nil {
			unhealthy++
			if _, ord := getPodAppNameAndOrdinal(invalidPods[i]); ord < firstUnhealthyOrdinal {
				firstUnhealthyOrdinal = ord
				firstUnhealthyPod = invalidPods[i]
			}
		}
	}

	if unhealthy > 0 {
		klog.V(4).Info("App has unhealthy pods", "app", klog.KObj(app), "unhealthyReplicas", unhealthy, "pod", klog.KObj(firstUnhealthyPod))
	}

	if app.DeletionTimestamp != nil {
		// app is being deleted
		return &status, nil
	}

	// check if all pods match spec now. if not burstable, update can't be applied
	allPodsMatch := true
	logger := klog.FromContext(ctx)
	var runErr error

	for i := range replicas {
		if replicas[i] == nil {
			continue
		}
		// pods in these two phase should be restarted
		if replicas[i].Status.Phase == v1.PodFailed || replicas[i].Status.Phase == v1.PodSucceeded {
			allPodsMatch = false
			if replicas[i].DeletionTimestamp == nil {
				if err := c.podController.DeleteStatefulPod(ctx, app, replicas[i]); err != nil {
					c.recorder.Eventf(app, v1.EventTypeWarning, "FailedDelete", "failed to delete pod %s: %v", replicas[i].Name, err)
					runErr = err
				}
			}
			break
		}

		// if pod not created, create one
		if replicas[i].Status.Phase == "" {
			allPodsMatch = false
			if err := c.podController.CreateStatefulPod(ctx, app, replicas[i]); err != nil {
				condition := apps.StatefulSetCondition{
					Type:    unicore.ConditionFailCreatePod,
					Status:  v1.ConditionTrue,
					Message: fmt.Sprintf("failed to create pod %s: %v", replicas[i].Name, err),
				}
				setAppCondition(&status, condition)
				runErr = err
				break
			}
			// if burstable not allowed, break to make a monotonic creation
			if !burstable {
				break
			}
		}

		if replicas[i].Status.Phase == v1.PodPending {
			allPodsMatch = false
			logger.V(4).Info("App is creating pvc for pending pod", "app", klog.KObj(app), klog.KObj(replicas[i]))
			if err := c.podController.createPVC(app, replicas[i]); err != nil {
				runErr = err
				break
			}
		}

		// no bursting: for terminating pod: wait until graceful exit
		if replicas[i].DeletionTimestamp != nil && !burstable {
			logger.V(4).Info("App is waiting for pod to terminate", "app", klog.KObj(app), "pod", klog.KObj(replicas[i]))
			allPodsMatch = false
			break
		}
		if replicas[i].DeletionTimestamp != nil && burstable {
			logger.V(4).Info("App pod is terminating, skip this loop", "app", klog.KObj(app), "pod", klog.KObj(replicas[i]))
			break
		}

		// TODO: Update InPlaceUpdateReady condition for pod

		isAvailable, checkInterval := getPodAvailableAndNextCheckInterval(replicas[i], minReadySeconds)
		if !burstable {
			// not burstable and a pod is unavailable, preceding procedures can't be done
			if !isAvailable {
				allPodsMatch = false
				if checkInterval > 0 {
					// check it next reconcile
					requeue_duration.Push(getAppKey(app), checkInterval)
					logger.V(4).Info("waiting for pod to be available after ready for minReadySeconds",
						"app", klog.KObj(app), "waitTime", checkInterval, "pod", klog.KObj(replicas[i]),
						"minReadySeconds", minReadySeconds)
				} else {
					logger.V(4).Info("waiting for pod to be available", "app", klog.KObj(app), "pod", klog.KObj(replicas[i]))
				}
				break
			}
		} else if !isAvailable {
			// burstable but unavailable
			logger.V(4).Info("app pod is unavailable, skip this loop", "app", klog.KObj(app), "pod", klog.KObj(replicas[i]))
			if checkInterval > 0 {
				// check it next reconcile
				requeue_duration.Push(getAppKey(app), checkInterval)
			}
			break
		}

		if matchAppAndPod(app, replicas[i]) && matchAppPVC(app, replicas[i]) {
			continue
		}

		// avoid changing shared cache
		replica := replicas[i].DeepCopy()
		// finally we can update the pod
		if err := c.podController.UpdateStatefulPod(ctx, app, replica); err != nil {
			condition := apps.StatefulSetCondition{
				Type:    unicore.ConditionFailUpdatePod,
				Status:  v1.ConditionTrue,
				Message: fmt.Sprintf("failed to update pod %s: %v", replicas[i].Name, err),
			}
			setAppCondition(&status, condition)
			runErr = err
			break
		}
	}

	if runErr != nil || !allPodsMatch {
		updateStatus(&status, minReadySeconds, currentRevision, updateRevision, replicas, invalidPods)
		return &status, runErr
	}

	// at this point, if not burstable, all pods of spec.Replicas are available.
	// then we can make deletion of invalid pods in a decreasing order, if all predecessors are ready
	shouldExit := false
	for i := range invalidPods {
		pod := invalidPods[i]
		if pod.DeletionTimestamp != nil {
			if !burstable {
				// if not burstable, exit and wait until pod terminated
				shouldExit = true
				logger.V(4).Info("waiting for pod to terminate before scaling down", "app",
					klog.KObj(app), "pod", klog.KObj(pod))
			}
		}
		// if it's not the first unhealthy pod, exit until predecessors are ready
		if burstable && !getPodReady(pod) && pod != firstUnhealthyPod {
			logger.V(4).Info("waiting for preceding pod to be ready before scaling down", "app",
				klog.KObj(app), "pod", klog.KObj(pod), "preceding-pod", klog.KObj(firstUnhealthyPod))
			shouldExit = true
			break
		}

		logger.V(4).Info("pod is terminating for scale down", "app", klog.KObj(app), "pod", klog.KObj(pod))
		runErr = c.podController.DeleteStatefulPod(ctx, app, replicas[i])
		if runErr != nil {
			c.recorder.Eventf(app, v1.EventTypeWarning, "FailedDelete", "failed to delete pod %s: %v", replicas[i].Name, runErr)
			shouldExit = true
			break
		}
	}

	updateStatus(&status, minReadySeconds, currentRevision, updateRevision, replicas, invalidPods)
	if shouldExit || runErr != nil {
		return &status, runErr
	}

	// for onDelete strategy, pods will only be updated when manually deleted
	if app.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType {
		return &status, nil
	}

}

// decide and create a pod for an app either from currentRevision or updateRevision
func newVersionedPodForApp(currentApp, updateApp *unicore.App, currentRevision, updateRevision string, ordinal int,
	replicas []*v1.Pod) *v1.Pod {
	if isCurrentRevisionExpected(currentApp, updateRevision, ordinal, replicas) {
		pod := newAppPod(currentApp, ordinal)
		if pod.Labels == nil {
			pod.Labels = make(map[string]string)
		}
		pod.Labels[apps.StatefulSetRevisionLabel] = currentRevision
		return pod
	}
	pod := newAppPod(updateApp, ordinal)
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[apps.StatefulSetRevisionLabel] = updateRevision
	return pod
}

// check if the given ordinal pod is expected to be at currentRevision instead of updateRevision
func isCurrentRevisionExpected(app *unicore.App, updateRevision string, ordinal int, replicas []*v1.Pod) bool {
	// use no rolling update and revisions, from spec data to create pods instead
	if app.Spec.UpdateStrategy.Type != apps.RollingUpdateStatefulSetStrategyType {
		return false
	}
	if app.Spec.UpdateStrategy.RollingUpdate == nil {
		{
			return ordinal < int(app.Status.CurrentReplicas)
		}
	}
	// use ordered update, pods ordinals in [:RollingUpdate.Partition) are expected to be at CurrentRevision
	if !app.Spec.UpdateStrategy.RollingUpdate.UnorderedUpdate {
		return ordinal < int(*app.Spec.UpdateStrategy.RollingUpdate.Partition)
	}
	// if unordered strategy is set, Partition is the expected amount of pods at CurrentRevision
	var currentRevisionCnt int
	for i, pod := range replicas {
		if pod == nil || i == ordinal {
			continue
		}
		if pod.GetLabels()[apps.ControllerRevisionHashLabelKey] != updateRevision {
			currentRevisionCnt++
		}
	}
	return currentRevisionCnt < int(*app.Spec.UpdateStrategy.RollingUpdate.Partition)
}

func getAppConditionFromType(status *unicore.AppStatus, condType apps.StatefulSetConditionType) *apps.StatefulSetCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// update status.Conditions with given condition
func setAppCondition(status *unicore.AppStatus, condition apps.StatefulSetCondition) {
	currentCondition := getAppConditionFromType(status, condition.Type)
	if currentCondition != nil && currentCondition.Status == condition.Status && currentCondition.Reason == condition.Reason {
		return
	}
	if currentCondition != nil && currentCondition.Status == condition.Status {
		condition.LastTransitionTime = currentCondition.LastTransitionTime
	}

	// filter out all items with the same condition type
	newConditions := make([]apps.StatefulSetCondition, 0)
	for _, c := range status.Conditions {
		if c.Type != condition.Type {
			newConditions = append(newConditions, c)
		}
	}
	status.Conditions = append(newConditions, condition)
}
