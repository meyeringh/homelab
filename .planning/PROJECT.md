# SSH Bastion Host

## What This Is

A persistent, SSH-accessible development environment running as a StatefulSet in the homelab K3s cluster. It provides a familiar shell with all SRE tooling (Claude Code, vim, tmux, kubectl, Python venvs) accessible from any device — phone, laptop, or borrowed computer — via SSH on port 2222.

## Core Value

A consistent, always-available dev environment reachable from anywhere via SSH, with all personal tooling, configs, and secrets ready to go.

## Requirements

### Validated

- ✓ K3s cluster with ArgoCD GitOps — existing
- ✓ External-Secrets operator with ClusterSecretStore — existing
- ✓ Rook-Ceph block storage for PVCs — existing
- ✓ Cilium L2 LoadBalancer for direct port exposure — existing
- ✓ Cloudflare DDNS for dynamic IP — existing
- ✓ VolSync for PVC backups — existing

### Active

- [ ] Custom Docker image with dev tooling (Claude Code, vim, tmux, kubectl, helm, Python 3 + venv, Node.js, git)
- [ ] GitHub Actions CI pipeline to build and push image to GHCR
- [ ] StatefulSet deployment with PVC for /home persistence
- [ ] SSH server on port 2222 with key-based authentication
- [ ] LoadBalancer service exposing port 2222 directly (no Cloudflare tunnel)
- [ ] DDNS subdomain (bastion.meyeringh.org) pointing to home IP
- [ ] Secrets injected via External-Secrets operator (SSH authorized_keys, API tokens like ANTHROPIC_API_KEY)
- [ ] Helm wrapper chart in apps/bastion/ following existing homelab patterns

### Out of Scope

- Cloudflare tunnel — bypassed intentionally, direct SSH exposure like minecraft
- Web-based terminal (ttyd, code-server) — pure SSH access is the goal
- Multi-user support — single user (meyeringh) only
- Init scripts for tool installation — all tools baked into Docker image
- Tailscale SSH — SSH keys only for now

## Context

- The bastion image Dockerfile + CI lives in a separate repo at `/home/meyeringh/git/bastion` (GitHub: meyeringh/bastion → ghcr.io/meyeringh/bastion)
- The Helm chart deploying it lives here in homelab under `apps/bastion/`
- Minecraft server uses a similar direct-exposure pattern (LoadBalancer, no tunnel, DDNS)
- The user works across many git projects, so Python venvs and per-project tooling are essential
- Existing homelab uses app-template chart for services without dedicated upstream Helm charts

## Constraints

- **Networking**: Must use LoadBalancer service type with Cilium L2, not Cloudflare tunnel — SSH is a raw TCP protocol
- **DNS**: DDNS subdomain under meyeringh.org via existing cloudflare-ddns
- **Image registry**: GHCR (ghcr.io/meyeringh/bastion), built via GitHub Actions in the bastion repo
- **Storage**: Single PVC for /home/meyeringh, Rook-Ceph block storage
- **Auth**: SSH key-based only, no password auth
- **Port**: 2222 externally

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Custom Dockerfile over init scripts | Fast startup, reproducible, version-pinned tools | — Pending |
| StatefulSet + PVC over VM/LXC | Stays in Kubernetes GitOps workflow, uses existing storage | — Pending |
| Port 2222 over 22 | Less scanner noise on non-standard port | — Pending |
| SSH keys only, no Tailscale | Simpler, works from any device without Tailscale client | — Pending |
| Separate bastion repo for image | Decouples image build lifecycle from homelab GitOps | — Pending |
| GHCR over local registry | No need to run/maintain a registry in cluster | — Pending |

---
*Last updated: 2026-03-15 after initialization*
