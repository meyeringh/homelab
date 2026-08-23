# Homelab

Single-node K3s cluster for `meyeringh.org`, managed by ArgoCD from the `master`
branch of this repo. Everything the cluster runs is declared here; a push to
`master` is a deploy.

Forked from [khuedoan/homelab](https://github.com/khuedoan/homelab) and since
diverged substantially — Gateway API instead of ingress-nginx, Kanidm/Dex SSO,
CloudNative-PG per app, VolSync/restic backups. Upstream's documentation no
longer describes this setup. Thanks to Khue for the foundation.

## Overview

| Layer      | Tool                    | Where       |
| ---------- | ----------------------- | ----------- |
| Bare metal | Ansible (PXE + Fedora)  | `metal/`    |
| Kubernetes | K3s + Cilium            | `metal/`    |
| GitOps     | ArgoCD ApplicationSet   | `system/argocd` |
| Ingress    | Gateway API (Cilium)    | `system/gateway` |
| Identity   | Kanidm → Dex → app SSO  | `platform/` |
| Storage    | Rook-Ceph, CloudNative-PG | `system/` |
| Backups    | VolSync + restic to S3  | `scripts/backup` |
| External   | OpenTofu (Cloudflare)   | `external/` |

Services live in three directories that ArgoCD scans automatically:
`system/` (cluster infrastructure), `platform/` (identity, secrets, observability,
Renovate) and `apps/` (Nextcloud, Vaultwarden, Jellyfin, Home Assistant,
Paperless, Linkwarden, Webtrees, Actual Budget, LibreTranslate, Minecraft,
RustDesk, Tailscale, WireGuard).

## Network

Services are reachable on the LAN and over VPN only. The Gateway holds an address
from the Cilium load-balancer pool (192.168.1.4/30, 192.168.1.8/30), so the public
`*.meyeringh.org` DNS records published by external-dns point at private
addresses. Remote access is WireGuard or Tailscale.

The two exceptions, both pointed at the WAN address by `cloudflare-ddns`:
`wg.meyeringh.org` (the VPN endpoint itself) and `rustdesk.meyeringh.org`.

## Hardware

Single node, `metal0`:

- `192.168.1.2`, NIC `eno1`, system disk `nvme0n1`
- Control-plane endpoint `192.168.1.3` (kube-vip)
- Load-balancer pools `192.168.1.4/30` and `192.168.1.8/30`

Backups replicate off-cluster to a Minio S3 endpoint on the LAN.

## Usage

```bash
make configure   # rewrite domain and addresses (fork setup)
make             # metal → system → external → smoke-test → post-install
```

Individual stages: `make metal`, `make system`, `make external`,
`make smoke-test`, `make post-install`, `make backup`, `make restore`,
`make test`.

`make metal` re-runs the PXE/DHCP-proxy stage, which serves a kickstart to
anything that PXE-boots on the LAN. Run `make -C metal cluster` instead when the
node is already provisioned.

After provisioning, the kubeconfig is at `metal/kubeconfig.yaml`:

```bash
export KUBECONFIG=$PWD/metal/kubeconfig.yaml
kubectl get applications -n argocd
```

## Development

`flake.nix` plus direnv provides the full toolchain (ansible, helm, kubectl, k9s,
opentofu, go, pre-commit, …). Install the hooks with `make git-hooks`.

Repository conventions and architecture notes for coding agents are in
[`AGENTS.md`](AGENTS.md).

## License

GPLv3 — see [`LICENSE.md`](LICENSE.md).
