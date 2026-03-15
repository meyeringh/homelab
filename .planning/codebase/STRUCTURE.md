# Codebase Structure

**Analysis Date:** 2026-03-15

## Directory Layout

```
homelab/
├── metal/                      # Ansible: bare metal provisioning
│   ├── boot.yml                # Playbook: PXE boot + WoL
│   ├── cluster.yml             # Playbook: K3s + Cilium install
│   ├── inventories/            # Ansible inventories (prod.yml, stag.yml)
│   ├── group_vars/             # Ansible group variables
│   ├── roles/
│   │   ├── prerequisites/      # OS prep
│   │   ├── k3s/                # K3s install
│   │   ├── cilium/             # Cilium CNI install
│   │   ├── automatic_upgrade/  # K3s auto-upgrade controller
│   │   ├── pxe_server/         # Docker Compose PXE server
│   │   └── wake/               # Wake-on-LAN
│   ├── kubeconfig.yaml         # Kubeconfig after provisioning (gitignored? check)
│   └── Makefile                # make boot / make cluster
│
├── system/                     # Core infra Helm charts
│   ├── argocd/                 # ArgoCD + ApplicationSet (the GitOps engine)
│   ├── bootstrap.yml           # Ansible: initial ArgoCD bootstrap
│   ├── cert-manager/           # Let's Encrypt certificates
│   ├── cf-switch/              # Cloudflare WAF toggle utility
│   ├── cloudflared/            # Cloudflare Tunnel daemon
│   ├── cloudflare-ddns/        # Dynamic DNS updater for home IP
│   ├── cloudnative-pg/         # CloudNative-PG operator for PostgreSQL
│   ├── external-dns/           # Syncs Ingress hostnames to Cloudflare DNS
│   ├── ingress-nginx/          # NGINX ingress controller
│   ├── loki/                   # Loki log aggregation (loki-stack)
│   ├── monitoring-system/      # kube-prometheus-stack (Prometheus + Alertmanager)
│   ├── rook-ceph/              # Rook-Ceph block storage operator + cluster
│   ├── volsync-system/         # VolSync PVC backup/replication
│   └── Makefile                # make (runs system/bootstrap.yml)
│
├── platform/                   # Platform services Helm charts
│   ├── dex/                    # Dex OIDC broker (bridges apps to Kanidm)
│   ├── external-secrets/       # External Secrets Operator
│   ├── global-secrets/         # ClusterSecretStore + secret-generator job
│   ├── grafana/                # Grafana dashboards and alerting
│   ├── kanidm/                 # Kanidm identity provider (OIDC/LDAP)
│   ├── proton/                 # Proton Bridge (email)
│   └── renovate/               # Renovate bot for dependency updates
│
├── apps/                       # User application Helm charts
│   ├── actualbudget/           # Actual Budget personal finance
│   ├── home/                   # Home Assistant
│   ├── jellyfin/               # Jellyfin media server
│   ├── meyeringh-org/          # Personal website
│   ├── minecraft/              # Minecraft server
│   ├── nextcloud/              # Nextcloud (file sync + office)
│   ├── paperless/              # Paperless-ngx document management
│   ├── rustdesk/               # RustDesk remote desktop relay
│   ├── tailscale/              # Tailscale subnet router
│   ├── vaultwarden/            # Vaultwarden (Bitwarden-compatible)
│   └── webtrees/               # Webtrees genealogy app
│
├── external/                   # OpenTofu (Terraform) for external infra
│   ├── main.tf                 # Root module: cloudflare, ntfy, extra-secrets
│   ├── variables.tf            # Input variables
│   ├── terraform.tfvars        # Variable values (not committed)
│   ├── terraform.tfvars.example # Template for tfvars
│   ├── namespaces.yml          # Helper for namespace pre-creation
│   ├── modules/
│   │   ├── cloudflare/         # Tunnel, DNS records, API tokens → k8s secrets
│   │   ├── extra-secrets/      # Generic Kubernetes secret from tfvars data
│   │   └── ntfy/               # ntfy push notification credentials
│   └── Makefile                # make (runs tofu apply)
│
├── test/                       # Go integration + smoke tests
│   ├── smoke_test.go           # Verifies ArgoCD, Grafana, Kanidm over HTTPS
│   ├── integration_test.go     # Verifies ArgoCD ingress availability
│   ├── external_test.go        # External connectivity tests
│   ├── tools_test.go           # Test helper/tooling
│   ├── benchmark/              # Benchmark tests (security, storage)
│   ├── go.mod                  # Go module definition
│   └── Makefile                # make / make filter=Smoke
│
├── scripts/                    # Utility shell scripts
│   ├── new-service             # Scaffold a new app in apps/
│   ├── backup                  # Configure/trigger VolSync backups
│   ├── configure               # Initial repo configuration
│   ├── hacks                   # Post-install fixups
│   ├── onboard-user            # Kanidm user onboarding
│   ├── argocd-admin-password   # Retrieve ArgoCD admin password
│   └── [other helpers]
│
├── flake.nix                   # Nix dev environment (all tooling)
├── .envrc                      # direnv: loads Nix flake
├── .pre-commit-config.yaml     # Pre-commit hooks
├── .yamllint.yaml              # YAML lint rules
├── Makefile                    # Top-level: metal → system → external → smoke-test
└── CLAUDE.md                   # Repo conventions for Claude
```

