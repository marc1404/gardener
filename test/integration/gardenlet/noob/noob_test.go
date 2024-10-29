package noob_test

import (
	"fmt"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

var _ = Describe("Noob controller tests", func() {
	var shoot *gardencorev1beta1.Shoot

	BeforeEach(func() {
		By("Create Shoot")
		shoot = &gardencorev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: testID + "-",
				Namespace:    testNamespace.Name,
				Labels:       map[string]string{testID: testRunID},
			},
			Spec: gardencorev1beta1.ShootSpec{
				SecretBindingName: ptr.To("my-provider-account"),
				CloudProfileName:  ptr.To("test-cloudprofile"),
				Region:            "foo-region",
				Provider: gardencorev1beta1.Provider{
					Type: "aws",
					Workers: []gardencorev1beta1.Worker{
						{
							Name:    "cpu-worker",
							Minimum: 2,
							Maximum: 2,
							Machine: gardencorev1beta1.Machine{Type: "large"},
						},
					},
				},
				Kubernetes: gardencorev1beta1.Kubernetes{Version: "1.26.1"},
				Networking: &gardencorev1beta1.Networking{Type: ptr.To("foo-networking")},
			},
		}

		Expect(testClient.Create(ctx, shoot)).To(Succeed())

		DeferCleanup(func() {
			By("Delete Shoot")
			Expect(client.IgnoreNotFound(testClient.Delete(ctx, shoot))).To(Succeed())
		})
	})

	It("should create a ConfigMap that contains the resourceVersion of a Shoot cluster", func() {
		By("Wait until manager has observed Shoot")
		Eventually(func() error {
			return mgrClient.Get(ctx, client.ObjectKeyFromObject(shoot), &gardencorev1beta1.Shoot{})
		}).Should(Succeed())

		By("Wait until noob controller has created the ConfigMap")
		configMap := &corev1.ConfigMap{}
		getShootProtocolConfigMapEventually(shoot, configMap)

		Expect(configMap.Data).To(HaveKeyWithValue("resourceVersion", shoot.ResourceVersion))
	})

	It("should update the ConfigMap with the resourceVersion of a Shoot cluster", func() {
		configMap := &corev1.ConfigMap{}
		getShootProtocolConfigMapEventually(shoot, configMap)

		resourceVersion := configMap.Data["resourceVersion"]

		updateShootResourceVersion(shoot)

		protocolResourceVersion, _ := strconv.Atoi(resourceVersion)
		shootResourceVersion, _ := strconv.Atoi(shoot.ResourceVersion)
		Expect(protocolResourceVersion).To(BeNumerically("<", shootResourceVersion))
	})

	It("should recreate the shoot protocol ConfigMap when it's deleted", func() {
		configMap := &corev1.ConfigMap{}
		getShootProtocolConfigMapEventually(shoot, configMap)

		Expect(testClient.Delete(ctx, configMap)).To(Succeed())
		getShootProtocolConfigMapEventually(shoot, configMap)
	})

	It("should not modify a ConfigMap that only partially matches the expected name", func() {
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace.Name,
				Name:      "shoot-protocol--too--many--parts",
			},
			Data: map[string]string{
				"resourceVersion": "-1",
			},
		}

		Expect(testClient.Create(ctx, configMap)).To(Succeed())

		updateShootResourceVersion(shoot)

		Expect(testClient.Get(ctx, client.ObjectKeyFromObject(configMap), configMap)).To(Succeed())
		Expect(configMap.Data["resourceVersion"]).To(Equal("-1"))
	})

	It("should not modify a ConfigMap that refers to a Shoot that doesn't exist", func() {
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace.Name,
				Name:      "shoot-protocol--doesnt--exist",
			},
			Data: map[string]string{
				"resourceVersion": "-1",
			},
		}

		Expect(testClient.Create(ctx, configMap)).To(Succeed())

		updateShootResourceVersion(shoot)

		Expect(testClient.Get(ctx, client.ObjectKeyFromObject(configMap), configMap)).To(Succeed())
		Expect(configMap.Data["resourceVersion"]).To(Equal("-1"))
	})
})

func getShootProtocolConfigMapEventually(shoot *gardencorev1beta1.Shoot, configMap *corev1.ConfigMap) {
	configMapKey := client.ObjectKey{
		Namespace: testNamespace.Name,
		Name:      fmt.Sprintf("shoot-protocol--%v--%v", shoot.Namespace, shoot.Name),
	}

	Eventually(func() error {
		return testClient.Get(ctx, configMapKey, configMap)
	}).Should(Succeed())
}

func updateShootResourceVersion(shoot *gardencorev1beta1.Shoot) {
	shoot.Labels["bogus"] = strconv.Itoa(int(time.Now().Unix()))

	Expect(testClient.Update(ctx, shoot)).To(Succeed())
}
