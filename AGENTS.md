# Homelab

Single-node K3s cluster on `meyeringh.org`, managed by ArgoCD from the `master`
branch of this repo.

## Architecture

- **K3s** single node `metal0` (192.168.1.2), control-plane endpoint 192.168.1.3
  via kube-vip. Cilium CNI — no Flannel, no kube-proxy, no Traefik, no servicelb.
- **ArgoCD** auto-discovers every directory in `system/`, `platform/` and `apps/`
  through one ApplicationSet.
- **Helm-only.** Each service is a wrapper chart: `Chart.yaml` (dependency on the
  upstream chart) + `values.yaml` + optional `templates/` for extra objects.
  No Kustomize.
- **Ansible** provisions bare metal (`metal/`).
- **OpenTofu** manages Cloudflare and bootstrap secrets (`external/`).

## Repo structure

```
metal/          Ansible: PXE boot, Fedora kickstart, k3s, kube-vip, Cilium
system/         argocd, blackbox-exporter, cert-manager, cloudflare-ddns,
                cloudnative-pg, external-dns, gateway, loki, monitoring-system,
                rook-ceph, smartctl-exporter, volsync-system
platform/       dex, external-secrets, global-secrets, grafana, kanidm, proton,
                renovate
apps/           actualbudget, home, jellyfin, libretranslate, linkwarden,
                minecraft, nextcloud, paperless, rustdesk, tailscale,
                vaultwarden, webtrees, wireguard
external/       OpenTofu: Cloudflare DNS + API tokens, ntfy, extra secrets
test/           Go smoke tests (terratest)
scripts/        Operational helpers, see below
```

## ArgoCD discovery

`system/argocd/values.yaml` defines a single ApplicationSet named `root` with one
git generator over `system/*`, `platform/*`, `apps/*` (depth 1). Each directory
becomes an Application named `{{path.basename}}`, deployed into a namespace of
the same name.

- `repoURL: https://github.com/meyeringh/homelab`, `revision: master`.
- `prune: true`, `selfHeal: true`, `ServerSideApply=true`, `CreateNamespace=true`.
- **There are no sync waves and no Helm hooks anywhere.** Ordering is pure
  retry-driven convergence (`retry.limit: 10`, 1m→16m backoff). Objects that need
  a CRD from another Application simply fail and retry until it lands.
- A new service needs nothing but a directory with a `Chart.yaml`.

## Ingress is Gateway API, not Ingress

There are zero `kind: Ingress` objects in this repo. Do not add one, and do not
write tooling that reads the Ingress API.

- `system/gateway` owns a single `Gateway/main` in namespace `gateway`,
  `gatewayClassName: cilium`, listeners `http:80` and `https:443` terminating the
  `wildcard-meyeringh-org-tls` cert from cert-manager.
- Every workload attaches with an HTTPRoute:
  `parentRefs: [{name: main, namespace: gateway, sectionName: https}]`, either via
  the chart's own `route:`/`httproute:` values or a hand-written template.
- `system/external-dns` uses `sources: [gateway-httproute]` only. Adding an
  HTTPRoute publishes its hostname to Cloudflare automatically.
- To resolve a service hostname programmatically, list HTTPRoutes in the
  namespace and read `.spec.hostnames[0]`.

## Network exposure

Everything is LAN/VPN-only by IP, and that is deliberate.

- The Gateway takes an address from the Cilium LB-IPAM pools 192.168.1.4/30 and
  192.168.1.8/30, so every `*.meyeringh.org` service record published by
  external-dns resolves to an **RFC1918 address**. Public DNS, private target: not
  reachable from the internet.
- Two names are intentionally public. `system/cloudflare-ddns` overwrites
  `wg.meyeringh.org` and `rustdesk.meyeringh.org` with the WAN address every 5
  minutes; both depend on router port forwards.
- Remote access is WireGuard (`apps/wireguard`, LB IP 192.168.1.6) or Tailscale
  (`apps/tailscale`, subnet router advertising 192.168.1.0/24).
- The apex `meyeringh.org` and `www` point at GitHub Pages
  (`meyeringh/meyeringh-org`); unrelated to the cluster.
- There are no NetworkPolicies and no gateway-level authentication. Each app's own
  login is the only boundary, including on the LAN.

## Secrets

- All in-cluster secrets are `ExternalSecret`s against the ClusterSecretStore
  `global-secrets` (or `proton` for SMTP).
- `platform/global-secrets` runs a Go Job that generates random values listed in
  `files/secret-generator/config.yaml` into the `global-secrets` namespace.
- Externally-sourced values (Cloudflare tokens, ntfy, restic/S3, Tailscale auth
  key, Renovate PAT, WireGuard key) come from OpenTofu in `external/`, driven by
  the untracked `external/terraform.tfvars`.
- The `kanidm.dex` credential is the exception: it is created imperatively by
  `scripts/hacks` (`make post-install`), because it can only be minted after
  Kanidm is running.
- `apps/paperless` still needs one hand-created secret (`nextcloud-webdav-secret`,
  see `apps/paperless/README.md`).

## Auth

Kanidm (`auth.meyeringh.org`, self-hosted IdP, nginx TLS sidecar) → Dex
(`dex.meyeringh.org`, single `kanidm` OIDC connector) → per-app static clients
defined in `platform/dex/values.yaml`. Client secrets are generated by the
secret-generator and injected via `platform/dex/templates/secret.yaml`.

