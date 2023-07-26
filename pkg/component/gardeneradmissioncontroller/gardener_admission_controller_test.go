// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gardeneradmissioncontroller_test

import (
	"context"
	"encoding/json"

	"github.com/Masterminds/semver"
	"github.com/hashicorp/go-multierror"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	certificatesv1 "k8s.io/api/certificates/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	admissioncontrollerv1alpha1 "github.com/gardener/gardener/pkg/admissioncontroller/apis/config/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	operationsv1alpha1 "github.com/gardener/gardener/pkg/apis/operations/v1alpha1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/component"
	. "github.com/gardener/gardener/pkg/component/gardeneradmissioncontroller"
	componenttest "github.com/gardener/gardener/pkg/component/test"
	"github.com/gardener/gardener/pkg/logger"
	operatorclient "github.com/gardener/gardener/pkg/operator/client"
	"github.com/gardener/gardener/pkg/resourcemanager/controller/garbagecollector/references"
	"github.com/gardener/gardener/pkg/utils"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"
	retryfake "github.com/gardener/gardener/pkg/utils/retry/fake"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"
	"github.com/gardener/gardener/pkg/utils/test"
	"github.com/gardener/gardener/pkg/utils/test/matchers"
)

const (
	managedResourceNameRuntime = "gardener-admission-controller-runtime"
	managedResourceNameVirtual = "gardener-admission-controller-virtual"
)

