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

package controller

import (
	"context"
	"fmt"
	unicore "github.com/mcyouyou/unicore/api/deployer/v1"
	unicoreApp "github.com/mcyouyou/unicore/internal/controller/app"
	"github.com/mcyouyou/unicore/internal/controller/requeue_duration"
	clientset "github.com/mcyouyou/unicore/pkg/generated/clientset/versioned"
	lister "github.com/mcyouyou/unicore/pkg/generated/listers/deployer/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	kubecontroller "k8s.io/kubernetes/pkg/controller"
	"k8s.io/kubernetes/pkg/controller/history"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var controllerKind = unicore.SchemeGroupVersion.WithKind("App")

// AppReconciler reconciles a App object
type AppReconciler struct {
	Scheme          *runtime.Scheme
	Lister          lister.AppLister
	PodController   *unicoreApp.PodController
	StateController *unicoreApp.StateController
	UnicoreCli      *clientset.Clientset
	PodLister       corelisters.PodLister
	KubePodControl  kubecontroller.PodControlInterface
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=unicore.mcyou.cn,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=unicore.mcyou.cn,resources=apps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=unicore.mcyou.cn,resources=apps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the App object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	_ = log.FromContext(ctx)

	startTime := time.Now()
	defer func() {
		if retErr == nil {
			if res.Requeue || res.RequeueAfter > 0 {
				klog.InfoS("finished syncing App", "app", req, "timeCost", time.Since(startTime), "result", res)
			} else {
				klog.InfoS("finished syncing App", "app", req, "timeCost", time.Since(startTime))
			}
		} else {
			klog.ErrorS(retErr, "failed syncing App", "app", req, "timeCost", time.Since(startTime))
		}
	}()

	app, err := r.Lister.Apps(req.Namespace).Get(req.Name)
	if errors.IsNotFound(err) {
		klog.InfoS("app deleted", "app", req, "timeCost", time.Since(startTime))
		return reconcile.Result{}, nil
	}
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("get app err:%v", err))
		return reconcile.Result{}, err
	}

	selector, err := metav1.LabelSelectorAsSelector(app.Spec.Selector)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("parse selector err:%v", err))
		// no need to retry this
		return reconcile.Result{}, nil
	}

	if err := r.adoptOrphanRevisions(app); err != nil {
		return reconcile.Result{}, err
	}

	pods, err := r.getAppPods(ctx, app, selector)
	if err != nil {
		return reconcile.Result{}, err
	}
	if err := r.StateController.UpdateApp(ctx, app, pods); err != nil {
		return reconcile.Result{RequeueAfter: requeue_duration.Pop(unicoreApp.GetAppKey(app))}, err
	}
	klog.V(4).InfoS("sync app succeeded", "app", req, "timeCost", time.Since(startTime))
	return ctrl.Result{}, nil
}

// get the pods that the app should manage
func (r *AppReconciler) getAppPods(ctx context.Context, app *unicore.App, selector labels.Selector) ([]*corev1.Pod, error) {
	pods, err := r.PodLister.Pods(app.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing Pods
	canAdoptFunc := kubecontroller.RecheckDeletionTimestamp(func(ctx context.Context) (metav1.Object, error) {
		fresh, err := r.UnicoreCli.UnicoreV1().Apps(app.Namespace).Get(ctx, app.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.UID != app.UID {
			return nil, fmt.Errorf("original App %v/%v is gone: got uid %v, wanted %v", app.Namespace, app.Name, fresh.UID, app.UID)
		}
		return fresh, nil
	})

	cm := kubecontroller.NewPodControllerRefManager(r.KubePodControl, app, selector, controllerKind, canAdoptFunc)

	filter := func(pod *corev1.Pod) bool {
		owner, _ := unicoreApp.GetPodAppNameAndOrdinal(pod)
		return owner == app.Name
	}
	return cm.ClaimPods(ctx, pods, filter)
}

// list revisions whose controller in metadata is nil, and adopt them if UID match the app
func (r *AppReconciler) adoptOrphanRevisions(app *unicore.App) error {
	revisions, err := r.StateController.ListRevisions(app)
	if err != nil {
		return err
	}
	orphans := make([]*appsv1.ControllerRevision, 0)
	for i := range revisions {
		if metav1.GetControllerOf(revisions[i]) == nil {
			orphans = append(orphans, revisions[i])
		}
	}
	if len(orphans) > 0 {
		fresh, err := r.UnicoreCli.UnicoreV1().Apps(app.Namespace).Get(context.TODO(), app.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if fresh.UID != app.UID {
			return fmt.Errorf("original app %s/%s is refreshed, uid not match", app.Namespace, app.Name)
		}
		return r.StateController.AdoptOrphanRevisions(app, orphans)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// watch app and pods
	return ctrl.NewControllerManagedBy(mgr).
		For(&unicore.App{}).
		Complete(r)
}

func NewAppReconciler(mgr ctrl.Manager) (*AppReconciler, error) {
	cache := mgr.GetCache()
	appInformer, err := cache.GetInformerForKind(context.TODO(), controllerKind)
	if err != nil {
		return nil, err
	}
	podInformer, err := cache.GetInformerForKind(context.TODO(), corev1.SchemeGroupVersion.WithKind("Pod"))
	if err != nil {
		return nil, err
	}
	pvcInformer, err := cache.GetInformerForKind(context.TODO(), corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))
	if err != nil {
		return nil, err
	}
	scInformer, err := cache.GetInformerForKind(context.TODO(), storagev1.SchemeGroupVersion.WithKind("StorageClass"))
	if err != nil {
		return nil, err
	}
	revInformer, err := cache.GetInformerForKind(context.TODO(), appsv1.SchemeGroupVersion.WithKind("ControllerRevision"))
	if err != nil {
		return nil, err
	}
	appLister := lister.NewAppLister(appInformer.(toolscache.SharedIndexInformer).GetIndexer())
	podLister := corelisters.NewPodLister(podInformer.(toolscache.SharedIndexInformer).GetIndexer())
	pvcLister := corelisters.NewPersistentVolumeClaimLister(pvcInformer.(toolscache.SharedIndexInformer).GetIndexer())
	scLister := storagelisters.NewStorageClassLister(scInformer.(toolscache.SharedIndexInformer).GetIndexer())

	unicoreCli, err := clientset.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}
	kubeCli, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedv1.EventSinkImpl{Interface: kubeCli.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "app-controller"})

	podController := unicoreApp.NewPodController(kubeCli, podLister, pvcLister, scLister, recorder)

	return &AppReconciler{
		Scheme:        mgr.GetScheme(),
		Lister:        appLister,
		PodController: podController,
		StateController: unicoreApp.NewStateController(podController, recorder,
			history.NewHistory(kubeCli, appslisters.NewControllerRevisionLister(revInformer.(toolscache.SharedIndexInformer).GetIndexer()))),
		UnicoreCli:     unicoreCli,
		PodLister:      podLister,
		KubePodControl: kubecontroller.RealPodControl{KubeClient: kubeCli, Recorder: recorder},
	}, nil

}
