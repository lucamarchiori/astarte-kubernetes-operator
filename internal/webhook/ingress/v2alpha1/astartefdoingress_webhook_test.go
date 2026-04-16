/*
This file is part of Astarte.

Copyright 2020-26 SECO Mind Srl.

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

package v2alpha1

import (
	"context"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	ingressv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/ingress/v2alpha1"
	integrationutils "github.com/astarte-platform/astarte-kubernetes-operator/test/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("AstarteFDOIngress Webhook testing", Ordered, Serial, func() {
	const (
		namespace   = "astarte-fdo-ingress-webhook-tests"
		astarteName = "example-astarte"
		secretName  = "fdo-tls-secret"
	)

	var astarte *apiv2alpha1.Astarte

	BeforeAll(func() {
		integrationutils.CreateNamespace(k8sClient, namespace)
	})

	AfterAll(func() {
		integrationutils.DeleteNamespace(k8sClient, namespace)
	})

	BeforeEach(func() {
		astarte = baseAstarte.DeepCopy()
		astarte.SetName(astarteName)
		astarte.SetNamespace(namespace)
		astarte.Spec.FDO = &apiv2alpha1.AstarteFDOSpec{Enable: true}
		integrationutils.DeployAstarte(k8sClient, astarte)

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"tls.crt": []byte("cert"),
				"tls.key": []byte("key"),
			},
		}
		Expect(k8sClient.Create(context.Background(), secret)).To(Succeed())
	})

	AfterEach(func() {
		integrationutils.TeardownResourcesInNamespace(context.Background(), k8sClient, namespace)
	})

	It("should default ingressClass to haproxy", func() {
		fdoIngress := &ingressv2alpha1.AstarteFDOIngress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fdo-ingress-defaults",
				Namespace: namespace,
			},
			Spec: ingressv2alpha1.AstarteFDOIngressSpec{
				Astarte: astarteName,
			},
		}

		Expect(k8sClient.Create(context.Background(), fdoIngress)).To(Succeed())
		Eventually(func(g Gomega) {
			created := &ingressv2alpha1.AstarteFDOIngress{}
			g.Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fdoIngress.Name, Namespace: namespace}, created)).To(Succeed())
			g.Expect(created.Spec.IngressClass).To(Equal("haproxy"))
		}, integrationutils.Timeout, integrationutils.Interval).Should(Succeed())
	})

	It("should reject a resource when the referenced Astarte does not exist", func() {
		fdoIngress := &ingressv2alpha1.AstarteFDOIngress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fdo-ingress-missing-astarte",
				Namespace: namespace,
			},
			Spec: ingressv2alpha1.AstarteFDOIngressSpec{
				Astarte: "missing-astarte",
			},
		}

		err := k8sClient.Create(context.Background(), fdoIngress)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsInvalid(err)).To(BeTrue())
	})

	It("should reject a resource when FDO is not enabled in the referenced Astarte", func() {
		astarte.Spec.FDO.Enable = false
		Expect(k8sClient.Update(context.Background(), astarte)).To(Succeed())

		fdoIngress := &ingressv2alpha1.AstarteFDOIngress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fdo-ingress-disabled-fdo",
				Namespace: namespace,
			},
			Spec: ingressv2alpha1.AstarteFDOIngressSpec{
				Astarte: astarteName,
			},
		}

		err := k8sClient.Create(context.Background(), fdoIngress)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsInvalid(err)).To(BeTrue())
	})

	It("should reject a resource when the TLS secret does not exist", func() {
		fdoIngress := &ingressv2alpha1.AstarteFDOIngress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fdo-ingress-missing-secret",
				Namespace: namespace,
			},
			Spec: ingressv2alpha1.AstarteFDOIngressSpec{
				Astarte:     astarteName,
				IngressClass: "haproxy",
				TLSSecret:   "missing-secret",
			},
		}

		err := k8sClient.Create(context.Background(), fdoIngress)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsInvalid(err)).To(BeTrue())
	})

	It("should accept a valid resource", func() {
		fdoIngress := &ingressv2alpha1.AstarteFDOIngress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fdo-ingress-valid",
				Namespace: namespace,
			},
			Spec: ingressv2alpha1.AstarteFDOIngressSpec{
				Astarte:      astarteName,
				IngressClass: "haproxy",
				TLSSecret:    secretName,
			},
		}

		Expect(k8sClient.Create(context.Background(), fdoIngress)).To(Succeed())
	})
})
