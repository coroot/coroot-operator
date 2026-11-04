package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func newTestScheme(t *testing.T) *runtime.Scheme {
	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, corootv1.AddToScheme(s))
	require.NoError(t, gatewayv1.AddToScheme(s))
	return s
}

func newTestCorootReconciler(t *testing.T, options func(*CorootReconciler), objects ...client.Object) *CorootReconciler {
	s := newTestScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objects...).
		WithStatusSubresource(&corootv1.Coroot{}).
		Build()
	r := &CorootReconciler{
		Client:                c,
		Scheme:                s,
		instances:             map[ctrl.Request]bool{},
		versions:              map[App]string{},
		IngressAPIVersion:     IngressAPIV1,
		HTTPRouteAPIAvailable: true,
		RegistryConfig:        &RegistryConfig{},
	}
	if options != nil {
		options(r)
	}
	return r
}

func newTestCoroot() *corootv1.Coroot {
	return &corootv1.Coroot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coroot",
			Namespace: "coroot-ns",
			UID:       types.UID("test-uid"),
		},
	}
}

func nParentRefs(n int) []gatewayv1.ParentReference {
	refs := make([]gatewayv1.ParentReference, 0, n)
	for i := 0; i < n; i++ {
		refs = append(refs, gatewayv1.ParentReference{Name: gatewayv1.ObjectName(fmt.Sprintf("gw-%d", i))})
	}
	return refs
}

func httpRouteBackendRef(t *testing.T, route *gatewayv1.HTTPRoute) gatewayv1.BackendRef {
	t.Helper()
	require.Len(t, route.Spec.Rules, 1)
	require.Len(t, route.Spec.Rules[0].BackendRefs, 1)
	return route.Spec.Rules[0].BackendRefs[0].BackendRef
}

func httpRouteMatchPath(t *testing.T, route *gatewayv1.HTTPRoute) string {
	t.Helper()
	require.Len(t, route.Spec.Rules, 1)
	require.Len(t, route.Spec.Rules[0].Matches, 1)
	match := route.Spec.Rules[0].Matches[0]
	require.NotNil(t, match.Path)
	require.NotNil(t, match.Path.Value)
	return *match.Path.Value
}