var _ = Describe("GardenerAdmissionController", func() {
	var (
		ctx context.Context

		fakeOps           *retryfake.Ops
		fakeClient        client.Client
		fakeSecretManager secretsmanager.Interface
		deployer          component.DeployWaiter
		testValues        Values

		namespace = "some-namespace"
	)

	BeforeEach(func() {
		ctx = context.Background()

		fakeOps = &retryfake.Ops{MaxAttempts: 2}
		DeferCleanup(test.WithVars(
			&retry.Until, fakeOps.Until,
		))

		fakeClient = fakeclient.NewClientBuilder().WithScheme(operatorclient.RuntimeScheme).Build()
		fakeSecretManager = fakesecretsmanager.New(fakeClient, namespace)

		testValues = Values{}
	})

	JustBeforeEach(func() {
		deployer = New(fakeClient, namespace, fakeSecretManager, testValues)
	})

	Describe("#Deploy", func() {
		BeforeEach(func() {
			blockMode := admissioncontrollerv1alpha1.ResourceAdmissionWebhookMode("block")

			// These are typical configuration values set for the admission controller and serves as the base for the following tests.
			testValues = Values{
				RuntimeVersion: semver.MustParse("v1.27.0"),
				ResourceAdmissionConfiguration: &admissioncontrollerv1alpha1.ResourceAdmissionConfiguration{
					Limits: []admissioncontrollerv1alpha1.ResourceLimit{
						{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"secrets", "configmaps"},
							Size:        resource.MustParse("1Mi"),
						},
						{
							APIGroups:   []string{"core.gardener.cloud"},
							APIVersions: []string{"v1beta1"},
							Resources:   []string{"shoots"},
							Size:        resource.MustParse("100Ki"),
						},
					},
					UnrestrictedSubjects: []rbacv1.Subject{{
						Kind:      "ServiceAccount",
						Name:      "foo",
						Namespace: "default",
					}},
					OperationMode: &blockMode,
				},
				TopologyAwareRoutingEnabled: true,
			}

			Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ca-gardener", Namespace: namespace}})).To(Succeed())
			Expect(fakeClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "generic-token-kubeconfig", Namespace: namespace}})).To(Succeed())
		})

		Context("with common values", func() {
			It("should successfully deploy", func() {
				Expect(deployer.Deploy(ctx)).To(Succeed())
				verifyExpectations(ctx, fakeClient, fakeSecretManager, namespace, "4ef77c17", testValues)
			})
		})

		Context("when TopologyAwareRouting is disabled", func() {
			BeforeEach(func() {
				testValues.TopologyAwareRoutingEnabled = false
			})

			It("should successfully deploy", func() {
				Expect(deployer.Deploy(ctx)).To(Succeed())
				verifyExpectations(ctx, fakeClient, fakeSecretManager, namespace, "4ef77c17", testValues)
			})
		})

		Context("when TopologyAwareRouting is enabled for Kubernetes versions <= 1.26", func() {
			BeforeEach(func() {
				testValues.TopologyAwareRoutingEnabled = true
				testValues.RuntimeVersion = semver.MustParse("v1.26.5")
			})

			It("should successfully deploy", func() {
				Expect(deployer.Deploy(ctx)).To(Succeed())
				verifyExpectations(ctx, fakeClient, fakeSecretManager, namespace, "4ef77c17", testValues)
			})
		})

		Context("without ResourceAdmissionConfiguration", func() {
			BeforeEach(func() {
				testValues.ResourceAdmissionConfiguration = nil
			})

			It("should successfully deploy", func() {
				Expect(deployer.Deploy(ctx)).To(Succeed())
				verifyExpectations(ctx, fakeClient, fakeSecretManager, namespace, "6d282905", testValues)
			})
		})
	})

	Describe("#Wait", func() {
		Context("when ManagedResources are ready", func() {
			It("should successfully wait", func() {
				runtimeManagedResource := &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedResourceNameRuntime,
						Namespace: namespace,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						Conditions: []gardencorev1beta1.Condition{
							{Type: resourcesv1alpha1.ResourcesApplied, Status: gardencorev1beta1.ConditionTrue},
							{Type: resourcesv1alpha1.ResourcesHealthy, Status: gardencorev1beta1.ConditionTrue},
						},
					},
				}
				Expect(fakeClient.Create(ctx, runtimeManagedResource)).To(Succeed())

				virtualManagedResource := &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedResourceNameVirtual,
						Namespace: namespace,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						Conditions: []gardencorev1beta1.Condition{
							{Type: resourcesv1alpha1.ResourcesApplied, Status: gardencorev1beta1.ConditionTrue},
							{Type: resourcesv1alpha1.ResourcesHealthy, Status: gardencorev1beta1.ConditionTrue},
						},
					},
				}
				Expect(fakeClient.Create(ctx, virtualManagedResource)).To(Succeed())

				Expect(deployer.Wait(ctx)).To(Succeed())
			})
		})

		Context("when Runtime ManagedResource doesn't get ready", func() {
			It("should time out waiting", func() {
				runtimeManagedResource := &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedResourceNameRuntime,
						Namespace: namespace,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						Conditions: []gardencorev1beta1.Condition{
							{Type: resourcesv1alpha1.ResourcesApplied, Status: gardencorev1beta1.ConditionFalse},
							{Type: resourcesv1alpha1.ResourcesHealthy, Status: gardencorev1beta1.ConditionTrue},
						},
					},
				}
				Expect(fakeClient.Create(ctx, runtimeManagedResource)).To(Succeed())

				virtualManagedResource := &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedResourceNameVirtual,
						Namespace: namespace,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						Conditions: []gardencorev1beta1.Condition{
							{Type: resourcesv1alpha1.ResourcesApplied, Status: gardencorev1beta1.ConditionTrue},
							{Type: resourcesv1alpha1.ResourcesHealthy, Status: gardencorev1beta1.ConditionTrue},
						},
					},
				}
				Expect(fakeClient.Create(ctx, virtualManagedResource)).To(Succeed())

				err := deployer.Wait(ctx)

				multiErr, ok := err.(*multierror.Error)
				Expect(ok).To(BeTrue())
				Expect(multiErr.Errors).To(HaveLen(1))
				Expect(multiErr.Errors[0]).To(MatchError("retry failed with max attempts reached, last error: managed resource some-namespace/gardener-admission-controller-runtime is not healthy"))
			})
		})

		Context("when Virtual ManagedResource doesn't get ready", func() {
			It("should time out waiting", func() {
				runtimeManagedResource := &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedResourceNameRuntime,
						Namespace: namespace,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						Conditions: []gardencorev1beta1.Condition{
							{Type: resourcesv1alpha1.ResourcesApplied, Status: gardencorev1beta1.ConditionTrue},
							{Type: resourcesv1alpha1.ResourcesHealthy, Status: gardencorev1beta1.ConditionTrue},
						},
					},
				}
				Expect(fakeClient.Create(ctx, runtimeManagedResource)).To(Succeed())

				virtualManagedResource := &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedResourceNameVirtual,
						Namespace: namespace,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						Conditions: []gardencorev1beta1.Condition{
							{Type: resourcesv1alpha1.ResourcesApplied, Status: gardencorev1beta1.ConditionTrue},
							{Type: resourcesv1alpha1.ResourcesHealthy, Status: gardencorev1beta1.ConditionProgressing},
						},
					},
				}
				Expect(fakeClient.Create(ctx, virtualManagedResource)).To(Succeed())

				err := deployer.Wait(ctx)

				multiErr, ok := err.(*multierror.Error)
				Expect(ok).To(BeTrue())
				Expect(multiErr.Errors).To(HaveLen(1))
				Expect(multiErr.Errors[0]).To(MatchError("retry failed with max attempts reached, last error: managed resource some-namespace/gardener-admission-controller-virtual is not healthy"))
			})
		})

		Context("when ManagedResources are not available", func() {
			It("should time out waiting", func() {
				Expect(deployer.Wait(ctx).Error()).To(And(
					ContainSubstring("managedresources.resources.gardener.cloud \"gardener-admission-controller-virtual\" not found"),
					ContainSubstring("managedresources.resources.gardener.cloud \"gardener-admission-controller-runtime\" not found"),
				))
			})
		})
	})

	Describe("#WaitCleanup", func() {
		Context("when ManagedResources are not available", func() {
			It("should successfully wait", func() {
				Expect(deployer.WaitCleanup(ctx)).To(Succeed())
			})
		})

		Context("when Runtime ManagedResource is still available", func() {
			It("should time out waiting", func() {
				runtimeManagedResource := &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedResourceNameRuntime,
						Namespace: namespace,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						Conditions: []gardencorev1beta1.Condition{
							{Type: resourcesv1alpha1.ResourcesApplied, Status: gardencorev1beta1.ConditionFalse},
							{Type: resourcesv1alpha1.ResourcesHealthy, Status: gardencorev1beta1.ConditionTrue},
						},
					},
				}
				Expect(fakeClient.Create(ctx, runtimeManagedResource)).To(Succeed())

				err := deployer.WaitCleanup(ctx)

				multiErr, ok := err.(*multierror.Error)
				Expect(ok).To(BeTrue())
				Expect(multiErr.Errors).To(HaveLen(1))
				Expect(multiErr.Errors[0]).To(MatchError("retry failed with max attempts reached, last error: resource some-namespace/gardener-admission-controller-runtime still exists"))
			})
		})

		Context("when Virtual ManagedResource is still available", func() {
			It("should time out waiting", func() {
				runtimeManagedResource := &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedResourceNameVirtual,
						Namespace: namespace,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						Conditions: []gardencorev1beta1.Condition{
							{Type: resourcesv1alpha1.ResourcesApplied, Status: gardencorev1beta1.ConditionTrue},
							{Type: resourcesv1alpha1.ResourcesHealthy, Status: gardencorev1beta1.ConditionTrue},
						},
					},
				}
				Expect(fakeClient.Create(ctx, runtimeManagedResource)).To(Succeed())

				err := deployer.WaitCleanup(ctx)

				multiErr, ok := err.(*multierror.Error)
				Expect(ok).To(BeTrue())
				Expect(multiErr.Errors).To(HaveLen(1))
				Expect(multiErr.Errors[0]).To(MatchError("retry failed with max attempts reached, last error: resource some-namespace/gardener-admission-controller-virtual still exists"))
			})
		})
	})

	Describe("#Destroy", func() {
		Context("when resources don't exist", func() {
			It("should successful destroy", func() {
				Expect(deployer.Destroy(ctx)).To(Succeed())

				verifyResourcesGone(ctx, fakeClient, namespace)
			})
		})

		It("should successful destroy", func() {
			runtimeManagedResource := &resourcesv1alpha1.ManagedResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      managedResourceNameRuntime,
					Namespace: namespace,
				},
				Spec: resourcesv1alpha1.ManagedResourceSpec{
					SecretRefs: []corev1.LocalObjectReference{{Name: "managedresource-" + managedResourceNameRuntime}},
				},
			}
			runtimeManagedResourceSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      runtimeManagedResource.Spec.SecretRefs[0].Name,
					Namespace: namespace,
				},
			}

			Expect(fakeClient.Create(ctx, runtimeManagedResourceSecret)).To(Succeed())
			Expect(fakeClient.Create(ctx, runtimeManagedResource)).To(Succeed())

			Expect(deployer.Destroy(ctx)).To(Succeed())

			verifyResourcesGone(ctx, fakeClient, namespace)
		})
	})
})

