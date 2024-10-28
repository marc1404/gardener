package noob

import (
	"context"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
)

const ControllerName = "noob"

func AddToManager(mgr manager.Manager, seedCluster, gardenCluster cluster.Cluster) error {
	r := &Reconciler{
		SeedClient:   seedCluster.GetClient(),
		GardenClient: gardenCluster.GetClient(),
	}

	return builder.
		ControllerManagedBy(mgr).
		Named(ControllerName).
		Watches(&corev1.ConfigMap{}, &handler.Funcs{
			DeleteFunc: func(_ context.Context, event event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				name := event.Object.GetName()
				if !strings.HasPrefix(name, "shoot-protocol--") {
					return
				}

				enqueueReconcileRequest(q, event.Object.GetName(), event.Object.GetNamespace())
			},
		}).
		WatchesRawSource(
			source.Kind[client.Object](gardenCluster.GetCache(),
				&gardencorev1beta1.Shoot{}, watchHandler()),
		).
		Complete(r)
}

func watchHandler() handler.EventHandler {
	return &handler.Funcs{
		CreateFunc: onCreate,
		UpdateFunc: onUpdate,
		DeleteFunc: onDelete,
	}
}

func onCreate(_ context.Context, event event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	enqueueReconcileRequest(q, event.Object.GetName(), event.Object.GetNamespace())
}

func onUpdate(_ context.Context, event event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	enqueueReconcileRequest(q, event.ObjectNew.GetName(), event.ObjectNew.GetNamespace())
}

func onDelete(_ context.Context, event event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	enqueueReconcileRequest(q, event.Object.GetName(), event.Object.GetNamespace())
}

func enqueueReconcileRequest(q workqueue.TypedRateLimitingInterface[reconcile.Request], objectName, objectNamespace string) {
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      objectName,
		Namespace: objectNamespace,
	}})
}
