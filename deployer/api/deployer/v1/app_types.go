/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"github.com/mcyouyou/unicore/api/deployer/v1/inplace"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AppSpec defines the desired state of App
type AppSpec struct {
	// desired number of pods, defaults to 1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// the template of the App's pod to create
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Template v1.PodTemplateSpec `json:"template"`

	// selects the label that App Controller managed, should match Template
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Selector *metav1.LabelSelector `json:"selector"`

	// PVC's template, at least one should match Template's volume mnt
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	VolumeClaimTemplates []v1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

	// updateStrategy indicates the AppUpdateStrategy that will be
	// employed to update Pods in the App when a revision is made to
	// Template. Behave the same as sts.
	UpdateStrategy AppUpdateStrategy `json:"updateStrategy,omitempty"`

	// serviceName that app controller to create to manage the pods
	ServiceName string `json:"serviceName,omitempty"`

	// the maximum of history records of the template that app controller keeps
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit,omitempty"`

	// same as sts. if parallel strategy is set, pod will be managed ignoring the ordinal. if orderedReady is set,
	// pod will be scale/update considering ordinal and wait for former pod's ready status
	PodManagementPolicy apps.PodManagementPolicyType `json:"podManagementPolicy,omitempty"`

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// AppStatus defines the observed state of App
type AppStatus struct {

	// specify the current version of app's spec, comparing to object meta to check if update needed
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// replicas is the number of Pods created by the App controller.
	Replicas int32 `json:"replicas"`

	// readyReplicas is the number of Pods created by the App controller that have a Ready Condition.
	ReadyReplicas int32 `json:"readyReplicas"`

	// AvailableReplicas is the number of Pods created by the App controller that have been ready for
	//minReadySeconds.
	AvailableReplicas int32 `json:"availableReplicas"`

	// currentReplicas is the number of Pods created by the App controller from the App version
	// indicated by currentRevision.
	CurrentReplicas int32 `json:"currentReplicas"`

	// updatedReplicas is the number of Pods created by the App controller from the App version
	// indicated by updateRevision.
	UpdatedReplicas int32 `json:"updatingReplicas"`

	// updatedReadyReplicas is the number of updated Pods created by the StatefulSet controller that have a Ready Condition.
	UpdatedReadyReplicas int32 `json:"updatedReadyReplicas,omitempty"`

	// updatedAvailableReplicas is the number of updated Pods created by the StatefulSet controller that have a Ready condition
	//for atleast minReadySeconds.
	UpdatedAvailableReplicas int32 `json:"updatedAvailableReplicas,omitempty"`

	// currentRevision, if not empty, indicates the version of the App used to generate Pods in the
	// sequence [0,currentReplicas).
	CurrentRevision string `json:"currentRevision"`

	// updateRevision, if not empty, indicates the version of the App used to generate Pods in the sequence
	// [replicas-updatedReplicas,replicas)
	UpdateRevision string `json:"updatingRevision,omitempty"`

	// Represents the latest available observations of a statefulset's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []apps.StatefulSetCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// LabelSelector is label selectors for query over pods that should match the replica count used by HPA.
	LabelSelector string `json:"labelSelector,omitempty"`

	// used to ensure a unique revision hash
	CollisionCount *int32 `json:"collisionCount,omitempty"`

	// VolumeClaims represents the status of compatibility between existing PVCs
	// and their respective templates. It tracks whether the PersistentVolumeClaims have been updated
	// to match any changes made to the volumeClaimTemplates, ensuring synchronization
	// between the defined templates and the actual PersistentVolumeClaims in use.
	VolumeClaims []VolumeClaimStatus `json:"volumeClaims,omitempty"`

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient

// App is the Schema for the apps API
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}

// AppUpdateStrategy indicates the strategy that the App
// controller will use to perform updates. It includes any additional parameters
// necessary to perform the update for the indicated strategy.
type AppUpdateStrategy struct {
	// Type indicates the type of the AppUpdateStrategy, which is as same as the sts.
	// Default is RollingUpdate.
	// +optional
	Type apps.StatefulSetUpdateStrategyType `json:"type,omitempty"`
	// RollingUpdate is used to communicate parameters when Type is RollingUpdateStatefulSetStrategyType.
	// +optional
	RollingUpdate *RollingUpdateAppStrategy `json:"rollingUpdate,omitempty"`
}

type RollingUpdateAppStrategy struct {
	// Partition indicates the ordinal at which the App's pods should be partitioned by default,
	// which means pod whose ordinal >= Partition should be updated
	// But if unorderedUpdate has been set:
	//   - Partition indicates the number of pods with non-updated revisions when rolling update.
	//   - It means controller will update $(replicas - partition) number of pod.
	// Default value is 0.
	// +optional
	Partition *int32 `json:"partition,omitempty"`
	// The maximum number of pods that can be unavailable during the update.
	// Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
	// Absolute number is calculated from percentage by rounding down.
	// Also, maxUnavailable can just be allowed to work with Parallel podManagementPolicy.
	// Defaults to 1.
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
	// PodUpdatePolicy indicates how pods should be updated
	// Default value is "ReCreate"
	// +optional
	PodUpdatePolicy PodUpdateStrategyType `json:"podUpdatePolicy,omitempty"`
	// Paused indicates that the StatefulSet is paused.
	// Default value is false
	// +optional
	Paused bool `json:"paused,omitempty"`
	// UnorderedUpdate contains strategies for non-ordered update.
	// If it is not nil, pods will be updated with non-ordered sequence.
	// Noted that UnorderedUpdate can only be allowed to work with Parallel podManagementPolicy
	// +optional
	UnorderedUpdate bool `json:"unorderedUpdate,omitempty"`
	// InPlaceUpdateStrategy contains strategies for in-place update.
	// +optional
	InPlaceUpdateStrategy *inplace.InPlaceUpdateStrategy `json:"inPlaceUpdateStrategy,omitempty"`
	// MinReadySeconds indicates how long will the pod be considered ready after it's updated.
	// MinReadySeconds works with both OrderedReady and Parallel podManagementPolicy.
	// It affects the pod scale up speed when the podManagementPolicy is set to be OrderedReady.
	// Combined with MaxUnavailable, it affects the pod update speed regardless of podManagementPolicy.
	// Default value is 0, max is 300.
	// +optional
	MinReadySeconds *int32 `json:"minReadySeconds,omitempty"`
}

// PodUpdateStrategyType is a string enumeration type that enumerates
// all possible ways we can update a Pod when updating application
type PodUpdateStrategyType string

const (
	// RecreatePodUpdateStrategyType indicates that we always delete Pod and create new Pod
	// during Pod update, which is the default behavior
	RecreatePodUpdateStrategyType PodUpdateStrategyType = "ReCreate"
	// InPlaceIfPossiblePodUpdateStrategyType indicates that we try to in-place update Pod instead of
	// recreating Pod when possible. Currently, only image update of pod spec is allowed. Any other changes to the pod
	// spec will fall back to ReCreate PodUpdateStrategyType where pod will be recreated.
	InPlaceIfPossiblePodUpdateStrategyType PodUpdateStrategyType = "InPlaceIfPossible"
	// InPlaceOnlyPodUpdateStrategyType indicates that we will in-place update Pod instead of
	// recreating pod. We only allow image update for pod spec. Any other changes to the pod spec will be
	// rejected by kube-apiserver
	InPlaceOnlyPodUpdateStrategyType PodUpdateStrategyType = "InPlaceOnly"
)

// VolumeClaimStatus describes the status of a volume claim template.
// It provides details about the compatibility and readiness of the volume claim.
type VolumeClaimStatus struct {
	// VolumeClaimName is the name of the volume claim.
	// This is a unique identifier used to reference a specific volume claim.
	VolumeClaimName string `json:"volumeClaimName"`
	// CompatibleReplicas is the number of replicas currently compatible with the volume claim.
	// It indicates how many replicas can function properly, being compatible with this volume claim.
	// Compatibility is determined by whether the PVC spec storage requests are greater than or equal to the template spec storage requests
	CompatibleReplicas int32 `json:"compatibleReplicas"`
	// CompatibleReadyReplicas is the number of replicas that are both ready and compatible with the volume claim.
	// It highlights that these replicas are not only compatible but also ready to be put into service immediately.
	// Compatibility is determined by whether the pvc spec storage requests are greater than or equal to the template spec storage requests
	// The "ready" status is determined by whether the PVC status capacity is greater than or equal to the PVC spec storage requests.
	CompatibleReadyReplicas int32 `json:"compatibleReadyReplicas"`
}

// conditions of app
const (
	ConditionFailCreatePod apps.StatefulSetConditionType = "FailedCreatePod"
	ConditionFailUpdatePod apps.StatefulSetConditionType = "FailedUpdatePod"
)