func verifyResourcesGone(ctx context.Context, fakeClient client.Client, namespace string) {
	ExpectWithOffset(1, fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: "managedresource-" + managedResourceNameRuntime}, &corev1.Secret{})).To(matchers.BeNotFoundError())
	ExpectWithOffset(1, fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: managedResourceNameRuntime}, &resourcesv1alpha1.ManagedResource{})).To(matchers.BeNotFoundError())
	ExpectWithOffset(1, fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: "managedresource-" + managedResourceNameVirtual}, &corev1.Secret{})).To(matchers.BeNotFoundError())
	ExpectWithOffset(1, fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: managedResourceNameVirtual}, &resourcesv1alpha1.ManagedResource{})).To(matchers.BeNotFoundError())
	ExpectWithOffset(1, fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: "shoot-access-gardener-admission-controller"}, &corev1.Secret{})).To(matchers.BeNotFoundError())
}

func verifyExpectations(ctx context.Context, fakeClient client.Client, fakeSecretManager secretsmanager.Interface, namespace, configMapChecksum string, testValues Values) {
	By("Check Gardener Access Secret")
	accessSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shoot-access-gardener-admission-controller",
			Namespace: namespace,
			Labels: map[string]string{
				"resources.gardener.cloud/purpose": "token-requestor",
				"resources.gardener.cloud/class":   "shoot",
			},
			Annotations: map[string]string{
				"serviceaccount.resources.gardener.cloud/name":      "gardener-admission-controller",
				"serviceaccount.resources.gardener.cloud/namespace": "kube-system",
			},
		},
		Type: corev1.SecretTypeOpaque,
	}

	actualShootAccessSecret := &corev1.Secret{}
	Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(accessSecret), actualShootAccessSecret)).To(Succeed())
	accessSecret.ResourceVersion = "1"
	Expect(actualShootAccessSecret).To(Equal(accessSecret))

	By("Check Runtime Cluster Resources")
	serverCert, ok := fakeSecretManager.Get("gardener-admission-controller-cert")
	Expect(ok).To(BeTrue())

	runtimeManagedResourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managedresource-" + managedResourceNameRuntime,
			Namespace: namespace,
		},
	}
	Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(runtimeManagedResourceSecret), runtimeManagedResourceSecret)).To(Succeed())

	Expect(string(runtimeManagedResourceSecret.Data["configmap__some-namespace__gardener-admission-controller-"+configMapChecksum+".yaml"])).To(Equal(configMap(namespace, testValues)))
	Expect(string(runtimeManagedResourceSecret.Data["deployment__some-namespace__gardener-admission-controller.yaml"])).To(Equal(deployment(namespace, "gardener-admission-controller-"+configMapChecksum, serverCert.Name, testValues)))
	Expect(string(runtimeManagedResourceSecret.Data["service__some-namespace__gardener-admission-controller.yaml"])).To(Equal(service(namespace, testValues)))
	Expect(string(runtimeManagedResourceSecret.Data["verticalpodautoscaler__some-namespace__gardener-admission-controller.yaml"])).To(Equal(vpa(namespace)))

	By("Check Virtual Cluster Resources")
	virtualManagedResourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managedresource-" + managedResourceNameVirtual,
			Namespace: namespace,
		},
	}
	Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(virtualManagedResourceSecret), virtualManagedResourceSecret)).To(Succeed())

	caGardener, ok := fakeSecretManager.Get("ca-gardener")
	Expect(ok).To(BeTrue())

	Expect(string(virtualManagedResourceSecret.Data["clusterrole____gardener.cloud_system_admission-controller.yaml"])).To(Equal(clusterRole()))
	Expect(string(virtualManagedResourceSecret.Data["clusterrolebinding____gardener.cloud_admission-controller.yaml"])).To(Equal(clusterRoleBinding()))
	Expect(string(virtualManagedResourceSecret.Data["validatingwebhookconfiguration____gardener-admission-controller.yaml"])).To(Equal(validatingWebhookConfiguration(namespace, caGardener.Data["bundle.crt"], testValues)))
}

