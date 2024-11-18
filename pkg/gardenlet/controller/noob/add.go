package noob

import (
	"context"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "noob"

func (r *Reconciler) AddToManager(mgr manager.Manager) error {
	return builder.
		ControllerManagedBy(mgr).
		Named(ControllerName).
		Watches(&corev1.ConfigMap{}, &handler.Funcs{
			DeleteFunc: func(_ context.Context, event event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				name := event.Object.GetName()
				if !strings.HasPrefix(name, "shoot-protocol--") {
					return
				}

				q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      event.Object.GetName(),
					Namespace: event.Object.GetNamespace(),
				}})
			},
		}).
		WatchesRawSource(
			source.Kind[client.Object](r.GardenClusterCache,
				&gardencorev1beta1.Shoot{}, &handler.EnqueueRequestForObject{}),
		).
		Complete(r)
}
