# Technology Stack

**Analysis Date:** 2026-03-15

## Languages

**Primary:**
- YAML - All Kubernetes manifests, Helm values, Ansible playbooks, ArgoCD config
- HCL (Terraform/OpenTofu) - External infrastructure in `external/`
- Go 1.21 - Integration and smoke tests in `test/`

**Secondary:**
- Python 3 - Utility scripts in `scripts/` (backup, onboard-user, etc.)
- Nix - Dev environment definition in `flake.nix`
- Shell/Bash - Short utility scripts in `scripts/`

## Runtime

**Environment:**
- Linux (bare metal node: metal0 @ 192.168.1.2)
- K3s (single-node Kubernetes) — no Flannel, no Traefik, no kube-proxy

**Package Manager:**
- Nix flake (`flake.nix` + `flake.lock`) for dev tooling
- Helm for Kubernetes workloads (all services are Helm charts)
- No npm/pip lockfiles; Python deps come from Nix

## Frameworks

**GitOps / Orchestration:**
- ArgoCD `9.4.10` — auto-discovers apps from `system/*/`, `platform/*/`, `apps/*/` via ApplicationSet git directory generator
- ArgoCD Apps `2.0.4` — ApplicationSet definition (`system/argocd/values.yaml`)

**Kubernetes Distribution:**
- K3s — provisioned via Ansible role `metal/roles/k3s/`
- Cilium — CNI, L2 announcements, LB pool 192.168.1.4/30 (`metal/roles/cilium/`)

**Infrastructure as Code:**
- OpenTofu `~> 1.7` — drop-in Terraform replacement, manages external infra (`external/`)
- Ansible — bare metal provisioning, PXE boot, K3s install (`metal/`)

**Testing:**
- Terratest `v0.46.1` — Go-based Kubernetes integration/smoke tests (`test/`)
- gotestsum — test runner with better output formatting

**Build/Dev:**
- Nix devShell (nixpkgs nixos-25.05) — all tooling pinned via flake
- direnv (`.envrc`) — auto-activates Nix shell
- pre-commit — hooks for lint/format/security checks

## Key Dependencies

**Helm Chart Sources:**
- `bjw-s-labs/helm-charts` app-template `4.6.2` — used by ~12 services as a generic app wrapper
- `argoproj/argo-helm` argo-cd `9.4.10`
- `prometheus-community/helm-charts` kube-prometheus-stack `82.10.3`
- `grafana/helm-charts` loki-stack `2.10.3`, grafana `10.5.15`
- `charts.jetstack.io` cert-manager `v1.20.0`
- `kubernetes-sigs/external-dns` `1.15.0`
- `kubernetes.github.io/ingress-nginx` `4.15.0`
- `charts.rook.io/release` rook-ceph + rook-ceph-cluster `v1.19.2`
- `cloudnative-pg.github.io/charts` cloudnative-pg `0.27.1`
- `backube/helm-charts` volsync `0.15.0`
- `charts.external-secrets.io` external-secrets `2.1.0`
- `charts.dexidp.io` dex `0.24.0`
- `docs.renovatebot.com/helm-charts` renovate `46.68.1`
- `nextcloud.github.io/helm` nextcloud `8.9.1`
- `pajikos.github.io/home-assistant-helm-chart` home-assistant `0.3.47`
- `meyeringh.github.io/cf-switch` cf-switch `0.30.0`

**Critical Go Test Dependencies:**
- `github.com/gruntwork-io/terratest v0.46.1` (via fork `github.com/khuedoan/terratest`)
- `k8s.io/client-go v0.27.2`

**Critical Python Script Dependencies (via Nix):**
- `kubernetes` — used by `scripts/backup` to apply CRDs via K8s API
- `jinja2`, `netaddr`, `pexpect`, `rich` — used by various scripts

## Configuration

**Environment:**
- Kubeconfig at `metal/kubeconfig.yaml` — set via `KUBECONFIG` env var in Makefile
- `KUBE_CONFIG_PATH` also exported for OpenTofu Kubernetes provider
- External secrets vars in `external/terraform.tfvars` (see `external/terraform.tfvars.example`)
- No `.env` files — all secrets via External Secrets Operator pulling from Kubernetes

**Build:**
- `Makefile` at root — orchestrates full deploy: `metal → system → external → smoke-test → post-install`
- `metal/Makefile` — runs Ansible playbooks
- `system/Makefile` — system component deploys
- `external/Makefile` — OpenTofu apply
- `test/Makefile` — runs Go tests

**Linting / Quality:**
- `.pre-commit-config.yaml` — yamllint, helmlint, tofu-fmt, tofu-validate, tflint, shellcheck, gofmt, golint, secret detection
- `.yamllint.yaml` — YAML lint rules

## Platform Requirements

**Development:**
- Nix with flakes enabled (provides all tooling: ansible, helm, kubectl, k9s, tofu, go, etc.)
- direnv for auto-activation
- SSH key at `~/.ssh/id_ed25519`

**Production:**
- Single bare-metal node (metal0)
- K3s installed via Ansible
- ArgoCD manages all workloads post-bootstrap
- Terraform Cloud (`app.terraform.io`, org `meyeringh`) for remote state of `external/`

---

*Stack analysis: 2026-03-15*