func TestCorootHTTPRoute(t *testing.T) {
	r := &CorootReconciler{versions: map[App]string{}, RegistryConfig: &RegistryConfig{}}

	tests := []struct {
		name  string
		spec  corootv1.CorootSpec
		check func(t *testing.T, route *gatewayv1.HTTPRoute, cr *corootv1.Coroot)
	}{
		{
			name: "default service port",
			spec: corootv1.CorootSpec{
				Service:   corootv1.ServiceSpec{Port: 8080},
				HTTPRoute: &corootv1.HTTPRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "gateway"}}},
			},
			check: func(t *testing.T, route *gatewayv1.HTTPRoute, cr *corootv1.Coroot) {
				backend := httpRouteBackendRef(t, route)
				assert.Equal(t, "coroot-coroot", string(backend.Name))
				require.NotNil(t, backend.Port)
				assert.Equal(t, gatewayv1.PortNumber(8080), *backend.Port)
				assert.Equal(t, "/", httpRouteMatchPath(t, route))
				assert.NotNil(t, route.Spec.Rules[0].Matches[0].Path.Type)
				assert.Equal(t, gatewayv1.PathMatchPathPrefix, *route.Spec.Rules[0].Matches[0].Path.Type)
				assert.Len(t, route.Spec.Hostnames, 0)
				assert.Equal(t, "gateway", string(route.Spec.ParentRefs[0].Name))
			},
		},
		{
			name: "custom service port",
			spec: corootv1.CorootSpec{
				Service:   corootv1.ServiceSpec{Port: 9090},
				HTTPRoute: &corootv1.HTTPRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "gateway"}}},
			},
			check: func(t *testing.T, route *gatewayv1.HTTPRoute, cr *corootv1.Coroot) {
				backend := httpRouteBackendRef(t, route)
				require.NotNil(t, backend.Port)
				assert.Equal(t, gatewayv1.PortNumber(9090), *backend.Port)
			},
		},
		{
			name: "hostnames, cross-namespace parent ref and listener",
			spec: corootv1.CorootSpec{
				Service: corootv1.ServiceSpec{Port: 8080},
				HTTPRoute: &corootv1.HTTPRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{{
						Name:        "coroot-gateway",
						Namespace:   ptr.To(gatewayv1.Namespace("infra")),
						SectionName: ptr.To(gatewayv1.SectionName("https")),
					}},
					Hostnames: []gatewayv1.Hostname{"coroot.example.com", "coroot-2.example.com"},
				},
			},
			check: func(t *testing.T, route *gatewayv1.HTTPRoute, cr *corootv1.Coroot) {
				assert.Equal(t, []gatewayv1.Hostname{"coroot.example.com", "coroot-2.example.com"}, route.Spec.Hostnames)
				ref := route.Spec.ParentRefs[0]
				assert.Equal(t, "coroot-gateway", string(ref.Name))
				require.NotNil(t, ref.Namespace)
				assert.Equal(t, "infra", string(*ref.Namespace))
				require.NotNil(t, ref.SectionName)
				assert.Equal(t, "https", string(*ref.SectionName))
			},
		},
		{
			name: "path normalization",
			spec: corootv1.CorootSpec{
				Service: corootv1.ServiceSpec{Port: 8080},
				HTTPRoute: &corootv1.HTTPRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{{Name: "gateway"}},
					Path:       "coroot",
				},
			},
			check: func(t *testing.T, route *gatewayv1.HTTPRoute, cr *corootv1.Coroot) {
				assert.Equal(t, "/coroot", httpRouteMatchPath(t, route))
			},
		},
		{
			name: "filters passed through",
			spec: corootv1.CorootSpec{
				Service: corootv1.ServiceSpec{Port: 8080},
				HTTPRoute: &corootv1.HTTPRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{{Name: "gateway"}},
					Filters: []gatewayv1.HTTPRouteFilter{
						{
							Type: gatewayv1.HTTPRouteFilterType("ExtensionRef"),
							ExtensionRef: &gatewayv1.LocalObjectReference{
								Group: "traefik.io",
								Kind:  "Middleware",
								Name:  "coroot-middleware",
							},
						},
						{
							Type: gatewayv1.HTTPRouteFilterType("RequestHeaderModifier"),
							RequestHeaderModifier: &gatewayv1.HTTPHeaderFilter{
								Set: []gatewayv1.HTTPHeader{{Name: "X-Forwarded-Prefix", Value: "/coroot"}},
							},
						},
					},
				},
			},
			check: func(t *testing.T, route *gatewayv1.HTTPRoute, cr *corootv1.Coroot) {
				filters := route.Spec.Rules[0].Filters
				require.Len(t, filters, 2)
				assert.Equal(t, gatewayv1.HTTPRouteFilterType("ExtensionRef"), filters[0].Type)
				require.NotNil(t, filters[0].ExtensionRef)
				assert.Equal(t, gatewayv1.Group("traefik.io"), filters[0].ExtensionRef.Group)
				assert.Equal(t, gatewayv1.Kind("Middleware"), filters[0].ExtensionRef.Kind)
				assert.Equal(t, gatewayv1.ObjectName("coroot-middleware"), filters[0].ExtensionRef.Name)
				assert.Equal(t, gatewayv1.HTTPRouteFilterType("RequestHeaderModifier"), filters[1].Type)
				require.NotNil(t, filters[1].RequestHeaderModifier)
				assert.Equal(t, "X-Forwarded-Prefix", string(filters[1].RequestHeaderModifier.Set[0].Name))
			},
		},
		{
			name: "built route does not alias nested spec fields",
			spec: corootv1.CorootSpec{
				Service: corootv1.ServiceSpec{Port: 8080},
				HTTPRoute: &corootv1.HTTPRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{{
						Name: "gateway",
						Port: ptr.To(gatewayv1.PortNumber(443)),
					}},
					Filters: []gatewayv1.HTTPRouteFilter{{
						Type:                  gatewayv1.HTTPRouteFilterType("RequestHeaderModifier"),
						RequestHeaderModifier: &gatewayv1.HTTPHeaderFilter{Set: []gatewayv1.HTTPHeader{{Name: "X-One", Value: "1"}}},
					}},
				},
			},
			check: func(t *testing.T, route *gatewayv1.HTTPRoute, cr *corootv1.Coroot) {
				ref := route.Spec.ParentRefs[0]
				require.NotNil(t, ref.Port)
				assert.Equal(t, gatewayv1.PortNumber(443), *ref.Port)
				routeHeaders := route.Spec.Rules[0].Filters[0].RequestHeaderModifier.Set
				require.NotNil(t, routeHeaders)

				// Mutate the original spec, including pointer-backed parent ref fields and nested filter slices.
				*cr.Spec.HTTPRoute.ParentRefs[0].Port = 8443
				cr.Spec.HTTPRoute.Filters[0].RequestHeaderModifier.Set = append(
					cr.Spec.HTTPRoute.Filters[0].RequestHeaderModifier.Set,
					gatewayv1.HTTPHeader{Name: "X-Two", Value: "2"},
				)
				cr.Spec.HTTPRoute.Filters[0].RequestHeaderModifier.Set[0].Value = "changed"
				cr.Spec.HTTPRoute.Filters[0].RequestHeaderModifier = &gatewayv1.HTTPHeaderFilter{}

				assert.Equal(t, gatewayv1.PortNumber(443), *ref.Port, "parent ref port mutation must not affect the route")
				assert.Len(t, routeHeaders, 1, "nested filter slice must not alias the spec")
				assert.Equal(t, "1", routeHeaders[0].Value)
			},
		},
		{
			name: "annotations do not alias the spec",
			spec: corootv1.CorootSpec{
				Service: corootv1.ServiceSpec{Port: 8080},
				HTTPRoute: &corootv1.HTTPRouteSpec{
					ParentRefs:  []gatewayv1.ParentReference{{Name: "gateway"}},
					Annotations: map[string]string{"a": "1"},
				},
			},
			check: func(t *testing.T, route *gatewayv1.HTTPRoute, cr *corootv1.Coroot) {
				assert.Equal(t, map[string]string{"a": "1"}, route.Annotations)
				cr.Spec.HTTPRoute.Annotations["a"] = "2"
				cr.Spec.HTTPRoute.Annotations["b"] = "3"
				assert.Equal(t, map[string]string{"a": "1"}, route.Annotations, "mutating spec must not affect the built route")
				route.Annotations["a"] = "3"
				assert.Equal(t, map[string]string{"a": "2", "b": "3"}, cr.Spec.HTTPRoute.Annotations, "mutating the route must not affect the spec")
			},
		},
		{
			name: "nil httpRoute spec",
			spec: corootv1.CorootSpec{
				Service: corootv1.ServiceSpec{Port: 8080},
			},
			check: func(t *testing.T, route *gatewayv1.HTTPRoute, cr *corootv1.Coroot) {
				assert.Equal(t, "coroot", route.Name)
				assert.Equal(t, "coroot-ns", route.Namespace)
				assert.Empty(t, route.Spec.Rules)
				assert.Empty(t, route.Spec.ParentRefs)
				assert.Nil(t, route.Annotations)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cr := newTestCoroot()
			cr.Spec = tc.spec
			route := r.corootHTTPRoute(cr)
			assert.Equal(t, "coroot", route.Name)
			assert.Equal(t, "coroot-ns", route.Namespace)
			assert.Equal(t, map[string]string{
				"app.kubernetes.io/managed-by": "coroot-operator",
				"app.kubernetes.io/part-of":    "coroot",
				"app.kubernetes.io/component":  "httproute",
			}, route.Labels)
			tc.check(t, route, cr)
		})
	}
}

func TestHTTPRouteValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		spec   corootv1.CorootSpec
		errors []string
	}{
		{
			name: "no httpRoute",
		},
		{
			name: "valid same-namespace gateway",
			spec: corootv1.CorootSpec{
				HTTPRoute: &corootv1.HTTPRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "gateway"}}},
			},
		},
		{
			name: "valid cross-namespace gateway with explicit group",
			spec: corootv1.CorootSpec{
				HTTPRoute: &corootv1.HTTPRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{{
						Name:      "gateway",
						Namespace: ptr.To(gatewayv1.Namespace("infra")),
						Group:     ptr.To(gatewayv1.Group(gatewayv1.GroupName)),
					}},
				},
			},
		},
		{
			name: "ingress and httpRoute conflict",
			spec: corootv1.CorootSpec{
				Ingress:   &corootv1.IngressSpec{Host: "coroot.example.com"},
				HTTPRoute: &corootv1.HTTPRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "gateway"}}},
			},
			errors: []string{"spec.ingress and spec.httpRoute are mutually exclusive; configure only one"},
		},
		{
			name: "no parentRefs",
			spec: corootv1.CorootSpec{
				HTTPRoute: &corootv1.HTTPRouteSpec{ParentRefs: []gatewayv1.ParentReference{}},
			},
			errors: []string{"spec.httpRoute.parentRefs must contain between 1 and 32 references (got 0)"},
		},
		{
			name: "too many parentRefs",
			spec: corootv1.CorootSpec{
				HTTPRoute: &corootv1.HTTPRouteSpec{ParentRefs: nParentRefs(33)},
			},
			errors: []string{"spec.httpRoute.parentRefs must contain between 1 and 32 references (got 33)"},
		},
		{
			name: "parentRef without name",
			spec: corootv1.CorootSpec{
				HTTPRoute: &corootv1.HTTPRouteSpec{ParentRefs: []gatewayv1.ParentReference{{}}},
			},
			errors: []string{"spec.httpRoute.parentRefs[0].name is required"},
		},
		{
			name: "httpDisabled",
			spec: corootv1.CorootSpec{
				HTTPDisabled: true,
				HTTPRoute:    &corootv1.HTTPRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "gateway"}}},
			},
			errors: []string{"spec.httpRoute is not supported with spec.httpDisabled=true: TLS terminates at the Gateway, keep plain HTTP enabled for the Coroot service"},
		},
		{
			name: "invalid parent kind",
			spec: corootv1.CorootSpec{
				HTTPRoute: &corootv1.HTTPRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{{Name: "svc", Kind: ptr.To(gatewayv1.Kind("Service"))}},
				},
			},
			errors: []string{`spec.httpRoute.parentRefs[0].kind must be "Gateway"`},
		},
		{
			name: "invalid parent group",
			spec: corootv1.CorootSpec{
				HTTPRoute: &corootv1.HTTPRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{{Name: "gw", Group: ptr.To(gatewayv1.Group("foo.example.com"))}},
				},
			},
			errors: []string{`spec.httpRoute.parentRefs[0].group must be "gateway.networking.k8s.io"`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cr := newTestCoroot()
			cr.Spec = tc.spec
			assert.Equal(t, tc.errors, httpRouteValidationErrors(cr))
		})
	}
}