func configMap(namespace string, testValues Values) string {
	admissionConfig := &admissioncontrollerv1alpha1.AdmissionControllerConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissioncontroller.config.gardener.cloud/v1alpha1",
			Kind:       "AdmissionControllerConfiguration",
		},
		GardenClientConnection: componentbaseconfigv1alpha1.ClientConnectionConfiguration{
			QPS:        100,
			Burst:      130,
			Kubeconfig: "/var/run/secrets/gardener.cloud/shoot/generic-kubeconfig/kubeconfig",
		},
		LogLevel:  testValues.LogLevel,
		LogFormat: logger.FormatJSON,
		Server: admissioncontrollerv1alpha1.ServerConfiguration{
			Webhooks: admissioncontrollerv1alpha1.HTTPSServer{
				Server: admissioncontrollerv1alpha1.Server{Port: 2719},
				TLS:    admissioncontrollerv1alpha1.TLSServer{ServerCertDir: "/etc/gardener-admission-controller/srv"},
			},
			HealthProbes:                   &admissioncontrollerv1alpha1.Server{Port: 2722},
			Metrics:                        &admissioncontrollerv1alpha1.Server{Port: 2723},
			ResourceAdmissionConfiguration: testValues.ResourceAdmissionConfiguration,
		},
	}

	data, err := json.Marshal(admissionConfig)
	utilruntime.Must(err)
	data, err = yaml.JSONToYAML(data)
	utilruntime.Must(err)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels: map[string]string{
				"app":  "gardener",
				"role": "admission-controller",
			},
			Name:      "gardener-admission-controller",
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.yaml": string(data),
		},
	}
	utilruntime.Must(kubernetesutils.MakeUnique(configMap))

	return componenttest.Serialize(configMap)
}

