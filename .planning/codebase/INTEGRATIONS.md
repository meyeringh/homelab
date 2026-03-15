# External Integrations

**Analysis Date:** 2026-03-15

## DNS & Networking

**Cloudflare:**
- DNS management for `meyeringh.org` — managed via OpenTofu in `external/modules/cloudflare/`
- Zero Trust Tunnel (`cloudflared`) — all public ingress routed through Cloudflare tunnel; tunnel credentials injected as K8s secret `cloudflared-credentials` in namespace `cloudflared`
- DDNS — `cloudflare-ddns` updates `rustdesk.meyeringh.org` A record for direct-IP services (port `21116/21117`)
- External-DNS — auto-syncs Kubernetes Ingress hosts to Cloudflare DNS; API token in secret `cloudflare-api-token` (namespace `external-dns`)
- cert-manager — DNS01 ACME challenge via Cloudflare for Let's Encrypt certs; API token in secret `cloudflare-api-token` (namespace `cert-manager`)
- WAF control — cf-switch `0.30.0` manages Cloudflare WAF rules; API token in secret `cloudflare-api-token` (namespace `cf-switch`)
- Auth: `cloudflare_email` + `cloudflare_api_key` vars in `external/terraform.tfvars`; per-service scoped API tokens created by OpenTofu

**Tailscale:**
- Runs as subnet router (`TS_ROUTES: 192.168.1.0/24`) exposing LAN to Tailnet
- Auth key via secret `tailscale-auth` (key `TS_AUTHKEY`) — `apps/tailscale/values.yaml`
- Image: `ghcr.io/tailscale/tailscale:v1.94.2`

## Certificate Management

**Let's Encrypt (ACME):**
- ClusterIssuer `letsencrypt-prod` in `system/cert-manager/templates/clusterissuer.yaml`
- DNS01 solver via Cloudflare (no HTTP01 challenges)
- All ingresses annotated with `cert-manager.io/cluster-issuer: letsencrypt-prod`

## Secret Management

**External Secrets Operator:**
- Operator deployed at `platform/external-secrets/` (chart `2.1.0`)
- ClusterSecretStore `global-secrets` defined in `platform/global-secrets/templates/clustersecretstore/clustersecretstore.yaml`
- Provider: in-cluster Kubernetes secrets in namespace `global-secrets`
- Secret generator job (`platform/global-secrets/files/secret-generator/`) bootstraps random secrets (OIDC client secrets, DB passwords, admin tokens) on first install

**Terraform Cloud:**
- Remote backend for `external/` state: `app.terraform.io`, org `meyeringh`, workspace `homelab-external`
- Stores Cloudflare credentials, ntfy auth, and `extra_secrets` map for third-party API tokens

## Authentication & Identity

**Kanidm (Primary IdP):**
- Deployed at `platform/kanidm/` — `docker.io/kanidm/server:1.9.2`
- Exposed at `https://auth.meyeringh.org`
- Provides LDAP (port 636) and OIDC
- OIDC issuer: `https://auth.meyeringh.org/oauth2/openid/dex`
- Used as upstream connector for Dex

**Dex (OIDC Bridge):**
- Deployed at `platform/dex/` (chart `0.24.0`)
- Exposed at `https://dex.meyeringh.org`
- Storage: Kubernetes (in-cluster)
- Upstream connector: Kanidm OIDC — credentials via env vars `KANIDM_CLIENT_ID` / `KANIDM_CLIENT_SECRET` from secret `dex-secrets`
- SSO clients configured in `platform/dex/values.yaml`:
  - `grafana-sso` → `https://grafana.meyeringh.org/login/generic_oauth`
  - `nextcloud` → `https://cloud.meyeringh.org/apps/oidc_login/oidc`
  - `paperless` → `https://paperless.meyeringh.org/accounts/oidc/dex/login/callback/`
  - `argocd` → `https://argocd.meyeringh.org/auth/callback`
  - `webtrees` → `https://family.meyeringh.org/index.php?route=/OAuth2Client`
  - `jellyfin` → `https://media.meyeringh.org/sso/OID/redirect/dex`

## Data Storage

**Databases:**
- CloudNative-PG — PostgreSQL operator at `system/cloudnative-pg/` (chart `0.27.1`)
- Per-app PostgreSQL clusters: nextcloud (`nextcloud-postgres-rw`), vaultwarden (`vaultwarden-postgres-app`), paperless (`paperless-postgres-rw`), webtrees (`webtrees-postgres-rw`)
- DB credentials via External Secrets (e.g. secret `nextcloud-postgres-app`, key `password`)

**Block Storage:**
- Rook-Ceph at `system/rook-ceph/` (chart `v1.19.2`)
- Ceph version `quay.io/ceph/ceph:v19.2.3`
- StorageClass `standard-rwo` (default, block, RWO, replicated x2)
- StorageClass `standard-rwx` (CephFS, RWX — used by `meyeringh-org`)

