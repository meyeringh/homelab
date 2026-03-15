# Homelab

K3s-based single-node Kubernetes homelab on meyeringh.org, managed via ArgoCD GitOps.

## Architecture

- **K3s** single master/worker (metal0 @ 192.168.1.2), Cilium networking, no Flannel/Traefik/kube-proxy
- **ArgoCD** auto-discovers services from `system/`, `platform/`, `apps/` via ApplicationSet git directory generator
- **Helm-only** deployments — each service has `Chart.yaml` + `values.yaml`, no Kustomize
- **Ansible** for bare metal provisioning (`metal/`)
- **OpenTofu** for external infra (Cloudflare, secrets) in `external/`

## Repo Structure

```
metal/          # Ansible playbooks: PXE boot, K3s install, Cilium
system/         # Core infra: argocd, cert-manager, ingress-nginx, rook-ceph,
                # cloudnative-pg, loki, kube-prometheus-stack, volsync,
                # external-dns, cloudflared, cloudflare-ddns
platform/       # Platform services: dex, kanidm, grafana, external-secrets, renovate
apps/           # User apps: nextcloud, vaultwarden, jellyfin, home-assistant,
                # webtrees, paperless, minecraft, rustdesk, tailscale,
                # actual-budget, meyeringh-org
external/       # OpenTofu: Cloudflare DNS, tunnels, secrets
test/           # Go integration tests (terratest), smoke tests
scripts/        # Utility scripts: backup, restore, new-service, onboard-user
```

## Key Patterns

- **Secrets**: External-Secrets operator pulls from ClusterSecretStore; Terraform manages external secrets
- **Auth**: Kanidm (LDAP/OIDC backend) → Dex (OIDC bridge) → per-app SSO
- **Ingress**: NGINX ingress + cert-manager (Let's Encrypt) + external-dns (Cloudflare)
- **Storage**: Rook-Ceph block storage, CloudNative-PG for PostgreSQL, VolSync for backups
- **DNS**: External-DNS auto-syncs to Cloudflare, DDNS for dynamic IP
- **Networking**: Cilium L2 announcements, LB IP pool 192.168.1.4/30

## Adding a New Service

Run `scripts/new-service` to scaffold. Each service needs:
- `Chart.yaml` with helm dependency
- `values.yaml` with config
- Place in `apps/`, `platform/`, or `system/` — ArgoCD picks it up automatically

## Kubeconfig

After deployment, the kubeconfig is at `metal/kubeconfig.yaml`. Use `KUBECONFIG=metal/kubeconfig.yaml kubectl ...` or set the context accordingly.

## Commands

```bash
make                    # Full deploy: metal → system → external → smoke-test → post-install
make metal              # Ansible provisioning
make system             # System components
make external           # OpenTofu apply
make smoke-test         # Run smoke tests
```

## Dev Environment

Nix flake (`flake.nix`) + direnv provides: ansible, helm, kubectl, k9s, tofu, go, pre-commit.

## Pre-commit Hooks

yamllint, helm lint, tofu validate/fmt, shellcheck, go fmt/lint, git security checks.

## Testing

Go-based tests in `test/` using terratest + gotestsum. Smoke tests verify ArgoCD, Grafana, Kanidm availability over HTTPS.