func deployment(namespace, configSecretName, serverCertSecretName string, testValues Values) string {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gardener-admission-controller",
			Namespace: namespace,
			Labels: map[string]string{
				"app":  "gardener",
				"role": "admission-controller",
				"high-availability-config.resources.gardener.cloud/type": "server",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":  "gardener",
					"role": "admission-controller",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: utils.MergeStringMaps(GetLabels(), map[string]string{
						"app":                              "gardener",
						"role":                             "admission-controller",
						"networking.gardener.cloud/to-dns": "allowed",
						"networking.resources.gardener.cloud/to-virtual-garden-kube-apiserver-tcp-443": "allowed",
					}),
				},
				Spec: corev1.PodSpec{
					PriorityClassName:            "gardener-garden-system-400",
					AutomountServiceAccountToken: pointer.Bool(false),
					Containers: []corev1.Container{
						{
							Name:            "gardener-admission-controller",
							Image:           testValues.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: []string{
								"--config=/etc/gardener-admission-controller/config/config.yaml",
							},
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("200Mi"),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(2722),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      5,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/readyz",
										Port:   intstr.FromInt(2722),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 10,
								TimeoutSeconds:      5,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "gardener-admission-controller-cert",
									MountPath: "/etc/gardener-admission-controller/srv",
									ReadOnly:  true,
								},
								{
									Name:      "gardener-admission-controller-config",
									MountPath: "/etc/gardener-admission-controller/config",
								},
								{
									Name:      "kubeconfig",
									MountPath: "/var/run/secrets/gardener.cloud/shoot/generic-kubeconfig",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "gardener-admission-controller-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{SecretName: serverCertSecretName},
							},
						},
						{
							Name: "gardener-admission-controller-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: configSecretName},
								},
							},
						},
						{
							Name: "kubeconfig",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									DefaultMode: pointer.Int32(420),
									Sources: []corev1.VolumeProjection{
										{
											Secret: &corev1.SecretProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "generic-token-kubeconfig",
												},
												Items: []corev1.KeyToPath{{
													Key:  "kubeconfig",
													Path: "kubeconfig",
												}},
												Optional: pointer.Bool(false),
											},
										},
										{
											Secret: &corev1.SecretProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "shoot-access-gardener-admission-controller",
												},
												Items: []corev1.KeyToPath{{
													Key:  "token",
													Path: "token",
												}},
												Optional: pointer.Bool(false),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	utilruntime.Must(references.InjectAnnotations(deployment))

	return componenttest.Serialize(deployment)
}

