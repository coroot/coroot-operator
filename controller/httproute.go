package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corootv1 "github.io/coroot/operator/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// HTTPRouteAPIV1 is the Gateway API version the operator manages HTTPRoutes with.
const HTTPRouteAPIV1 = "gateway.networking.k8s.io/v1"

// ErrHTTPRouteAPIUnavailable is returned when spec.httpRoute is configured but the Gateway API
// HTTPRoute resource is not available in the cluster (Gateway API CRDs not installed).
var ErrHTTPRouteAPIUnavailable = errors.New("Gateway API (" + HTTPRouteAPIV1 + ") HTTPRoute is not available in this cluster; install the Gateway API CRDs")

// httpRouteValidationErrors reports configuration problems with spec.httpRoute.
// The returned errors must prevent the HTTPRoute from being created.
func httpRouteValidationErrors(cr *corootv1.Coroot) []string {
	hr := cr.Spec.HTTPRoute
	if hr == nil {
		return nil
	}
	var errs []string
	if cr.Spec.Ingress != nil {
		errs = append(errs, "spec.ingress and spec.httpRoute are mutually exclusive; configure only one")
	}
	if cr.Spec.HTTPDisabled {
		// The route backend would point to a plain HTTP service port that does not exist when
		// HTTP is disabled; refusing is clearer than silently routing to the HTTPS port.
		errs = append(errs, "spec.httpRoute is not supported with spec.httpDisabled=true: TLS terminates at the Gateway, keep plain HTTP enabled for the Coroot service")
	}
	if n := len(hr.ParentRefs); n < 1 || n > 32 {
		errs = append(errs, fmt.Sprintf("spec.httpRoute.parentRefs must contain between 1 and 32 references (got %d)", n))
	}
	for i, ref := range hr.ParentRefs {
		if ref.Name == "" {
			errs = append(errs, fmt.Sprintf("spec.httpRoute.parentRefs[%d].name is required", i))
			continue
		}
		if ref.Kind != nil && *ref.Kind != "Gateway" {
			errs = append(errs, fmt.Sprintf("spec.httpRoute.parentRefs[%d].kind must be \"Gateway\"", i))
		}
		if ref.Group != nil && *ref.Group != gatewayv1.GroupName {
			errs = append(errs, fmt.Sprintf("spec.httpRoute.parentRefs[%d].group must be %q", i, gatewayv1.GroupName))
		}
	}
	return errs
}

// corootHTTPRoute builds the HTTPRoute owned by the operator. It routes all matching
// requests to the operator-managed <name>-coroot service (numeric HTTP port).
func (r *CorootReconciler) corootHTTPRoute(cr *corootv1.Coroot) *gatewayv1.HTTPRoute {
	ls := Labels(cr, "httproute")
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}
	hr := cr.Spec.HTTPRoute
	if hr == nil {
		return route
	}
	// Deep copy so the built route never aliases the Coroot spec (parentRefs ports,
	// filter extension refs, nested header lists, ...).
	spec := hr.DeepCopy()
	route.Annotations = spec.Annotations

	path := spec.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	port := gatewayv1.PortNumber(cr.Spec.Service.Port)
	route.Spec = gatewayv1.HTTPRouteSpec{
		CommonRouteSpec: gatewayv1.CommonRouteSpec{
			ParentRefs: spec.ParentRefs,
		},
		Hostnames: spec.Hostnames,
		Rules: []gatewayv1.HTTPRouteRule{{
			Matches: []gatewayv1.HTTPRouteMatch{{
				Path: &gatewayv1.HTTPPathMatch{
					Type:  ptr.To(gatewayv1.PathMatchPathPrefix),
					Value: ptr.To(path),
				},
			}},
			Filters: spec.Filters,
			BackendRefs: []gatewayv1.HTTPBackendRef{{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: gatewayv1.ObjectName(fmt.Sprintf("%s-coroot", cr.Name)),
						Port: &port,
					},
				},
			}},
		}},
	}
	return route
}

// httpRouteOwnedBy reports whether the route is controller-owned by this exact Coroot
// (UID match; names can be reused, UIDs cannot).
func httpRouteOwnedBy(route *gatewayv1.HTTPRoute, cr *corootv1.Coroot) bool {
	for _, ref := range route.OwnerReferences {
		if ref.Controller != nil && *ref.Controller &&
			ref.UID == cr.UID &&
			ref.APIVersion == corootv1.GroupVersion.String() && ref.Kind == "Coroot" {
			return true
		}
	}
	return false
}

// reconcileHTTPRoute creates or updates (delete=true: removes) the owned HTTPRoute.
// Existing same-name routes not controller-owned by this Coroot are refused, never adopted,
// and never deleted. Errors are returned to the caller instead of being swallowed, so failed
// reconciliation is not silently reported as OK.
func (r *CorootReconciler) reconcileHTTPRoute(ctx context.Context, cr *corootv1.Coroot, delete bool) error {
	if !r.HTTPRouteAPIAvailable {
		if delete {
			return nil
		}
		return ErrHTTPRouteAPIUnavailable
	}
	logger := log.FromContext(ctx)
	if delete {
		existing := &gatewayv1.HTTPRoute{}
		err := r.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: cr.Name}, existing)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if !httpRouteOwnedBy(existing, cr) {
			logger.Info("skipping deletion of HTTPRoute not owned by this Coroot", "name", existing.Name)
			return nil
		}
		// Preconditions pin the exact object seen above, so a route replaced in between is not deleted.
		err = r.Delete(ctx, existing, client.Preconditions{UID: &existing.UID, ResourceVersion: &existing.ResourceVersion})
		if err != nil && !apierrors.IsNotFound(err) {
			// Conflict means the route changed between the ownership check and the delete;
			// requeue so ownership is re-evaluated before retrying.
			return err
		}
		logger.Info("deleted")
		return nil
	}
	route := r.corootHTTPRoute(cr)
	spec := route.Spec
	annotations := route.Annotations
	res, err := controllerutil.CreateOrUpdate(ctx, r.Client, route, func() error {
		// controllerutil.CreateOrUpdate fetched the live object into route before calling
		// this closure; ResourceVersion is empty only for a not-yet-created route. Refuse to
		// adopt or overwrite any existing route we do not controller-own.
		if route.ResourceVersion != "" && !httpRouteOwnedBy(route, cr) {
			return fmt.Errorf("HTTPRoute %s/%s exists and is not controller-owned by this Coroot; refusing to adopt", route.Namespace, route.Name)
		}
		if err := controllerutil.SetControllerReference(cr, route, r.Scheme); err != nil {
			return err
		}
		return MergeSpecs(route, &route.Spec, spec, annotations)
	})
	if err != nil {
		return err
	}
	if res != controllerutil.OperationResultNone {
		logger.Info(fmt.Sprintf("%s", res))
	}
	return nil
}