## Directory Purposes

**`system/`:**
- Purpose: Core Kubernetes infrastructure that all workloads depend on. Deployed first during bootstrap.
- Contains: One subdirectory per service, each a self-contained Helm chart
- Key files: `system/argocd/values.yaml` (ApplicationSet definition), `system/bootstrap.yml` (initial install)

**`platform/`:**
- Purpose: Shared platform capabilities consumed by apps: identity (Kanidm, Dex), secrets (external-secrets, global-secrets), observability (Grafana), dependency updates (Renovate)
- Contains: One subdirectory per service, each a Helm chart

**`apps/`:**
- Purpose: End-user applications. Each is a standalone service in its own namespace.
- Contains: One subdirectory per app

**`metal/`:**
- Purpose: Idempotent bare metal provisioning. Run once to bring up the physical host, then rarely touched.
- Contains: Ansible playbooks and roles

**`external/`:**
- Purpose: Infrastructure that must exist before or alongside the Kubernetes cluster — Cloudflare configuration and bootstrapped secrets.
- Contains: OpenTofu modules; outputs are Kubernetes Secrets written directly to the cluster

**`test/`:**
- Purpose: Post-deployment verification using terratest + Go
- Contains: Go test files; smoke tests check that key services return HTTP 200

**`scripts/`:**
- Purpose: Operational utilities. Not part of the GitOps reconciliation loop.
- Contains: Shell scripts for one-off tasks (onboarding, backup setup, scaffolding)

## Key File Locations

**Entry Points:**
- `Makefile`: Top-level orchestration (`make` runs full deploy)
- `metal/Makefile`: Bare metal provisioning (`make boot`, `make cluster`)
- `system/bootstrap.yml`: First-time ArgoCD installation
- `external/main.tf`: External infrastructure root

**GitOps Control Plane:**
- `system/argocd/values.yaml`: ApplicationSet definition — controls what ArgoCD watches and how it deploys
- `system/argocd/values-seed.yaml`: Stripped-down values for first install (no metrics, minimal config)
- `system/argocd/Chart.yaml`: ArgoCD + argocd-apps chart dependencies

**Secret Management:**
- `platform/global-secrets/templates/clustersecretstore/clustersecretstore.yaml`: The `ClusterSecretStore/global-secrets` resource that all apps reference
- `external/modules/cloudflare/main.tf`: Where Cloudflare API tokens are created and injected as secrets
- `external/modules/extra-secrets/main.tf`: Generic secret injection from tfvars

**Service Template Pattern:**
- `apps/nextcloud/templates/secret.yaml`: Canonical example of ExternalSecret + postgres-cluster pattern
- `apps/nextcloud/templates/postgres-cluster.yaml`: Canonical CloudNative-PG Cluster definition

**Configuration:**
- `metal/inventories/prod.yml`: Physical host inventory (IP, MAC, disk, NIC)
- `metal/group_vars/`: Ansible group variables for cluster config
- `external/terraform.tfvars.example`: Required variables for OpenTofu
- `flake.nix`: Dev toolchain definition
- `.pre-commit-config.yaml`: Code quality gates