func service(namespace string, testValues Values) string {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gardener-admission-controller",
			Namespace: namespace,
			Labels: map[string]string{
				"app":  "gardener",
				"role": "admission-controller",
			},
			Annotations: map[string]string{
				"networking.resources.gardener.cloud/from-all-webhook-targets-allowed-ports": `[{"protocol":"TCP","port":2719}]`,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app":  "gardener",
				"role": "admission-controller",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Protocol:   corev1.ProtocolTCP,
					Port:       443,
					TargetPort: intstr.FromInt(2719),
				},
				{
					Name:       "metrics",
					Protocol:   corev1.ProtocolTCP,
					Port:       2723,
					TargetPort: intstr.FromInt(2723),
				},
			},
		},
	}

	if testValues.TopologyAwareRoutingEnabled {
		metav1.SetMetaDataLabel(&svc.ObjectMeta, "endpoint-slice-hints.resources.gardener.cloud/consider", "true")
		if testValues.RuntimeVersion.LessThan(semver.MustParse("v1.27")) {
			metav1.SetMetaDataAnnotation(&svc.ObjectMeta, "service.kubernetes.io/topology-aware-hints", "auto")
		} else {
			metav1.SetMetaDataAnnotation(&svc.ObjectMeta, "service.kubernetes.io/topology-mode", "auto")
		}
	}

	return componenttest.Serialize(svc)
}

