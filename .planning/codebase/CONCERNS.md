# Codebase Concerns

**Analysis Date:** 2026-03-15

## Tech Debt

**`scripts/hacks` — acknowledged throwaway script still in production use:**
- Issue: `scripts/hacks` is explicitly marked "TODO: retire this script" in its own docstring. It is called by `make post-install` as a required step in the full deploy pipeline.
- Files: `scripts/hacks`, `Makefile` (line 27)
- Impact: Post-install automation for Kanidm OAuth2 setup and group creation depends on this script. The underlying blocker (Kanidm Python client library) is tracked upstream.
- Fix approach: Migrate to proper Kanidm API calls once [kanidm/kanidm#2301](https://github.com/kanidm/kanidm/pull/2301) is merged and client library is updated.

**`scripts/configure` — WIP, marked for cleanup:**
- Issue: Script has `# WIP` and `# TODO clean this up` at the top. Domain/timezone configuration relies on sed-style string substitution across many files, which is fragile.
- Files: `scripts/configure`
- Impact: A user forking/adapting this repo who runs `make configure` may encounter incomplete behavior or incorrect replacements.
- Fix approach: Proper templating system or Helm values override.

**All CNPG PostgreSQL clusters run as single instances:**
- Issue: All four CloudNative-PG clusters use `instances: 1` — no standby replica.
- Files: `apps/vaultwarden/templates/postgres-cluster.yaml`, `apps/webtrees/templates/postgres-cluster.yaml`, `apps/nextcloud/templates/postgres-cluster.yaml`, `apps/paperless/templates/postgres-cluster.yaml`
- Impact: Any postgres pod restart or OOM kill causes downtime for that app with no automatic failover.
- Fix approach: Increase to `instances: 2` or `3` for critical apps (vaultwarden, nextcloud). This is a homelab single-node constraint — adding a second instance still shares the same physical node but provides faster recovery via HA.

**Rook-Ceph runs with single monitor and single manager:**
- Issue: `mon.count: 1` and `mgr.count: 1` in `system/rook-ceph/values.yaml`.
- Files: `system/rook-ceph/values.yaml`
- Impact: If the MON or MGR process crashes, Ceph enters a degraded state until it recovers. All PVC operations (mounts, provisioning) will be affected.
- Fix approach: Accepted trade-off for single-node homelab. Document explicitly as a known constraint.

**ArgoCD ApplicationSet uses `project: default` with no further constraint:**
- Issue: `project: default # TODO` in ArgoCD ApplicationSet template. All apps share a single ArgoCD project with no namespace-level isolation or resource allow-listing.
- Files: `system/argocd/values.yaml` (line 123)
- Impact: Any app can sync cluster-scoped resources. No multi-tenancy isolation.
- Fix approach: Create per-layer ArgoCD projects (`system`, `platform`, `apps`) with appropriate resource whitelists.

**VolSync backups deployed imperatively, not declaratively:**
- Issue: `ReplicationSource` resources are created by running `make backup` (which calls `scripts/backup`). They are not tracked in git as Helm templates.
- Files: `scripts/backup`, `Makefile` (lines 30-44)
- Impact: After a full cluster restore, backup schedules must be manually re-applied via `make backup`. State is not self-healing via ArgoCD.
- Fix approach: Move `ReplicationSource` definitions into app-level Helm chart templates so ArgoCD manages them.

**kanidm namespace backup is missing:**
- Issue: The `make backup` target covers nextcloud, vaultwarden, paperless, webtrees, proton, minecraft, home, jellyfin, actualbudget — but NOT kanidm.
- Files: `Makefile`, `platform/kanidm/`
- Impact: Kanidm holds all identity data (users, groups, OAuth2 clients). Loss of Kanidm without backup means manual reconstruction of all users and OAuth2 clients.
- Fix approach: Add `kanidm` PVC to `make backup` target.

**`scripts/take-screenshots` references a hardcoded local Firefox profile path:**
- Issue: `profile_directory="/home/khuedoan/.mozilla/firefox/h05irklw.default-release"` — hardcoded path for a different user, and marked `# TODO do not hard code`.
- Files: `scripts/take-screenshots`
- Impact: Script is non-functional on any machine other than the original author's. Also uses a deprecated Selenium API.
- Fix approach: Use `webdriver.Options()` with headless mode; remove profile directory dependency.

## Known Bugs / Functional Limitations

**Alertmanager webhook transformer only processes the first alert in a group:**
- Symptoms: When multiple alerts fire simultaneously, only `body.alerts[0]` is sent to ntfy. Remaining alerts in the batch are silently dropped.
- Files: `system/monitoring-system/files/webhook-transformer/alertmanager-to-ntfy.jsonnet` (line 36)
- Trigger: Any alert group with more than one firing alert.
- Workaround: `group_by: [namespace]` in alertmanager config limits grouping, reducing but not eliminating the risk.

**Webtrees OAuth2 config requires a manual pod restart on first install:**
- Symptoms: The `postStart` lifecycle hook writes OAuth2 config only if `config.ini.php` already exists. On initial install the file does not exist, so the hook writes a message to `/usr/share/message` and exits. A pod restart is required.
- Files: `apps/webtrees/values.yaml` (lines 30-50)
- Trigger: First deployment of webtrees in a fresh namespace.
- Workaround: Restart the webtrees pod once after initial deployment.

**Dex PKCE is disabled for Kanidm connector:**
- Symptoms: PKCE is not enabled for the Kanidm OIDC connector, tracked as waiting on upstream [dexidp/dex#3188](https://github.com/dexidp/dex/pull/3188). The Kanidm OAuth2 app also has `warning-insecure-client-disable-pkce` explicitly set in `scripts/hacks`.
- Files: `platform/dex/values.yaml` (line 29), `scripts/hacks` (line 77)
- Impact: Auth flow between Kanidm and Dex uses a less secure authorization code flow without PKCE.
- Fix approach: Enable once upstream PR is merged.

**Nextcloud collabora disables namespace mounting:**
- Symptoms: `--o:mount_namespaces=false` is set as a workaround for [CollaboraOnline/online#9534](https://github.com/CollaboraOnline/online/issues/9534).
- Files: `apps/nextcloud/values.yaml` (line 201)
- Impact: Collabora runs without namespace isolation, which is a container security regression.
- Fix approach: Track and remove flag when upstream issue is resolved.

**Minecraft runs with `online-mode=false`:**
- Symptoms: `online-mode=false` in `server.properties` means Mojang account verification is disabled. Any client with any username can connect.
- Files: `apps/minecraft/values.yaml` (line 72)
- Impact: No Mojang auth — relies entirely on the LoginSecurity plugin for player authentication. A compromised or absent plugin would allow unauthorized access.
- Fix approach: Enable `online-mode=true` or ensure LoginSecurity plugin is always current.

## Security Considerations

**Cloudflare provider authenticates with global API key, not scoped API token:**
- Risk: `versions.tf` configures the Cloudflare provider with `email` + `api_key` (global API key). This gives full account access, not least-privilege access.
- Files: `external/versions.tf` (lines 31-34), `external/variables.tf`
- Current mitigation: Key is stored in Terraform Cloud workspace, not in git.
- Recommendations: Migrate to `api_token` with scoped permissions (as already done for `external_dns` and `cert_manager` tokens created within the module).

**`volsync.backube/privileged-movers: "true"` applied globally to all namespaces:**
- Risk: Every ArgoCD-managed namespace gets the annotation that allows VolSync to run privileged mover pods, even namespaces that have no PVC backups.
- Files: `system/argocd/values.yaml` (lines 144-147)
- Current mitigation: Comment notes this "may be refactored in the future for finer granularity".
- Recommendations: Apply annotation only to namespaces that require VolSync backup.

**No pod-level security context enforced on most apps:**
- Risk: Only `system/cloudflare-ddns` sets `readOnlyRootFilesystem: true` and `allowPrivilegeEscalation: false` at container level. Most apps (vaultwarden, paperless, actualbudget, etc.) run with default security context.
- Files: All `apps/*/values.yaml` except `apps/cloudflare-ddns/values.yaml`
- Current mitigation: Cloudflare tunnel reduces direct internet exposure; all ingress via NGINX.
- Recommendations: Add `allowPrivilegeEscalation: false` and `readOnlyRootFilesystem: true` where container supports it.

**Dex `webtrees-test` static client with ngrok redirect URI left in production:**
- Risk: A test client `webtrees-test` with a redirect URI pointing to `unrefunding-holly-wrathfully.ngrok-free.dev` is configured in the production Dex instance.
- Files: `platform/dex/values.yaml` (lines 62-66)
- Impact: If the ngrok subdomain is claimed by a third party, they could complete an authorization code flow as any webtrees user.
- Recommendations: Remove `webtrees-test` static client from production Dex config.

**Kanidm uses a self-signed certificate instead of cert-manager Let's Encrypt:**
- Risk: Kanidm requires a TLS chain it can load internally. A self-signed certificate is issued by cert-manager (tracked as TODO in `platform/kanidm/templates/certificate.yaml` pointing to [kanidm/kanidm#1227](https://github.com/kanidm/kanidm/issues/1227)).
- Files: `platform/kanidm/templates/certificate.yaml`, `platform/kanidm/values.yaml`
- Current mitigation: TLS is still present; the self-signed cert is consumed internally only.
- Recommendations: Migrate to a cert-manager ClusterIssuer-signed cert once upstream supports external TLS chain.

## Performance Bottlenecks

**Nextcloud startup is slow, probes are tuned to compensate:**
- Problem: Liveness probe `initialDelaySeconds: 30` and `failureThreshold: 30` indicate startup takes up to ~7.5 minutes before Kubernetes considers the pod failed. Startup probe is disabled.
- Files: `apps/nextcloud/values.yaml` (lines 164-183)
- Cause: Nextcloud PHP application with large filesystem; `fsGroupChangePolicy: OnRootMismatch` is already set to mitigate chown time.
- Improvement path: Enable startup probe with higher failureThreshold; reduce liveness probe to normal values post-startup.

**Paperless startup probe has 120 failureThreshold (20-minute window):**
- Problem: `failureThreshold: 120` on the startup probe at `periodSeconds: 10` gives a 20-minute startup window before the container is considered failed.
- Files: `apps/paperless/values.yaml` (line 71)
- Cause: Paperless runs Django migrations on startup, which can be slow on initial deploy.
- Improvement path: Run migrations as an init container instead of relying on startup probe timeout.

**Jellyfin media stack packs 6 containers into a single pod:**
- Problem: Jellyfin, sabnzbd, prowlarr, radarr, sonarr, and jellyseerr all run in a single controller (pod). Failure or resource exhaustion of one container restarts the entire pod.
- Files: `apps/jellyfin/values.yaml`
- Cause: Shared volume access pattern for media files; `ReadWriteOnce` PVC cannot be shared across pods.
- Improvement path: Migrate to `ReadWriteMany` (CephFS `standard-rwx` StorageClass) and split into separate deployments, or accept as-is for homelab.

## Fragile Areas

**`scripts/hacks` is a required step in `make default`:**
- Files: `scripts/hacks`, `Makefile` (line 27 `post-install: @./scripts/hacks`)
- Why fragile: The script uses `pexpect` for interactive Kanidm CLI login (no stdin support in standard subprocess), resets account passwords as a side effect of login, and has several `TODO` items. Any Kanidm API change breaks the entire post-install step.
- Safe modification: Test changes against a dev Kanidm instance before merging. Do not change the Kanidm OAuth2 setup without updating this script.

**Dex stores sessions in Kubernetes CRDs (in-cluster storage):**
- Files: `platform/dex/values.yaml` (lines 13-17)
- Why fragile: `storage.type: kubernetes` with `inCluster: true` means all OAuth2 sessions/tokens are stored as Kubernetes objects. Restarting Dex will invalidate all active sessions. CRD storage does not support clustering.
- Safe modification: Any Dex config change causes a pod restart, logging out all users from all SSO-integrated apps simultaneously.

**Proton Mail Bridge state stored in PVC with no backup:**
- Files: `platform/proton/values.yaml`, `Makefile` (proton PVC is in backup list — included)
- Note: Proton is backed up, but the bridge requires interactive setup on first launch (credential re-entry). A restore will require manual re-authentication with ProtonMail.

**`meyeringh-org` uses `ReadWriteMany` (CephFS) with git-sync sidecar:**
- Files: `apps/meyeringh-org/values.yaml`
- Why fragile: git-sync pulls from GitHub and writes to the shared volume. If git-sync encounters a corrupt worktree or the repo is force-pushed, nginx will serve stale or broken content until the volume is manually cleaned.
- Safe modification: Validate repo state after any forced push to `meyeringh/meyeringh-org`.

**Webtrees OAuth2 module installed from GitHub at container start via `initContainer`:**
- Files: `apps/webtrees/values.yaml` (lines 7-22)
- Why fragile: Every pod start downloads a specific zip from a GitHub release URL. If the release is deleted or GitHub is unreachable, the container will fail to start.
- Safe modification: Pin module version; consider bundling module into a custom image or using an OCI artifact.

## Scaling Limits

**Single-node K3s cluster (metal0 is both master and worker):**
- Current capacity: 1 node (`metal0: 192.168.1.2` appears in both `masters` and `workers` groups in inventory).
- Limit: Any node-level maintenance, kernel upgrade, or hardware failure takes down all workloads simultaneously. No pod disruption budgets exist.
- Scaling path: Ansible inventory already has a `workers: hosts: {}` group; adding a second node is supported by the existing Ansible roles.

**Rook-Ceph OSD memory hard-limited at 4Gi:**
- Current capacity: `limits.memory: 4Gi` for OSD, with comment "1Gi per 1TB of storage".
- Limit: Implies cluster is sized for ~4TB of Ceph-managed storage. Nextcloud alone has a 7Ti PVC claim.
- Scaling path: Increase OSD memory limit proportionally if storage grows beyond 4TB actively used.

## Dependencies at Risk

**`loki-stack` chart (Grafana Labs):**
- Risk: `loki-stack` chart version `2.10.3` is significantly behind current Loki releases. Loki has undergone breaking schema changes (TSDB, ring store) in recent major versions. Renovation of this chart may require manual data migration.
- Files: `system/loki/Chart.yaml`
- Impact: Upgrade path to Loki 3.x is non-trivial; delayed upgrades accumulate schema drift.
- Migration plan: Migrate to the `grafana/loki` chart (single binary mode) which is the actively maintained successor.

**`home-assistant` chart from third-party repo:**
- Risk: The chart is not the official HA chart. The ingress spec uses `identifier: app` / `port: 8123` which is non-standard (inconsistent with other app-template based services using `service.identifier`).
- Files: `apps/home/Chart.yaml`, `apps/home/values.yaml`
- Impact: Chart updates may require values restructuring with no notice.

**Webtrees image `dtjs48jkt/webtrees` is a community-maintained image:**
- Risk: Not an official Docker Hub publisher. Image maintenance and security patching depends on a single community contributor.
- Files: `apps/webtrees/values.yaml` (line 76)
- Impact: Security fixes may lag official Webtrees releases.

**`cloudflare-ddns` uses `tag: latest`:**
- Risk: `favonia/cloudflare-ddns:latest` is pinned to latest, meaning any new release is pulled on pod restart without testing.
- Files: `system/cloudflare-ddns/values.yaml` (line 8)
- Impact: A breaking change in the DDNS tool could silently break dynamic DNS for `rustdesk.meyeringh.org` on next restart.
- Migration plan: Pin to a specific version tag; Renovate will then manage updates.

## Missing Critical Features

**No disaster recovery runbook:**
- Problem: The restore procedure is `make restore` (runs `scripts/backup --action restore` for each PVC), but there is no documented order-of-operations for full cluster rebuild: bootstrap → secrets → restore PVCs → restart apps.
- Blocks: Safe recovery from full hardware failure without data loss.

**No alerting on backup failure:**
- Problem: VolSync `ReplicationSource` resources report `.status.lastSyncTime` and conditions, but there are no PrometheusRules or alerting rules for backup failures or missed schedules.
- Files: `system/monitoring-system/values.yaml` (no backup alert rules)
- Risk: Backups could silently fail for weeks without notification.
- Priority: High

**No NetworkPolicies anywhere:**
- Problem: No `NetworkPolicy` resources exist in any namespace. Any compromised pod can reach any other service in the cluster, including the Kubernetes API server.
- Files: All `apps/*/`, `platform/*/`, `system/*/`
- Risk: Lateral movement within the cluster if any internet-facing app is compromised.
- Priority: Medium

## Test Coverage Gaps

**Smoke tests only cover 3 services (ArgoCD, Grafana, Kanidm):**
- What's not tested: The remaining 15+ apps (nextcloud, vaultwarden, paperless, jellyfin, etc.) have no automated availability checks.
- Files: `test/smoke_test.go`
- Risk: A broken deployment of core user-facing services goes undetected by CI.
- Priority: Low (homelab context; cost/benefit of full test coverage is low)

**Integration tests only validate Terraform syntax, not actual infra state:**
- What's not tested: `test/external_test.go` runs `terraform validate` only. No checks that DNS records exist, tunnels are healthy, or certificates are valid.
- Files: `test/external_test.go`
- Risk: Cloudflare config drift is not caught by tests.
- Priority: Low

**`ignoreTests: true` in Renovate config — automerge happens without tests passing:**
- What's not tested: Renovate is configured with `"ignoreTests": true`, meaning dependency update PRs are auto-merged to `master` without waiting for CI checks.
- Files: `platform/renovate/values.yaml` (line 11)
- Risk: A breaking chart upgrade can be auto-merged and deployed without any validation.
- Priority: Medium — consider removing `ignoreTests: true` or ensuring the smoke test suite is reliable enough to gate merges.

---

*Concerns audit: 2026-03-15*
