/*
This file is part of Astarte.

Copyright 2020-25 SECO Mind Srl.

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

// Why is there a no-lint here? Well, golangci-lint is failing on this whole file
// and it is not giving any useful information on why. Probably a bug in golangci-lint
// that is outdated. Disabling linting for this file for now, until we can upgrade golangci-lint.
// nolint
package reconcile

import (
	"context"
	"strings"

	apiv2alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/api/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.openly.dev/pointy"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Astarte Generic API reconcile tests", Ordered, Serial, func() {
	const (
		CustomAstarteName       = "my-astarte-generic-api"
		CustomAstarteNamespace  = "astarte-generic-api-test"
		CustomAstarteInstanceID = "myastarteinstanceid"
		AstarteVersion          = "1.3.0"
		CustomRabbitMQHost      = "rabbitmq.example.com"
		CustomRabbitMQPort      = 5672
		CustomVerneMQHost       = "vernemq.example.com"
		CustomVerneMQPort       = 8883
	)

	var cr *apiv2alpha1.Astarte

	BeforeAll(func() {
		if CustomAstarteNamespace != "default" {
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}}
			Eventually(func() error {
				err := k8sClient.Create(context.Background(), ns)
				if apierrors.IsAlreadyExists(err) {
					return nil
				}
				return err
			}, "10s", "250ms").Should(Succeed())
		}
	})

	AfterAll(func() {
		if CustomAstarteNamespace != "default" {
			astartes := &apiv2alpha1.AstarteList{}
			Expect(k8sClient.List(context.Background(), astartes, client.InNamespace(CustomAstarteNamespace))).To(Succeed())
			for _, a := range astartes.Items {
				_ = k8sClient.Delete(context.Background(), &a)
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &apiv2alpha1.Astarte{})
				}, "10s", "250ms").ShouldNot(Succeed())
			}
			// Do not delete the namespace here to avoid 'NamespaceTerminating' flakiness in subsequent specs
			// _ = k8sClient.Delete(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: CustomAstarteNamespace}})
		}
	})

	BeforeEach(func() {
		// Create and initialize a basic Astarte CR
		cr = &apiv2alpha1.Astarte{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CustomAstarteName,
				Namespace: CustomAstarteNamespace,
			},
			Spec: apiv2alpha1.AstarteSpec{
				Version: AstarteVersion,
				RabbitMQ: apiv2alpha1.AstarteRabbitMQSpec{
					Connection: &apiv2alpha1.AstarteRabbitMQConnectionSpec{
						HostAndPort: apiv2alpha1.HostAndPort{
							Host: CustomRabbitMQHost,
							Port: pointy.Int32(CustomRabbitMQPort),
						},
					},
				},
				VerneMQ: apiv2alpha1.AstarteVerneMQSpec{
					HostAndPort: apiv2alpha1.HostAndPort{
						Host: CustomVerneMQHost,
						Port: pointy.Int32(CustomVerneMQPort),
					},
				},
				Cassandra: apiv2alpha1.AstarteCassandraSpec{
					Connection: &apiv2alpha1.AstarteCassandraConnectionSpec{
						Nodes: []apiv2alpha1.HostAndPort{
							{
								Host: "cassandra.example.com",
								Port: pointy.Int32(9042),
							},
						},
					},
				},
				Components: apiv2alpha1.AstarteComponentsSpec{
					AppengineAPI:    apiv2alpha1.AstarteAppengineAPISpec{},
					RealmManagement: apiv2alpha1.AstarteGenericAPIComponentSpec{},
					Pairing:         apiv2alpha1.AstarteGenericAPIComponentSpec{},
					Housekeeping:    apiv2alpha1.AstarteGenericAPIComponentSpec{},
					Flow:            apiv2alpha1.AstarteGenericAPIComponentSpec{},
				},
			},
		}

		Expect(k8sClient.Create(context.Background(), cr)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
		}, "10s", "250ms").Should(Succeed())
	})

	AfterEach(func() {
		astartes := &apiv2alpha1.AstarteList{}
		Expect(k8sClient.List(context.Background(), astartes, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, a := range astartes.Items {
			Expect(k8sClient.Delete(context.Background(), &a)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: a.Name, Namespace: a.Namespace}, &apiv2alpha1.Astarte{})
			}, "10s", "250ms").ShouldNot(Succeed())
		}

		// Clean up any deployments left behind
		deployments := &appsv1.DeploymentList{}
		Expect(k8sClient.List(context.Background(), deployments, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, d := range deployments.Items {
			_ = k8sClient.Delete(context.Background(), &d)
		}

		// Clean up any services left behind
		services := &v1.ServiceList{}
		Expect(k8sClient.List(context.Background(), services, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, s := range services.Items {
			if s.Name != "kubernetes" {
				_ = k8sClient.Delete(context.Background(), &s)
			}
		}

		// Clean up any secrets left behind
		secrets := &v1.SecretList{}
		Expect(k8sClient.List(context.Background(), secrets, &client.ListOptions{Namespace: CustomAstarteNamespace})).To(Succeed())
		for _, s := range secrets.Items {
			if !strings.Contains(s.Name, "token") {
				_ = k8sClient.Delete(context.Background(), &s)
			}
		}

		Eventually(func() int {
			list := &apiv2alpha1.AstarteList{}
			_ = k8sClient.List(context.Background(), list, &client.ListOptions{Namespace: CustomAstarteNamespace})
			return len(list.Items)
		}, "10s", "250ms").Should(Equal(0))
	})

	Describe("Test EnsureAstarteGenericAPIComponent", func() {
		DescribeTable("should create deployment, service and cookie secret for different components",
			func(component apiv2alpha1.AstarteComponent) {
				// Set up component spec
				var apiSpec apiv2alpha1.AstarteGenericAPIComponentSpec
				switch component {
				case apiv2alpha1.AppEngineAPI:
					cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
					apiSpec = cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec
				case apiv2alpha1.RealmManagement:
					cr.Spec.Components.RealmManagement.Deploy = pointy.Bool(true)
					apiSpec = cr.Spec.Components.RealmManagement
				case apiv2alpha1.Pairing:
					cr.Spec.Components.Pairing.Deploy = pointy.Bool(true)
					apiSpec = cr.Spec.Components.Pairing
				case apiv2alpha1.Housekeeping:
					cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(true)
					apiSpec = cr.Spec.Components.Housekeeping
				case apiv2alpha1.FlowComponent:
					cr.Spec.Components.Flow.Deploy = pointy.Bool(true)
					apiSpec = cr.Spec.Components.Flow
				}

				Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
				Eventually(func() error {
					return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
				}, "10s", "250ms").Should(Succeed())

				Expect(EnsureAstarteGenericAPIComponent(cr, apiSpec, component, k8sClient, scheme.Scheme)).To(Succeed())

				// Deployment should exist
				deploymentName := cr.Name + "-" + component.DashedString()
				dep := &appsv1.Deployment{}
				Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())
				Expect(dep.Labels).ToNot(BeNil())
				Expect(dep.Labels["app"]).To(Equal(deploymentName))
				Expect(dep.Labels["component"]).To(Equal("astarte"))
				Expect(dep.Labels["astarte-component"]).To(Equal(component.DashedString()))

				// Service should exist
				serviceName := cr.Name + "-" + component.ServiceName()
				svc := &v1.Service{}
				Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serviceName, Namespace: cr.Namespace}, svc)).To(Succeed())
				Expect(svc.Spec.Ports).To(HaveLen(1))
				Expect(svc.Spec.Ports[0].Port).To(Equal(int32(astarteServicesPort)))

				// Cookie secret should exist
				cookieSecret := &v1.Secret{}
				cookieSecretName := deploymentName + "-cookie"
				Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: cookieSecretName, Namespace: cr.Namespace}, cookieSecret)).To(Succeed())

				// Verify container configuration
				Expect(dep.Spec.Template.Spec.Containers).To(HaveLen(1))
				container := dep.Spec.Template.Spec.Containers[0]
				Expect(container.Name).To(Equal(component.DashedString()))
				Expect(container.Image).To(ContainSubstring(component.DockerImageName()))
				Expect(container.Ports).To(HaveLen(1))
				Expect(container.Ports[0].Name).To(Equal("http"))
				Expect(container.Ports[0].ContainerPort).To(Equal(int32(astarteServicesPort)))

				// Verify probes
				Expect(container.ReadinessProbe).ToNot(BeNil())
				Expect(container.LivenessProbe).ToNot(BeNil())
				Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/health"))
				Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/health"))

				if component == apiv2alpha1.Housekeeping {
					// Housekeeping should have longer failure threshold
					Expect(container.ReadinessProbe.FailureThreshold).To(Equal(int32(15)))
					Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(15)))

					// Housekeeping should have JWT public key volume
					hasJWTVolume := false
					for _, vol := range dep.Spec.Template.Spec.Volumes {
						if vol.Name == "jwtpubkey" {
							hasJWTVolume = true
							break
						}
					}
					Expect(hasJWTVolume).To(BeTrue())
				} else {
					// Other components should have standard failure threshold
					Expect(container.ReadinessProbe.FailureThreshold).To(Equal(int32(5)))
					Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(5)))
				}
			},
			Entry("AppEngineAPI", apiv2alpha1.AppEngineAPI),
			Entry("RealmManagement", apiv2alpha1.RealmManagement),
			Entry("Pairing", apiv2alpha1.Pairing),
			Entry("Housekeeping", apiv2alpha1.Housekeeping),
		)

		It("should not create deployment when component is disabled", func() {
			component := apiv2alpha1.AppEngineAPI
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(false)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())

			// Deployment should not exist
			deploymentName := cr.Name + "-" + component.DashedString()
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).ToNot(Succeed())
		})

		It("should delete existing deployment when disabling component", func() {
			component := apiv2alpha1.AppEngineAPI
			deploymentName := cr.Name + "-" + component.DashedString()

			// Enable first
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())

			// Verify deployment exists
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())

			// Now disable
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(false)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())

			// Deployment should be deleted
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)
			}, "10s", "250ms").ShouldNot(Succeed())
		})

		It("should apply custom resource requirements", func() {
			component := apiv2alpha1.AppEngineAPI
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
			cr.Spec.Components.AppengineAPI.Resources = &v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("100m"),
					v1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("500m"),
					v1.ResourceMemory: resource.MustParse("512Mi"),
				},
			}
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())

			deploymentName := cr.Name + "-" + component.DashedString()
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())

			container := dep.Spec.Template.Spec.Containers[0]
			Expect(container.Resources.Requests.Cpu().String()).To(Equal("100m"))
			Expect(container.Resources.Requests.Memory().String()).To(Equal("128Mi"))
			Expect(container.Resources.Limits.Cpu().String()).To(Equal("500m"))
			Expect(container.Resources.Limits.Memory().String()).To(Equal("512Mi"))
		})

		It("should configure custom replica count", func() {
			component := apiv2alpha1.AppEngineAPI
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
			cr.Spec.Components.AppengineAPI.Replicas = pointy.Int32(3)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())

			deploymentName := cr.Name + "-" + component.DashedString()
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())

			Expect(*dep.Spec.Replicas).To(Equal(int32(3)))
		})

		It("should enable authentication disable flag", func() {
			component := apiv2alpha1.AppEngineAPI
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
			cr.Spec.Components.AppengineAPI.DisableAuthentication = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())

			deploymentName := cr.Name + "-" + component.DashedString()
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())

			container := dep.Spec.Template.Spec.Containers[0]
			hasDisableAuthEnv := false
			for _, env := range container.Env {
				if env.Name == "APPENGINE_API_DISABLE_AUTHENTICATION" && env.Value == "true" {
					hasDisableAuthEnv = true
					break
				}
			}
			Expect(hasDisableAuthEnv).To(BeTrue())
		})

		It("should create ServiceAccount for FlowComponent", func() {

		})

		It("should create ServiceAccount for AppEngineAPI", func() {
			component := apiv2alpha1.AppEngineAPI
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())

			deploymentName := cr.Name + "-" + component.DashedString()

			// ServiceAccount should exist
			sa := &v1.ServiceAccount{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, sa)).To(Succeed())

			// Role should exist
			role := &rbacv1.Role{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, role)).To(Succeed())

			// RoleBinding should exist
			roleBinding := &rbacv1.RoleBinding{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, roleBinding)).To(Succeed())

			// Deployment should reference the ServiceAccount
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())
			Expect(dep.Spec.Template.Spec.ServiceAccountName).To(Equal(deploymentName))
		})

		It("should configure priority classes when enabled", func() {
			component := apiv2alpha1.AppEngineAPI
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
			cr.Spec.Components.AppengineAPI.PriorityClass = "high"
			cr.Spec.Features.AstartePodPriorities = &apiv2alpha1.AstartePodPrioritiesSpec{
				Enable:              true,
				AstarteHighPriority: pointy.Int(1000),
				AstarteMidPriority:  pointy.Int(500),
				AstarteLowPriority:  pointy.Int(100),
			}
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			// Create PriorityClasses first
			Expect(EnsureAstartePriorityClasses(cr, k8sClient, scheme.Scheme)).To(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())

			deploymentName := cr.Name + "-" + component.DashedString()
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())

			Expect(dep.Spec.Template.Spec.PriorityClassName).To(Equal(AstarteHighPriorityName))
		})
	})

	Describe("Test EnsureAstarteGenericAPIComponentWithCustomProbe", func() {
		It("should use custom probe when provided", func() {
			component := apiv2alpha1.AppEngineAPI
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			customProbe := &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/custom-health",
						Port: intstr.FromInt(8080),
					},
				},
				InitialDelaySeconds: 20,
				TimeoutSeconds:      10,
				PeriodSeconds:       60,
				FailureThreshold:    3,
			}

			Expect(EnsureAstarteGenericAPIComponentWithCustomProbe(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme, customProbe)).To(Succeed())

			deploymentName := cr.Name + "-" + component.DashedString()
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())

			container := dep.Spec.Template.Spec.Containers[0]
			Expect(container.ReadinessProbe).ToNot(BeNil())
			Expect(container.LivenessProbe).ToNot(BeNil())
			Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/custom-health"))
			Expect(container.ReadinessProbe.HTTPGet.Port.IntVal).To(Equal(int32(8080)))
			Expect(container.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(20)))
			Expect(container.ReadinessProbe.TimeoutSeconds).To(Equal(int32(10)))
			Expect(container.ReadinessProbe.PeriodSeconds).To(Equal(int32(60)))
			Expect(container.ReadinessProbe.FailureThreshold).To(Equal(int32(3)))
		})
	})

	Describe("Test checkShouldDeploy", func() {
		It("should return true for components that should deploy by default", func() {
			apiSpec := apiv2alpha1.AstarteGenericAPIComponentSpec{}
			component := apiv2alpha1.AppEngineAPI
			result := checkShouldDeploy(log, "test-deployment", cr, apiSpec, component, k8sClient)
			Expect(result).To(BeTrue())
		})

		It("should return false for FlowComponent by default", func() {
			apiSpec := apiv2alpha1.AstarteGenericAPIComponentSpec{}
			component := apiv2alpha1.FlowComponent
			result := checkShouldDeploy(log, "test-deployment", cr, apiSpec, component, k8sClient)
			Expect(result).To(BeFalse())
		})

		It("should return true for FlowComponent when explicitly enabled", func() {
			apiSpec := apiv2alpha1.AstarteGenericAPIComponentSpec{
				AstarteGenericClusteredResource: apiv2alpha1.AstarteGenericClusteredResource{
					Deploy: pointy.Bool(true),
				},
			}
			component := apiv2alpha1.FlowComponent
			result := checkShouldDeploy(log, "test-deployment", cr, apiSpec, component, k8sClient)
			Expect(result).To(BeTrue())
		})

		It("should return false when deploy is explicitly disabled", func() {
			apiSpec := apiv2alpha1.AstarteGenericAPIComponentSpec{
				AstarteGenericClusteredResource: apiv2alpha1.AstarteGenericClusteredResource{
					Deploy: pointy.Bool(false),
				},
			}
			component := apiv2alpha1.AppEngineAPI
			result := checkShouldDeploy(log, "test-deployment", cr, apiSpec, component, k8sClient)
			Expect(result).To(BeFalse())
		})
	})

	Describe("Test getAstarteAPIProbe", func() {
		It("should return custom probe when provided", func() {
			customProbe := &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/custom",
						Port: intstr.FromInt(9000),
					},
				},
			}
			result := getAstarteAPIProbe(apiv2alpha1.AppEngineAPI, customProbe)
			Expect(result).To(Equal(customProbe))
		})

		It("should return housekeeping probe for housekeeping component", func() {
			result := getAstarteAPIProbe(apiv2alpha1.Housekeeping, nil)
			Expect(result).ToNot(BeNil())
			Expect(result.HTTPGet.Path).To(Equal("/health"))
			Expect(result.FailureThreshold).To(Equal(int32(15)))
		})

		It("should return generic probe for other components", func() {
			result := getAstarteAPIProbe(apiv2alpha1.AppEngineAPI, nil)
			Expect(result).ToNot(BeNil())
			Expect(result.HTTPGet.Path).To(Equal("/health"))
			Expect(result.FailureThreshold).To(Equal(int32(5)))
		})
	})

	Describe("Test getAstarteGenericAPIComponentGenericProbe", func() {
		It("should create probe with correct defaults", func() {
			result := getAstarteGenericAPIComponentGenericProbe("/test")
			Expect(result).ToNot(BeNil())
			Expect(result.HTTPGet.Path).To(Equal("/test"))
			Expect(result.HTTPGet.Port.String()).To(Equal("http"))
			Expect(result.InitialDelaySeconds).To(Equal(int32(10)))
			Expect(result.TimeoutSeconds).To(Equal(int32(5)))
			Expect(result.PeriodSeconds).To(Equal(int32(30)))
			Expect(result.FailureThreshold).To(Equal(int32(5)))
		})
	})

	Describe("Test getAstarteGenericAPIComponentGenericProbeWithThreshold", func() {
		It("should create probe with custom threshold", func() {
			result := getAstarteGenericAPIComponentGenericProbeWithThreshold("/test", 10)
			Expect(result).ToNot(BeNil())
			Expect(result.HTTPGet.Path).To(Equal("/test"))
			Expect(result.FailureThreshold).To(Equal(int32(10)))
		})
	})

	Describe("Test component-specific environment variables", func() {
		It("should add proper environment variables for Housekeeping", func() {
			component := apiv2alpha1.Housekeeping
			cr.Spec.Components.Housekeeping.Deploy = pointy.Bool(true)
			cr.Spec.Features.RealmDeletion = true
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.Housekeeping, component, k8sClient, scheme.Scheme)).To(Succeed())

			deploymentName := cr.Name + "-" + component.DashedString()
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())

			container := dep.Spec.Template.Spec.Containers[0]

			// Check for Housekeeping-specific environment variables
			hasRealmDeletionEnv := false
			hasJWTPublicKeyPath := false
			for _, env := range container.Env {
				if env.Name == "HOUSEKEEPING_ENABLE_REALM_DELETION" && env.Value == "true" {
					hasRealmDeletionEnv = true
				}
				if env.Name == "HOUSEKEEPING_API_JWT_PUBLIC_KEY_PATH" && env.Value == "/jwtpubkey/public-key" {
					hasJWTPublicKeyPath = true
				}
			}
			Expect(hasRealmDeletionEnv).To(BeTrue())
			Expect(hasJWTPublicKeyPath).To(BeTrue())
		})

		It("should add proper environment variables for Pairing", func() {
			component := apiv2alpha1.Pairing
			cr.Spec.Components.Pairing.Deploy = pointy.Bool(true)
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.Pairing, component, k8sClient, scheme.Scheme)).To(Succeed())

			deploymentName := cr.Name + "-" + component.DashedString()
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())

			container := dep.Spec.Template.Spec.Containers[0]

			// Check for Pairing-specific environment variables
			hasCFSSLURL := false
			hasBrokerURL := false
			for _, env := range container.Env {
				if env.Name == "PAIRING_CFSSL_URL" {
					hasCFSSLURL = true
				}
				if env.Name == "PAIRING_BROKER_URL" {
					hasBrokerURL = true
				}
			}
			Expect(hasCFSSLURL).To(BeTrue())
			Expect(hasBrokerURL).To(BeTrue())
		})

		It("should add proper environment variables for Flow", func() {
			// Leave FlowComponent test here for reference
		})
	})

	Describe("Test Astarte instance ID support", func() {
		It("should add instance ID environment variable when specified", func() {
			component := apiv2alpha1.AppEngineAPI
			cr.Spec.Components.AppengineAPI.Deploy = pointy.Bool(true)
			cr.Spec.AstarteInstanceID = CustomAstarteInstanceID
			Expect(k8sClient.Update(context.Background(), cr)).To(Succeed())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: CustomAstarteName, Namespace: CustomAstarteNamespace}, cr)
			}, "10s", "250ms").Should(Succeed())

			Expect(EnsureAstarteGenericAPIComponent(cr, cr.Spec.Components.AppengineAPI.AstarteGenericAPIComponentSpec, component, k8sClient, scheme.Scheme)).To(Succeed())

			deploymentName := cr.Name + "-" + component.DashedString()
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: deploymentName, Namespace: cr.Namespace}, dep)).To(Succeed())

			container := dep.Spec.Template.Spec.Containers[0]
			hasInstanceID := false
			for _, env := range container.Env {
				if env.Name == "ASTARTE_INSTANCE_ID" && env.Value == CustomAstarteInstanceID {
					hasInstanceID = true
					break
				}
			}
			Expect(hasInstanceID).To(BeTrue())
		})
	})
})