**Testing:**
- `test/smoke_test.go`: Smoke test entry point
- `test/integration_test.go`: Integration test entry point
- `test/Makefile`: `make filter=Smoke` runs smoke tests only

## Naming Conventions

**Directories:**
- Service directories match the Kubernetes namespace name (`apps/nextcloud` → namespace `nextcloud`)
- System services use descriptive lowercase names matching upstream chart names where possible
- Multi-word names use hyphens: `cloudflare-ddns`, `monitoring-system`, `volsync-system`

**Helm Charts:**
- `Chart.yaml`: Always `apiVersion: v2`, `version: 0.0.0` (version is managed upstream)
- `values.yaml`: Top-level key matches the upstream chart name (e.g., `argo-cd:`, `nextcloud:`, `app-template:`)

**Kubernetes Resources in Templates:**
- All metadata uses `{{ .Release.Name }}` and `{{ .Release.Namespace }}` for name/namespace
- Secret names follow pattern `<app>.<component>` for external-secrets keys (e.g., `nextcloud.redis`, `vaultwarden.vaultwarden`)

**Files:**
- Templates: lowercase with hyphens, descriptive names (`postgres-cluster.yaml`, `secret.yaml`, `clusterissuer.yaml`)
- Ansible playbooks: lowercase with hyphens (`boot.yml`, `cluster.yml`)
- OpenTofu: standard Terraform naming (`main.tf`, `variables.tf`, `versions.tf`)

## Where to Add New Code

**New User Application:**
1. Run `scripts/new-service <name>` to scaffold `apps/<name>/Chart.yaml` and `apps/<name>/values.yaml`
2. Edit `Chart.yaml` to add the upstream Helm chart dependency (or `app-template` if none exists)
3. Configure `values.yaml` with ingress, secrets, and service settings
4. Add `templates/secret.yaml` with `ExternalSecret` resources if the app needs secrets
5. Add `templates/postgres-cluster.yaml` if the app needs PostgreSQL
6. Commit to `master` — ArgoCD auto-discovers and deploys

**New Platform Service:**
- Implementation: `platform/<service-name>/`
- Same structure as apps: `Chart.yaml` + `values.yaml` + optional `templates/`

**New System Service:**
- Implementation: `system/<service-name>/`
- Same structure; must be stable before apps can depend on it

**New External Secret (for existing app):**
- Add `ExternalSecret` in `apps/<name>/templates/secret.yaml`
- Reference `ClusterSecretStore/global-secrets`
- Key format: `<app>.<component>`
- The actual secret value must be placed in the backing Kubernetes Secret by OpenTofu or manually

**New Cloudflare Resource:**
- Add to `external/modules/cloudflare/main.tf`
- Reference the `cloudflare_zones.zone` data source for the zone ID
- If injecting a secret into the cluster, use `kubernetes_secret_v1` resource

**New OpenTofu Module:**
- Create `external/modules/<name>/` with `main.tf`, `variables.tf`, `versions.tf`
- Add `module` block to `external/main.tf`
- Add input variables to `external/variables.tf` and `external/terraform.tfvars`

**New Ansible Role:**
- Create `metal/roles/<role-name>/tasks/` (and `defaults/`, `templates/` as needed)
- Add role to appropriate playbook (`cluster.yml` or `boot.yml`)

## Special Directories

**`metal/kubeconfig.yaml`:**
- Purpose: Kubeconfig written by Ansible after K3s install
- Generated: Yes (by Ansible)
- Referenced by: `KUBECONFIG` env var in root Makefile and `metal/Makefile`

**`external/.terraform/`:**
- Purpose: OpenTofu provider cache and state
- Generated: Yes (by `tofu init`)
- Committed: No (gitignored)

**`.planning/codebase/`:**
- Purpose: Architecture and convention documents for AI-assisted development
- Generated: Yes (by Claude)
- Committed: Yes

**`apps/rustdesk/charts/`:**
- Purpose: Vendored Helm chart (bundled instead of fetched from registry)
- Generated: Yes (by `helm dependency update`)
- Committed: Yes (exception — most services do NOT vendor charts)

**`platform/external-secrets/charts/` and `platform/renovate/charts/`:**
- Purpose: Vendored Helm charts for specific services
- Generated: Yes
- Committed: Yes

---

*Structure analysis: 2026-03-15*
