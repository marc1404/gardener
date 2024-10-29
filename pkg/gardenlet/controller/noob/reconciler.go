package noob

import (
	"context"
	"fmt"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

type Reconciler struct {
	SeedClient         client.Client
	GardenClient       client.Client
	GardenClusterCache cache.Cache
	GardenNamespace    string
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("reconciling from the noob controller!")

	shoot, err := r.resolveShoot(ctx, request)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error resolving shoot: %w", err)
	}
	if shoot == nil {
		return reconcile.Result{}, nil
	}

	configMap := &corev1.ConfigMap{}
	configMapName := types.NamespacedName{
		Namespace: r.GardenNamespace,
		Name:      fmt.Sprintf("shoot-protocol--%v--%v", shoot.Namespace, shoot.Name),
	}
	configMapIsFound := true
	if err := r.SeedClient.Get(ctx, configMapName, configMap); err != nil {
		if apierrors.IsNotFound(err) {
			configMapIsFound = false
		} else {
			return reconcile.Result{}, fmt.Errorf("error retrieving configmap from store: %w", err)
		}
	}

	if !configMapIsFound {
		configMap.Name = configMapName.Name
		configMap.Namespace = configMapName.Namespace
		configMap.Data = map[string]string{
			"resourceVersion": shoot.ResourceVersion,
		}

		if err := r.SeedClient.Create(ctx, configMap); err != nil {
			return reconcile.Result{}, fmt.Errorf("error creating configmap: %w", err)
		}
	}

	configMap.Data["resourceVersion"] = shoot.ResourceVersion

	if err := r.SeedClient.Update(ctx, configMap); err != nil {
		return reconcile.Result{}, fmt.Errorf("error updating configmap: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) resolveShoot(ctx context.Context, request reconcile.Request) (*gardencorev1beta1.Shoot, error) {
	log := logf.FromContext(ctx)

	namespacedName := request.NamespacedName
	if strings.HasPrefix(namespacedName.Name, "shoot-protocol--") {
		resolvedNamespacedName := getShootNamespacedNameFromConfigMapName(namespacedName.Name)

		if resolvedNamespacedName != nil {
			namespacedName = *resolvedNamespacedName
		}
	}

	shoot := &gardencorev1beta1.Shoot{}
	log.Info(fmt.Sprintf("namespacedName: %+v", namespacedName))
	if err := r.GardenClient.Get(ctx, namespacedName, shoot); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Object is gone, stop reconciling")
			return nil, nil
		}

		return nil, fmt.Errorf("error retrieving shoot from store: %w", err)
	}

	return shoot, nil
}

func getShootNamespacedNameFromConfigMapName(configMapName string) *types.NamespacedName {
	parts := strings.Split(configMapName, "--")
	if len(parts) != 3 {
		return nil
	}

	return &types.NamespacedName{
		Namespace: parts[1],
		Name:      parts[2],
	}
}