`scripts/onboard-user` creates a person in Kanidm and adds them to the `editor`
group. Note that `editor` currently maps to broad ArgoCD RBAC including
`exec, create`.

## Storage and backups

- Rook-Ceph block storage (`standard-rwo`), `replicated.size: 2`,
  `failureDomain: osd` on the single NVMe.
- PostgreSQL is per-app CloudNative-PG, one `Cluster` per app in that app's
  `templates/postgres-cluster.yaml`, single instance, no barman/WAL archiving.
- Backups are VolSync + restic to an off-cluster S3 (Minio on the LAN), set up
  out of band by `scripts/backup`. The PVC list is hand-maintained in the root
  `Makefile` under `backup:` and `restore:` — **any new persistent volume must be
  added to both lists or it is silently unprotected.**

## Bare metal

`metal/boot.yml` runs the `pxe_server` role on the workstation (docker-compose
dnsmasq DHCP-proxy + nginx serving an extracted Fedora ISO and per-MAC kickstart)
and WOL-wakes the node. `metal/cluster.yml` runs `prerequisites`, `k3s`,
`automatic_upgrade` over SSH, then installs Cilium from localhost.

- k3s `v1.36.0+k3s1`, Cilium `1.19.2`, Gateway API CRDs `v1.2.1`.
- Cilium and the Gateway API CRDs are installed imperatively by Ansible and are
  therefore **outside ArgoCD's drift detection**.
- `metal/inventories/prod.yml` is the only working inventory; `stag.yml` is stale.
- Kubeconfig lands at `metal/kubeconfig.yaml` (gitignored). Use
  `KUBECONFIG=metal/kubeconfig.yaml kubectl ...`.

## Commands

```bash
make                # metal system external smoke-test post-install clean
make configure      # scripts/configure — rewrite domain/IPs for a fork
make metal          # Ansible: PXE boot + cluster (boot re-arms PXE on the LAN)
make system         # Render and apply ArgoCD via system/bootstrap.yml
make external       # OpenTofu apply (needs HCP Terraform login)
make smoke-test     # Go smoke tests against live HTTPRoute hostnames
make post-install   # scripts/hacks — Kanidm groups + the Dex OAuth2 client
make backup         # Create VolSync ReplicationSources
make restore        # Create VolSync ReplicationDestinations
make test           # Full Go test suite
make clean          # Tear down the PXE docker-compose stack
make git-hooks      # pre-commit install
```

`scripts/`: `argocd-admin-password`, `backup`, `configure`, `get-dns-config`,
`get-status`, `hacks`, `helm-diff`, `kanidm-reset-password`, `new-service`,
`onboard-user`, `pxe-logs`, `take-screenshots`, `wireguard-phone-config`.
There is no `scripts/restore` — restore is `scripts/backup --action restore`.

`scripts/new-service` scaffolds into `apps/` only; for `platform/` or `system/`
copy an existing directory.

## Dev environment

Nix flake + direnv. `flake.nix` provides ansible, ansible-lint, docker,
docker-compose, dyff, go, gotestsum, jq, k9s, kanidm, kubectl, kubernetes-helm,
opentofu, pre-commit, qrencode, shellcheck, wireguard-tools, yamllint, and a
Python with jinja2, kubernetes, netaddr, pexpect, rich.

`.pre-commit-config.yaml` runs the standard pre-commit-hooks set
(`check-added-large-files`, `check-executables-have-shebangs`,
`check-merge-conflict`, `check-shebang-scripts-are-executable`,
`detect-private-key`, `end-of-file-fixer`, `mixed-line-ending`,
`trailing-whitespace`) plus yamllint, helmlint, tofu-fmt, tofu-validate, tflint,
shellcheck, gofmt and golint. `.yamllint.yaml` ignores `templates/`.

**There is no CI.** pre-commit is client-side and bypassable, while Renovate
auto-merges to `master` hourly with `ignoreTests: true` and ArgoCD auto-syncs and
prunes from that branch. Treat anything merged to `master` as deployed.

## Conventions

- Chart `name:` should match the directory name (three currently do not:
  `monitoring-system`, `volsync-system`, `proton`).
- Fully qualify image repositories (`docker.io/library/alpine`, not `alpine`) and
  quote image tags — an unquoted `tag: 3.20` is parsed as a float and Renovate has
  mangled it before.
- Set memory `requests == limits`; CPU requests are used only in `apps/jellyfin`.
- Prefer ExternalSecret `target.template` for rendering config files that contain
  credentials (see `apps/wireguard/templates/secret.yaml`).

## Known gaps

Do not mistake these for bugs to fix silently; they are tracked debt.

- `scripts/hacks` self-describes as temporary and resets the Kanidm `admin` and
  `idm_admin` passwords on every `make post-install`.
- Five near-identical CNPG `Cluster` templates hardcode `postgresql:17.5`.
- `test/tools_test.go` version constraints are all below what the flake ships, so
  `make test` fails there; `test/external_test.go` needs HCP Terraform credentials.
- `test/go.mod` still declares the upstream module path and `replace`s terratest
  with a third party's fork.
- Blackbox probes are scraped but no PrometheusRule alerts on them.