func vpa(namespace string) string {
	autoUpdateMode := vpaautoscalingv1.UpdateModeAuto

	return componenttest.Serialize(&vpaautoscalingv1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gardener-admission-controller",
			Namespace: namespace,
			Labels: map[string]string{
				"app":  "gardener",
				"role": "admission-controller",
			},
		},
		Spec: vpaautoscalingv1.VerticalPodAutoscalerSpec{
			TargetRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "gardener-admission-controller",
			},
			UpdatePolicy: &vpaautoscalingv1.PodUpdatePolicy{
				UpdateMode: &autoUpdateMode,
			},
			ResourcePolicy: &vpaautoscalingv1.PodResourcePolicy{
				ContainerPolicies: []vpaautoscalingv1.ContainerResourcePolicy{
					{
						ContainerName: "*",
						MinAllowed: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("25Mi"),
						},
					},
				},
			},
		},
	})
}

func clusterRole() string {
	return componenttest.Serialize(&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gardener.cloud:system:admission-controller",
			Labels: map[string]string{
				"app":  "gardener",
				"role": "admission-controller",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"core.gardener.cloud"},
				Resources: []string{
					"backupbuckets",
					"backupentries",
					"controllerinstallations",
					"secretbindings",
					"seeds",
					"shoots",
					"projects",
				},
				Verbs: []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"seedmanagement.gardener.cloud"},
				Resources: []string{
					"managedseeds",
				},
				Verbs: []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"operations.gardener.cloud"},
				Resources: []string{
					"bastions",
				},
				Verbs: []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{
					"configmaps",
				},
				Verbs: []string{"get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{
					"namespaces",
					"secrets",
					"serviceaccounts",
				},
				Verbs: []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{
					"leases",
				},
				Verbs: []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"certificates.k8s.io"},
				Resources: []string{
					"certificatesigningrequests",
				},
				Verbs: []string{"get", "list", "watch"},
			},
		},
	})
}

func clusterRoleBinding() string {
	return componenttest.Serialize(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gardener.cloud:admission-controller",
			Labels: map[string]string{
				"app":  "gardener",
				"role": "admission-controller",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "gardener.cloud:system:admission-controller",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "gardener-admission-controller",
			Namespace: "kube-system",
		}},
	})
}

