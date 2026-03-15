# Architecture

**Analysis Date:** 2026-03-15

## Pattern Overview

**Overall:** GitOps-driven declarative infrastructure — no imperative runtime state management. The repository IS the desired state. Changes are committed to git, ArgoCD reconciles the cluster.

**Key Characteristics:**
- Single source of truth: everything in git, ArgoCD enforces convergence
- Helm-only deployments — no Kustomize, no raw manifests applied directly
- Layered provisioning: bare metal (Ansible) → Kubernetes (K3s) → external infra (OpenTofu) → workloads (ArgoCD)
- Secrets live outside git; injected at runtime via External Secrets Operator
- All public-facing traffic routes through a Cloudflare Tunnel (no ports exposed directly)

## Layers

**Layer 1 — Bare Metal Provisioning:**
- Purpose: PXE boot and OS install for the physical host, then K3s + Cilium install
- Location: `metal/`
- Contains: Ansible playbooks and roles
- Depends on: Physical hardware, SSH access
- Used by: Everything above it

**Layer 2 — External Infrastructure:**
- Purpose: Cloudflare DNS records, tunnel credentials, API tokens injected as Kubernetes Secrets
- Location: `external/`
- Contains: OpenTofu (Terraform-compatible) root module and sub-modules
- Depends on: Running Kubernetes cluster (writes secrets directly via kubernetes provider)
- Used by: `system/cloudflared`, `system/cert-manager`, `system/external-dns`, `system/cloudflare-ddns`, `system/cf-switch`

**Layer 3 — System Services:**
- Purpose: Core cluster infrastructure that all workloads depend on
- Location: `system/`
- Contains: One Helm wrapper chart per service (Chart.yaml + values.yaml + optional templates/)
- Depends on: Layer 1 and Layer 2
- Used by: Platform and App layers

**Layer 4 — Platform Services:**
- Purpose: Cross-cutting concerns: identity, secrets management, observability, automation
- Location: `platform/`
- Contains: One Helm wrapper chart per service
- Depends on: Layer 3 (ingress, cert-manager, storage, external-secrets)
- Used by: App layer (all apps authenticate via Dex/Kanidm)

**Layer 5 — Applications:**
- Purpose: End-user workloads
- Location: `apps/`
- Contains: One Helm wrapper chart per app
- Depends on: All lower layers
- Used by: End users

## Data Flow

**Secret Provisioning Flow:**

1. OpenTofu (`external/`) creates Cloudflare API tokens and Cloudflare tunnel credentials
2. OpenTofu writes those as `kubernetes_secret_v1` resources directly into the cluster (e.g., `cloudflared` namespace, `cert-manager` namespace)
3. For app-level secrets, the `global-secrets` ClusterSecretStore (`platform/global-secrets/`) uses the External Secrets Operator to pull secrets from a Kubernetes Secret in the same namespace
4. Each app declares `ExternalSecret` resources (in `templates/secret.yaml`) referencing `ClusterSecretStore/global-secrets`
5. The ESO controller materializes secrets into the target namespace at runtime

**Request Ingress Flow:**

1. User request hits Cloudflare → proxied or tunneled via `cloudflared` (Cloudflare Tunnel)
2. Cloudflare Tunnel terminates inside the cluster in `system/cloudflared`
3. Traffic reaches `ingress-nginx` LoadBalancer (Cilium L2 IP 192.168.1.4/30)
4. NGINX routes to the correct service based on hostname
5. TLS terminates at NGINX; cert-manager issues Let's Encrypt certificates via DNS-01 challenge against Cloudflare

**GitOps Reconciliation Flow:**

1. Developer commits to `master` branch of `github.com/meyeringh/homelab`
2. ArgoCD ApplicationSet (defined in `system/argocd/values.yaml`) watches `system/*`, `platform/*`, `apps/*`
3. For each directory it finds, ArgoCD creates an Application with namespace = directory name
4. ArgoCD runs `helm template` on the directory and applies the diff with server-side apply
5. `syncPolicy.automated` with `prune: true` and `selfHeal: true` ensures cluster matches git

**DNS Propagation Flow:**

1. Service deploys with `external-dns.alpha.kubernetes.io/target` annotation on Ingress
2. `external-dns` controller detects the Ingress and creates/updates CNAME records in Cloudflare
3. For dynamic home IP: `cloudflare-ddns` periodically updates the A record for the base domain

**SSO Flow:**

1. User accesses app → redirected to Dex (`dex.meyeringh.org`)
2. Dex acts as OIDC broker → connects upstream to Kanidm (`auth.meyeringh.org`) via OIDC
3. Kanidm authenticates the user and returns groups
4. Dex issues a token to the application
5. App validates token against Dex's JWKS endpoint