func TestCorootStatefulSetURLBasePath(t *testing.T) {
	r := &CorootReconciler{versions: map[App]string{}, RegistryConfig: &RegistryConfig{}}

	getURLBasePath := func(t *testing.T, cr *corootv1.Coroot) (string, bool) {
		t.Helper()
		ss := r.corootStatefulSet(cr, ConfigEnvs{}, "hash")
		require.Len(t, ss.Spec.Template.Spec.Containers, 1)
		for _, env := range ss.Spec.Template.Spec.Containers[0].Env {
			if env.Name == "URL_BASE_PATH" {
				return env.Value, true
			}
		}
		return "", false
	}

	t.Run("no URL_BASE_PATH without path", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec.Service.Port = 8080
		cr.Spec.HTTPRoute = &corootv1.HTTPRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "gateway"}}}
		val, found := getURLBasePath(t, cr)
		assert.False(t, found)
		assert.Empty(t, val)
	})

	t.Run("httpRoute subpath", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec.Service.Port = 8080
		cr.Spec.HTTPRoute = &corootv1.HTTPRouteSpec{
			ParentRefs: []gatewayv1.ParentReference{{Name: "gateway"}},
			Path:       "coroot",
		}
		val, found := getURLBasePath(t, cr)
		assert.True(t, found)
		assert.Equal(t, "coroot", val)
	})

	t.Run("ingress takes precedence", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec.Service.Port = 8080
		cr.Spec.Ingress = &corootv1.IngressSpec{Host: "coroot.example.com", Path: "/ingress"}
		cr.Spec.HTTPRoute = &corootv1.HTTPRouteSpec{
			ParentRefs: []gatewayv1.ParentReference{{Name: "gateway"}},
			Path:       "/httproute",
		}
		val, found := getURLBasePath(t, cr)
		assert.True(t, found)
		assert.Equal(t, "/ingress", val)
	})
}

