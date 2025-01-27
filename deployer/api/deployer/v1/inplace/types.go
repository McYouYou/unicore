package inplace

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Strategy defines the strategies for in-place update.
type Strategy struct {
	// GracePeriodSeconds is the timespan between set Pod status to not-ready and update images in Pod spec
	// when in-place update a Pod.
	GracePeriodSeconds int32 `json:"gracePeriodSeconds,omitempty"`
}

const StatusKey string = "unicore.mcyou.cn/inplace-update-status"

type Status struct {
	Revision        string            `json:"revision,omitempty"`
	UpdateTimeStamp metav1.Time       `json:"updateTimeStamp,omitempty"`
	OldImgIds       map[string]string `json:"oldImgIds,omitempty"`
}