## Key Abstractions

**Helm Wrapper Chart:**
- Purpose: Every deployed service is a thin Helm chart that declares upstream chart dependencies and provides `values.yaml` overrides. No chart contains business logic.
- Examples: `system/argocd/Chart.yaml`, `apps/nextcloud/Chart.yaml`, `platform/kanidm/Chart.yaml`
- Pattern: `apiVersion: v2`, `version: 0.0.0`, one or more `dependencies` entries pointing to upstream charts

**app-template Pattern:**
- Purpose: Services without a dedicated Helm chart use the `bjw-s-labs/app-template` chart as a generic Kubernetes app wrapper
- Examples: `apps/vaultwarden/Chart.yaml`, `apps/jellyfin/Chart.yaml`, `platform/kanidm/Chart.yaml`, `system/cloudflare-ddns/Chart.yaml`
- Pattern: Single dependency on `app-template` v4.6.2 from `https://bjw-s-labs.github.io/helm-charts`; configuration entirely in `values.yaml`

**ExternalSecret:**
- Purpose: Declarative reference to a secret value stored elsewhere; ESO operator materializes actual Kubernetes Secrets
- Examples: `apps/nextcloud/templates/secret.yaml`, `apps/vaultwarden/templates/secret.yaml`
- Pattern: `ClusterSecretStore/global-secrets` is the standard store reference; `dataFrom.extract.key` uses dot-notation namespace (`app.component`)

**CloudNative-PG Cluster:**
- Purpose: Declarative PostgreSQL clusters managed by the CNPG operator
- Examples: `apps/nextcloud/templates/postgres-cluster.yaml`, `apps/paperless/templates/`
- Pattern: `apiVersion: postgresql.cnpg.io/v1`, `kind: Cluster`, single instance, monitoring enabled

**ArgoCD ApplicationSet:**
- Purpose: Auto-discovers all services in `system/`, `platform/`, `apps/` and creates ArgoCD Applications
- Location: `system/argocd/values.yaml` (under `argocd-apps.applicationsets.root`)
- Pattern: Git directory generator; namespace = directory basename; automated sync with pruning and self-heal

## Entry Points

**Cluster Bootstrap:**
- Location: `system/bootstrap.yml` (Ansible playbook)
- Triggers: Run manually via `make system` from the `system/` directory
- Responsibilities: Creates `argocd` namespace, renders ArgoCD Helm chart (with seed values on first install), applies manifests; ArgoCD then takes over all further reconciliation

**Bare Metal Boot:**
- Location: `metal/boot.yml`
- Triggers: `make boot` from `metal/`
- Responsibilities: Starts PXE server (Docker Compose), wakes metal0 via WoL

**Cluster Provisioning:**
- Location: `metal/cluster.yml`
- Triggers: `make cluster` from `metal/`
- Responsibilities: Installs prerequisites, K3s, automatic-upgrade controller, then Cilium CNI

**External Infra:**
- Location: `external/main.tf`
- Triggers: `make external` from repo root
- Responsibilities: Applies OpenTofu to create Cloudflare tunnel, DNS records, API tokens, and injects them as Kubernetes secrets

## Error Handling

**Strategy:** Kubernetes-native retry with exponential backoff. No custom error handling code.

**Patterns:**
- ArgoCD retry policy: up to 10 attempts, 1m initial backoff, 2x factor, 16m max — configured in `system/argocd/values.yaml`
- `selfHeal: true` means ArgoCD continuously corrects drift without human intervention
- Pre-commit hooks (`yamllint`, `helmlint`, `tofu-validate`, `shellcheck`) catch errors before commit

## Cross-Cutting Concerns

**Logging:** Loki stack (`system/loki/`) collects logs from all pods; Grafana (`platform/grafana/`) provides the query UI

**Metrics:** kube-prometheus-stack (`system/monitoring-system/`) with Prometheus + Alertmanager + Grafana data source; services expose `ServiceMonitor` resources to opt in

**Authentication:** Kanidm (LDAP/OIDC IdP) → Dex (OIDC broker) → per-app OIDC integration. Each app referencing `dex.meyeringh.org` as its OIDC issuer

**Certificate Management:** cert-manager (`system/cert-manager/`) with `ClusterIssuer/letsencrypt-prod` using DNS-01 via Cloudflare API token; all Ingress resources annotate `cert-manager.io/cluster-issuer: letsencrypt-prod`

**Backup:** VolSync (`system/volsync-system/`) provides PVC replication; configured per-namespace via `scripts/backup`

**DNS:** External-DNS (`system/external-dns/`) auto-syncs Ingress hostnames to Cloudflare; apps annotate Ingress with `external-dns.alpha.kubernetes.io/target`

---

*Architecture analysis: 2026-03-15*