**In-App Caches / KV:**
- Redis — used by Nextcloud (chart-bundled) and Paperless (app-template sidecar container)

**File Storage / Backup (S3-Compatible):**
- VolSync at `system/volsync-system/` (chart `0.15.0`) — daily Restic backups of PVCs to external S3
- S3 bucket, access key, secret key, and Restic password stored as external secrets (key `external` in ClusterSecretStore)
- Backup schedule: `0 1 * * *`; retention: 2 daily / 2 weekly / 2 monthly
- Configured via `scripts/backup` Python script using VolSync `ReplicationSource` CRD

## Monitoring & Observability

**Metrics:**
- kube-prometheus-stack `82.10.3` at `system/monitoring-system/`
- Prometheus scrapes all services via ServiceMonitor (selector `NilUsesHelmValues: false` = cluster-wide)
- Alertmanager configured with ntfy.sh webhook receiver

**Logs:**
- Loki stack `2.10.3` at `system/loki/` — Loki + Promtail
- Promtail DaemonSet ships container logs to Loki
- Grafana datasource: `http://loki.loki:3100`

**Dashboards:**
- Grafana `10.5.15` at `platform/grafana/`
- Exposed at `https://grafana.meyeringh.org`
- Datasources: Prometheus (`http://monitoring-system-kube-pro-prometheus.monitoring-system:9090`) and Loki
- SSO via Dex OIDC (`auth.generic_oauth`)

## Alerting

**ntfy.sh:**
- Push notifications via `https://ntfy.sh`
- Alertmanager sidecar `webhook-transformer` translates Alertmanager payloads to ntfy format
- Auth: secret `webhook-transformer` in `monitoring-system` (keys `NTFY_URL`, `NTFY_TOPIC`), provisioned by OpenTofu module `external/modules/ntfy/`

## Email (SMTP)

**Proton Mail Bridge:**
- Deployed at `platform/proton/` — `shenxn/protonmail-bridge:3.19.0-1`
- Exposed cluster-internally at `mail.meyeringh.org:587` (STARTTLS)
- Used by Nextcloud and Vaultwarden for outbound email
- SMTP credentials stored in per-app secrets (`nextcloud-mail-secret`, `vaultwarden-mail-secret`)

## CI / Automation

**ArgoCD:**
- Deployed at `system/argocd/` (chart `9.4.10`)
- Exposed at `https://argocd.meyeringh.org`
- ApplicationSet watches `github.com/meyeringh/homelab` branch `master`
- Syncs `system/*`, `platform/*`, `apps/*` automatically (prune + self-heal)
- OIDC auth via Dex; RBAC groups: `editor` (full access), default `role:readonly`

**Renovate:**
- Deployed at `platform/renovate/` (chart `46.68.1`) as a CronJob running daily at 09:00
- Manages repos: `meyeringh/homelab`, `meyeringh/cf-switch`
- Auto-merges minor/patch; major updates require manual review
- GitHub token via secret `renovate-secret`

**GitHub:**
- Source repository: `https://github.com/meyeringh/homelab`
- ArgoCD pulls manifests directly from this repo
- meyeringh.org website content synced from `https://github.com/meyeringh/meyeringh-org` via git-sync sidecar

## Webhooks & Callbacks

**Incoming:**
- None (no external webhook receivers defined)

**Outgoing:**
- Alertmanager → `http://localhost:8081` (ntfy webhook-transformer sidecar) → `https://ntfy.sh`

## Application-Specific External Integrations

**Paperless → Nextcloud:**
- rclone sidecar in `apps/paperless/` polls Nextcloud WebDAV (`nextcloud:`) every 30s
- Moves PDF/image files into Paperless consume directory
- rclone config stored in secret `nextcloud-webdav-secret`

**meyeringh.org → GitHub:**
- git-sync container (`registry.k8s.io/git-sync/git-sync:v4.6.0`) polls `github.com/meyeringh/meyeringh-org` every 1m
- Serves static site via nginx unprivileged

**Minecraft → SpigotMC / GitHub:**
- init container downloads plugins at pod start:
  - `ProtocolLib.jar` from `github.com/dmulloy2/ProtocolLib`
  - `LoginSecurity.jar` from `spigotmc.org`

## Environment Configuration

**Required external vars (in `external/terraform.tfvars`):**
- `cloudflare_email` — Cloudflare account email
- `cloudflare_api_key` — Cloudflare Global API Key
- `cloudflare_account_id` — Cloudflare account ID
- `ntfy.url` + `ntfy.topic` — ntfy.sh push notification endpoint
- `extra_secrets` — map of additional third-party secrets (e.g. Renovate GitHub token, Tailscale auth key)

**Secrets location:**
- External secrets bootstrapped by `global-secrets` secret generator job
- Infrastructure secrets (Cloudflare tokens, tunnel credentials) injected directly into K8s by OpenTofu
- Dev reference: `external/terraform.tfvars.example`

---

*Integration audit: 2026-03-15*
