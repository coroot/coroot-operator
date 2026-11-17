Coroot Kubernetes/Openshift operator.

## Exposing Coroot through a Gateway API HTTPRoute (optional)

As an alternative to `spec.ingress`, Coroot can be exposed through a Gateway API `HTTPRoute` attached to an existing, user-managed `Gateway`. The operator creates and manages only the `HTTPRoute` (in the same namespace as the Coroot resource); the Gateway API CRDs, the Gateway itself, TLS certificates, and middlewares are user-managed.

- TLS terminates at the Gateway. `spec.httpRoute` is not supported together with `spec.httpDisabled: true`, and `spec.ingress` / `spec.httpRoute` are mutually exclusive.
- `parentRefs` may reference a Gateway in another namespace; the Gateway must explicitly allow the route via `spec.listeners[].allowedRoutes` (a `ReferenceGrant` does not replace `allowedRoutes` for routes pointing to Gateways).
- The route forwards matching requests to the operator-managed `<name>-coroot` service on its numeric HTTP port (default 8080). The operator never modifies or deletes a same-name `HTTPRoute` that is not controller-owned by this Coroot resource.
- An `ExtensionRef` filter requires the Gateway implementation's Kubernetes CRD provider to be enabled, e.g., `providers.kubernetesCRD` in Traefik static configuration (`providers.kubernetesCRD.enabled: true` in the Traefik Helm chart).
- If `path` is set, Coroot is started with the corresponding `URL_BASE_PATH`.
- If the Gateway API CRDs are not installed at operator startup, the operator skips creating the route and reports the Coroot resource as `Misconfigured` when `spec.httpRoute` is set. Installing the CRDs after startup requires an operator restart.
- Upgrading the operator while the Gateway API is installed requires applying the updated operator ClusterRole and Coroot CRDs alongside the new image (not an image-only upgrade): with the Gateway API present the operator starts an HTTPRoute watch, which fails to sync under an older ClusterRole without the `httproutes` rules.

Example — an existing cross-namespace Gateway with TLS terminated at the Gateway, and an optional implementation-specific filter (a Traefik Middleware in the same namespace as the route):

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: coroot-gateway
  namespace: infra
spec:
  gatewayClassName: traefik
  listeners:
  - name: https
    port: 443
    protocol: HTTPS
    hostname: coroot.example.com
    tls:
      mode: Terminate
      certificateRefs:
      - name: coroot-tls-cert
    allowedRoutes:
      namespaces:
        from: Selector
        selector:
          matchLabels:
            kubernetes.io/metadata.name: coroot # allow the Coroot HTTPRoute from this namespace
---
apiVersion: coroot.com/v1
kind: Coroot
metadata:
  name: coroot
  namespace: coroot
spec:
  httpRoute:
    parentRefs:
    - name: coroot-gateway
      namespace: infra
      sectionName: https
    hostnames:
    - coroot.example.com
    path: /coroot
    filters:
    - type: ExtensionRef
      extensionRef:
        group: traefik.io
        kind: Middleware
        name: coroot-middleware # existing Middleware in namespace "coroot"
```