func TestReconcileHTTPRoute(t *testing.T) {
	req := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "coroot-ns", Name: "coroot"}}

	validSpec := func() corootv1.CorootSpec {
		return corootv1.CorootSpec{
			Service: corootv1.ServiceSpec{Port: 8080},
			HTTPRoute: &corootv1.HTTPRouteSpec{
				ParentRefs:  []gatewayv1.ParentReference{{Name: "gateway"}},
				Hostnames:   []gatewayv1.Hostname{"coroot.example.com"},
				Path:        "/coroot",
				Annotations: map[string]string{"example.com/anno": "v"},
			},
		}
	}

	t.Run("create, owner ref, multi-pass and no spec mutation", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec = validSpec()
		r := newTestCorootReconciler(t, nil, cr)

		_, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)

		route := &gatewayv1.HTTPRoute{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, route))
		assert.Equal(t, []gatewayv1.Hostname{"coroot.example.com"}, route.Spec.Hostnames)
		assert.Equal(t, "/coroot", httpRouteMatchPath(t, route))
		assert.Equal(t, "v", route.Annotations["example.com/anno"])
		// user annotations are applied together with the operator's last-applied bookkeeping annotation
		_, hasLastApplied := route.Annotations[AnnotationLastAppliedConfiguration]
		assert.True(t, hasLastApplied)
		// owner ref set to the Coroot CR
		require.Len(t, route.OwnerReferences, 1)
		assert.Equal(t, cr.Name, route.OwnerReferences[0].Name)
		require.NotNil(t, route.OwnerReferences[0].Controller)
		assert.True(t, *route.OwnerReferences[0].Controller)

		statusCoroot := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, statusCoroot))
		assert.Equal(t, "OK", statusCoroot.Status.Status)

		// the spec persisted in the cluster must remain exactly as submitted
		firstStored := statusCoroot.Spec
		_, err = r.Reconcile(context.Background(), req)
		require.NoError(t, err)

		stored := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, stored))
		assert.Equal(t, firstStored, stored.Spec, "spec must not be mutated by reconciliation")

		require.NoError(t, r.Get(context.Background(), req.NamespacedName, route))
		assert.Equal(t, []gatewayv1.Hostname{"coroot.example.com"}, route.Spec.Hostnames)
		assert.Equal(t, "/coroot", httpRouteMatchPath(t, route))
		assert.Equal(t, "v", route.Annotations["example.com/anno"])
	})

	t.Run("update", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec = validSpec()
		r := newTestCorootReconciler(t, nil, cr)
		_, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)

		require.NoError(t, r.Get(context.Background(), req.NamespacedName, cr))
		cr.Spec.HTTPRoute.Hostnames = []gatewayv1.Hostname{"other.example.com"}
		cr.Spec.HTTPRoute.Path = ""
		require.NoError(t, r.Update(context.Background(), cr))
		_, err = r.Reconcile(context.Background(), req)
		require.NoError(t, err)

		route := &gatewayv1.HTTPRoute{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, route))
		assert.Equal(t, []gatewayv1.Hostname{"other.example.com"}, route.Spec.Hostnames)
		assert.Equal(t, "/", httpRouteMatchPath(t, route))
	})

	t.Run("delete when spec removed", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec = validSpec()
		r := newTestCorootReconciler(t, nil, cr)
		_, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)

		require.NoError(t, r.Get(context.Background(), req.NamespacedName, cr))
		cr.Spec.HTTPRoute = nil
		require.NoError(t, r.Update(context.Background(), cr))
		_, err = r.Reconcile(context.Background(), req)
		require.NoError(t, err)

		route := &gatewayv1.HTTPRoute{}
		err = r.Get(context.Background(), req.NamespacedName, route)
		assert.Error(t, err)
	})

	t.Run("no spec does not create a route", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec.Service.Port = 8080
		r := newTestCorootReconciler(t, nil, cr)
		_, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)

		route := &gatewayv1.HTTPRoute{}
		err = r.Get(context.Background(), req.NamespacedName, route)
		assert.Error(t, err)
	})

	t.Run("invalid config does not create the route and reports Misconfigured", func(t *testing.T) {
		spec := validSpec()
		spec.Ingress = &corootv1.IngressSpec{Host: "coroot.example.com"}
		cr := newTestCoroot()
		cr.Spec = spec
		r := newTestCorootReconciler(t, nil, cr)

		result, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, time.Minute, result.RequeueAfter)

		statusCoroot := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, statusCoroot))
		assert.Equal(t, "Misconfigured", statusCoroot.Status.Status)
		assert.Equal(t, []string{"spec.ingress and spec.httpRoute are mutually exclusive; configure only one"}, statusCoroot.Status.Errors)

		route := &gatewayv1.HTTPRoute{}
		err = r.Get(context.Background(), req.NamespacedName, route)
		assert.Error(t, err)
	})

	t.Run("missing Gateway API with spec reports Misconfigured", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec = validSpec()
		r := newTestCorootReconciler(t, func(r *CorootReconciler) {
			r.HTTPRouteAPIAvailable = false
		}, cr)

		result, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, time.Minute, result.RequeueAfter)

		statusCoroot := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, statusCoroot))
		assert.Equal(t, "Misconfigured", statusCoroot.Status.Status)
		require.Len(t, statusCoroot.Status.Errors, 1)
		assert.Contains(t, statusCoroot.Status.Errors[0], "Gateway API (gateway.networking.k8s.io/v1) HTTPRoute is not available")

		route := &gatewayv1.HTTPRoute{}
		err = r.Get(context.Background(), req.NamespacedName, route)
		assert.Error(t, err)
	})

	t.Run("missing Gateway API without spec is not an error", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec.Service.Port = 8080
		r := newTestCorootReconciler(t, func(r *CorootReconciler) {
			r.HTTPRouteAPIAvailable = false
		}, cr)

		_, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)

		statusCoroot := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, statusCoroot))
		assert.Equal(t, "OK", statusCoroot.Status.Status)
	})

	t.Run("same-name route not owned by Coroot is preserved when spec is nil", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec.Service.Port = 8080
		foreign := &gatewayv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "coroot",
				Namespace:   "coroot-ns",
				Annotations: map[string]string{"foreign": "1"},
			},
		}
		r := newTestCorootReconciler(t, nil, cr, foreign)
		_, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)

		got := &gatewayv1.HTTPRoute{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, got))
		assert.Equal(t, map[string]string{"foreign": "1"}, got.Annotations)
	})

	t.Run("route owned by another controller is preserved when spec is nil", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec.Service.Port = 8080
		foreign := &gatewayv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "coroot",
				Namespace: "coroot-ns",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "batch/v1",
					Kind:       "Job",
					Name:       "other",
					UID:        "other-uid",
					Controller: ptr.To(true),
				}},
			},
		}
		r := newTestCorootReconciler(t, nil, cr, foreign)
		_, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)

		got := &gatewayv1.HTTPRoute{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, got))
		require.Len(t, got.OwnerReferences, 1)
		assert.Equal(t, "other-uid", string(got.OwnerReferences[0].UID))
	})

	t.Run("route owned by another controller is not adopted or mutated", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec = validSpec()
		foreign := &gatewayv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "coroot",
				Namespace: "coroot-ns",
				Annotations: map[string]string{
					"foreign": "1",
				},
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "batch/v1",
					Kind:       "Job",
					Name:       "other",
					UID:        "other-uid",
					Controller: ptr.To(true),
				}},
			},
		}
		r := newTestCorootReconciler(t, nil, cr, foreign)
		_, err := r.Reconcile(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not controller-owned by this Coroot")

		got := &gatewayv1.HTTPRoute{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, got))
		assert.Equal(t, map[string]string{"foreign": "1"}, got.Annotations, "foreign route must not be mutated")
		require.Empty(t, got.Spec.Rules)

		statusCoroot := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, statusCoroot))
		assert.Equal(t, "Misconfigured", statusCoroot.Status.Status)
	})

	t.Run("unowned route with spec set is refused, unchanged and Misconfigured", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec = validSpec()
		unowned := &gatewayv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "coroot",
				Namespace:   "coroot-ns",
				Annotations: map[string]string{"unowned": "1"},
			},
		}
		r := newTestCorootReconciler(t, nil, cr, unowned)
		_, err := r.Reconcile(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not controller-owned by this Coroot")

		got := &gatewayv1.HTTPRoute{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, got))
		assert.Equal(t, map[string]string{"unowned": "1"}, got.Annotations, "unowned route must not be mutated")
		require.Empty(t, got.Spec.Rules)
		require.Empty(t, got.OwnerReferences)

		statusCoroot := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, statusCoroot))
		assert.Equal(t, "Misconfigured", statusCoroot.Status.Status)
	})

	t.Run("route create denied does not stop other reconciliation", func(t *testing.T) {
		// Regression: route failures must not abort the reconcile before
		// Prometheus/ClickHouse/RBAC/retained resources are processed.
		cr := newTestCoroot()
		cr.Spec = validSpec()
		cr.Spec.Storage.ReclaimPolicy = corev1.PersistentVolumeReclaimRetain
		cr.Spec.Projects = []corootv1.ProjectSpec{{
			Name: "production",
			ApiKeys: []corootv1.ApiKeySpec{{
				KeySecret: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "apikey-secret"}, Key: "key"},
			}},
		}}
		apiKeySecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "apikey-secret", Namespace: "coroot-ns"},
			Data:       map[string][]byte{"key": []byte("initial")},
		}
		s := newTestScheme(t)
		c := fake.NewClientBuilder().
			WithScheme(s).
			WithObjects(cr, apiKeySecret).
			WithStatusSubresource(&corootv1.Coroot{}).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if _, ok := obj.(*gatewayv1.HTTPRoute); ok {
						return fmt.Errorf("denied create")
					}
					return client.Create(ctx, obj, opts...)
				},
			}).
			Build()
		r := &CorootReconciler{
			Client:                c,
			Scheme:                s,
			instances:             map[ctrl.Request]bool{},
			versions:              map[App]string{},
			IngressAPIVersion:     IngressAPIV1,
			HTTPRouteAPIAvailable: true,
			RegistryConfig:        &RegistryConfig{},
		}

		_, err := r.Reconcile(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied create")

		// ClickHouse StatefulSet still created
		clickhouse := &appsv1.StatefulSet{}
		require.NoError(t, r.Get(context.Background(), types.NamespacedName{Namespace: "coroot-ns", Name: "coroot-clickhouse-shard-0"}, clickhouse))

		// cluster-agent ClusterRole (retain=true path) still created
		clusterRole := &rbacv1.ClusterRole{}
		require.NoError(t, r.Get(context.Background(), types.NamespacedName{Name: "coroot-cluster-agent"}, clusterRole))

		// Coroot PVC (retain=true path) still created, without a controller ref
		pvc := &corev1.PersistentVolumeClaim{}
		require.NoError(t, r.Get(context.Background(), types.NamespacedName{Namespace: "coroot-ns", Name: "data-coroot-coroot-0"}, pvc))
		assert.Empty(t, pvc.OwnerReferences, "retained resources must not carry controller refs")

		// API key Secret (retain=true path) still updated
		secret := &corev1.Secret{}
		require.NoError(t, r.Get(context.Background(), types.NamespacedName{Namespace: "coroot-ns", Name: "apikey-secret"}, secret))
		_, ok := secret.Data["key"]
		assert.True(t, ok)

		// status must reflect the failure, not a stale OK
		statusCoroot := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, statusCoroot))
		assert.Equal(t, "Misconfigured", statusCoroot.Status.Status)
		require.Len(t, statusCoroot.Status.Errors, 1)
		assert.Contains(t, statusCoroot.Status.Errors[0], "denied create")
	})

	t.Run("previously OK CR reports Misconfigured when route update is denied", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec = validSpec()
		s := newTestScheme(t)
		c := fake.NewClientBuilder().
			WithScheme(s).
			WithObjects(cr).
			WithStatusSubresource(&corootv1.Coroot{}).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					if _, ok := obj.(*gatewayv1.HTTPRoute); ok {
						return fmt.Errorf("denied update")
					}
					return client.Update(ctx, obj, opts...)
				},
			}).
			Build()
		r := &CorootReconciler{
			Client:                c,
			Scheme:                s,
			instances:             map[ctrl.Request]bool{},
			versions:              map[App]string{},
			IngressAPIVersion:     IngressAPIV1,
			HTTPRouteAPIAvailable: true,
			RegistryConfig:        &RegistryConfig{},
		}

		// first pass: route created, CR is OK
		_, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)
		okCoroot := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, okCoroot))
		assert.Equal(t, "OK", okCoroot.Status.Status)

		// user changes the spec
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, cr))
		cr.Spec.HTTPRoute.Hostnames = []gatewayv1.Hostname{"denied.example.com"}
		require.NoError(t, r.Update(context.Background(), cr))

		// second pass: route update denied
		_, err = r.Reconcile(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied update")

		route := &gatewayv1.HTTPRoute{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, route))
		assert.Equal(t, []gatewayv1.Hostname{"coroot.example.com"}, route.Spec.Hostnames, "route must keep the previously applied spec")

		statusCoroot := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, statusCoroot))
		assert.Equal(t, "Misconfigured", statusCoroot.Status.Status)
		require.Len(t, statusCoroot.Status.Errors, 1)
		assert.Contains(t, statusCoroot.Status.Errors[0], "denied update")
	})

	t.Run("create error is propagated, not reported as OK", func(t *testing.T) {
		cr := newTestCoroot()
		cr.Spec = validSpec()
		s := newTestScheme(t)
		c := fake.NewClientBuilder().
			WithScheme(s).
			WithObjects(cr).
			WithStatusSubresource(&corootv1.Coroot{}).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if _, ok := obj.(*gatewayv1.HTTPRoute); ok {
						return fmt.Errorf("denied create")
					}
					return client.Create(ctx, obj, opts...)
				},
			}).
			Build()
		r := &CorootReconciler{
			Client:                c,
			Scheme:                s,
			instances:             map[ctrl.Request]bool{},
			versions:              map[App]string{},
			IngressAPIVersion:     IngressAPIV1,
			HTTPRouteAPIAvailable: true,
			RegistryConfig:        &RegistryConfig{},
		}

		_, err := r.Reconcile(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied create")

		statusCoroot := &corootv1.Coroot{}
		require.NoError(t, r.Get(context.Background(), req.NamespacedName, statusCoroot))
		assert.NotEqual(t, "OK", statusCoroot.Status.Status)
	})
}