func validatingWebhookConfiguration(namespace string, caBundle []byte, testValues Values) string {
	var (
		failurePolicyFail     = admissionregistrationv1.Fail
		sideEffectsNone       = admissionregistrationv1.SideEffectClassNone
		matchPolicyEquivalent = admissionregistrationv1.Equivalent
	)

	webhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gardener-admission-controller",
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name:                    "validate-namespace-deletion.gardener.cloud",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				TimeoutSeconds:          pointer.Int32(10),
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Delete},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"namespaces"},
					},
				}},
				FailurePolicy: &failurePolicyFail,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"gardener.cloud/role": "project",
					},
				},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL:      pointer.String("https://gardener-admission-controller." + namespace + "/webhooks/validate-namespace-deletion"),
					CABundle: caBundle,
				},
				SideEffects: &sideEffectsNone,
			},
			{
				Name:                    "validate-kubeconfig-secrets.gardener.cloud",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				TimeoutSeconds:          pointer.Int32(10),
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"secrets"},
					},
				}},
				FailurePolicy: &failurePolicyFail,
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{Key: "gardener.cloud/role", Operator: metav1.LabelSelectorOpIn, Values: []string{"project"}},
						{Key: "app", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"gardener"}},
					},
				},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL:      pointer.String("https://gardener-admission-controller." + namespace + "/webhooks/validate-kubeconfig-secrets"),
					CABundle: caBundle,
				},
				SideEffects: &sideEffectsNone,
			},
			{
				Name:                    "seed-restriction.gardener.cloud",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				TimeoutSeconds:          pointer.Int32(10),
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"secrets", "serviceaccounts"},
						},
					},
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{rbacv1.GroupName},
							APIVersions: []string{"v1"},
							Resources:   []string{"clusterrolebindings"},
						},
					},
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{coordinationv1.GroupName},
							APIVersions: []string{"v1"},
							Resources:   []string{"leases"},
						},
					},
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{certificatesv1.GroupName},
							APIVersions: []string{"v1"},
							Resources:   []string{"certificatesigningrequests"},
						},
					},
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{gardencorev1beta1.GroupName},
							APIVersions: []string{"v1beta1"},
							Resources:   []string{"backupentries", "internalsecrets", "shootstates"},
						},
					},
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Delete},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{gardencorev1beta1.GroupName},
							APIVersions: []string{"v1beta1"},
							Resources:   []string{"backupbuckets"},
						},
					},
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update, admissionregistrationv1.Delete},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{gardencorev1beta1.GroupName},
							APIVersions: []string{"v1beta1"},
							Resources:   []string{"seeds"},
						},
					},
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{operationsv1alpha1.GroupName},
							APIVersions: []string{"v1alpha1"},
							Resources:   []string{"bastions"},
						},
					},
				},
				FailurePolicy: &failurePolicyFail,
				MatchPolicy:   &matchPolicyEquivalent,
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL:      pointer.String("https://gardener-admission-controller." + namespace + "/webhooks/admission/seedrestriction"),
					CABundle: caBundle,
				},
				SideEffects: &sideEffectsNone,
			},
			{
				Name:                    "internal-domain-secret.gardener.cloud",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				TimeoutSeconds:          pointer.Int32(10),
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update, admissionregistrationv1.Delete},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"secrets"},
					},
				}},
				FailurePolicy: &failurePolicyFail,
				ObjectSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"role": "internal-domain",
					},
				},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL:      pointer.String("https://gardener-admission-controller." + namespace + "/webhooks/admission/validate-internal-domain"),
					CABundle: caBundle,
				},
				SideEffects: &sideEffectsNone,
			},
			{
				Name:                    "audit-policies.gardener.cloud",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				TimeoutSeconds:          pointer.Int32(10),
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{gardencorev1beta1.GroupName},
							APIVersions: []string{"v1beta1"},
							Resources:   []string{"shoots"},
						},
					},
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Update},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"configmaps"},
						},
					},
				},
				FailurePolicy: &failurePolicyFail,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"gardener.cloud/role": "project",
					},
				},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL:      pointer.String("https://gardener-admission-controller." + namespace + "/webhooks/audit-policies"),
					CABundle: caBundle,
				},
				SideEffects: &sideEffectsNone,
			},
			{
				Name:                    "admission-plugin-secret.gardener.cloud",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				TimeoutSeconds:          pointer.Int32(10),
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Update},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"secrets"},
						},
					},
				},
				FailurePolicy: &failurePolicyFail,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"gardener.cloud/role": "project",
					},
				},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL:      pointer.String("https://gardener-admission-controller." + namespace + "/webhooks/validate-admission-plugin-secret"),
					CABundle: caBundle,
				},
				SideEffects: &sideEffectsNone,
			},
		},
	}

	if testValues.ResourceAdmissionConfiguration != nil {
		webhookConfig.Webhooks = append(webhookConfig.Webhooks, admissionregistrationv1.ValidatingWebhook{
			Name:                    "validate-resource-size.gardener.cloud",
			AdmissionReviewVersions: []string{"v1", "v1beta1"},
			TimeoutSeconds:          pointer.Int32(10),
			Rules: []admissionregistrationv1.RuleWithOperations{
				{
					Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"secrets", "configmaps"},
					},
				},
				{
					Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{"core.gardener.cloud"},
						APIVersions: []string{"v1beta1"},
						Resources:   []string{"shoots"},
					},
				},
			},
			FailurePolicy: &failurePolicyFail,
			NamespaceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "gardener.cloud/role", Operator: metav1.LabelSelectorOpIn, Values: []string{"project"}},
					{Key: "app", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"gardener"}},
				},
			},
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				URL:      pointer.String("https://gardener-admission-controller." + namespace + "/webhooks/validate-resource-size"),
				CABundle: caBundle,
			},
			SideEffects: &sideEffectsNone,
		})
	}

	return componenttest.Serialize(webhookConfig)
}