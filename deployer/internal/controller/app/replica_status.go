package app

import (
	unicore "github.com/mcyouyou/unicore/api/deployer/v1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

type replicaStatus struct {
	replicas          int32
	readyReplicas     int32
	availableReplicas int32
	currentReplicas   int32

	updatedReplicas          int32
	updatedReadyReplicas     int32
	updatedAvailableReplicas int32
}

func updateStatus(status *unicore.AppStatus, minReadySeconds int32, currentRevision, updateRevision *apps.ControllerRevision,
	podLists ...[]*v1.Pod) {
	status.Replicas = 0
	status.ReadyReplicas = 0
	status.AvailableReplicas = 0
	status.CurrentReplicas = 0
	status.UpdatedReplicas = 0
	status.UpdatedAvailableReplicas = 0
	for _, list := range podLists {
		replicaStatus := calcReplicaStatus(list, minReadySeconds, currentRevision, updateRevision)
		status.Replicas += replicaStatus.replicas
		status.ReadyReplicas += replicaStatus.readyReplicas
		status.AvailableReplicas += replicaStatus.availableReplicas
		status.UpdatedAvailableReplicas += replicaStatus.updatedAvailableReplicas
		status.UpdatedReadyReplicas += replicaStatus.updatedReadyReplicas
		status.UpdatedReplicas += replicaStatus.updatedReplicas
		status.CurrentReplicas += replicaStatus.currentReplicas
	}
}

func calcReplicaStatus(pods []*v1.Pod, minReadySeconds int32, currentRevision, updateRevision *apps.ControllerRevision) replicaStatus {
	ret := replicaStatus{}
	for _, pod := range pods {
		if pod == nil {
			continue
		}

		if pod.Status.Phase != "" {
			ret.replicas++
		}

		if pod.Status.Phase == v1.PodRunning && getPodReady(pod) {
			ret.readyReplicas++
			if pod.Labels != nil && pod.Labels[apps.StatefulSetRevisionLabel] == updateRevision.Name {
				ret.updatedReadyReplicas++
				if getPodAvailable(pod, minReadySeconds) {
					ret.updatedAvailableReplicas++
				}
			}
			if getPodAvailable(pod, minReadySeconds) {
				ret.availableReplicas++
			}
		}
		// count the number of current and update replicas
		if pod.Status.Phase != "" && pod.DeletionTimestamp == nil {
			revision := getPodRevision(pod)
			if revision == currentRevision.Name {
				ret.currentReplicas++
			}
			if revision == updateRevision.Name {
				ret.updatedReplicas++
			}
		}
	}
	return ret
}
