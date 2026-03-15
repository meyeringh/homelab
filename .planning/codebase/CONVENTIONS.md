# Coding Conventions

**Analysis Date:** 2026-03-15

## Languages in Use

This repo spans multiple languages, each with distinct conventions:

- **YAML** — Helm values, Kubernetes manifests, Ansible playbooks, OpenTofu configs
- **Go** — `test/` integration tests, `platform/global-secrets/files/secret-generator/main.go`
- **Python** — `scripts/backup`, `scripts/configure`, `scripts/onboard-user`
- **Shell (sh)** — short utility scripts: `scripts/new-service`, `scripts/onboard-user`
- **HCL (OpenTofu)** — `external/` infrastructure

## Naming Patterns

**Helm chart directories:**
- Lowercase, hyphen-separated: `apps/vaultwarden`, `apps/home-assistant`, `system/cert-manager`, `system/cloudnative-pg`
- Directory name equals Kubernetes namespace name (e.g., `apps/vaultwarden` → namespace `vaultwarden`)

**Helm release names:**
- Match the directory/namespace name (e.g., release `vaultwarden` in namespace `vaultwarden`)

**Chart.yaml version:**
- Always `version: 0.0.0` — versioning is done by upstream dependency versions only

**Kubernetes resource names in templates:**
- Use `{{ .Release.Name }}` for main resource names
- Use `{{ .Release.Namespace }}` for namespace fields
- Explicit names for supplementary resources: `vaultwarden-admin`, `vaultwarden-postgres-app`

**Secret names:**
- Pattern: `<app>-<purpose>` — e.g., `vaultwarden-admin`, `webtrees-sso-secrets`, `nextcloud-mail-secret`

**ExternalSecret key paths:**
- Application secrets: `<app>.<app>` — e.g., `vaultwarden.vaultwarden`
- SSO secrets: `dex.<app>` — e.g., `dex.webtrees` with field `client_secret`
- External/shared secrets: `external` with specific property names

**OpenTofu resources:**
- Snake_case resource names: `cloudflare_api_token.external_dns`, `kubernetes_secret_v1.cloudflared_credentials`
- Kubernetes secrets get annotation `"app.kubernetes.io/managed-by" = "Terraform"`

**Go:**
- Standard Go conventions — PascalCase for exported types, camelCase for unexported
- Struct types are PascalCase: `RandomSecret`
- Functions are PascalCase when exported: `CheckVersion`

**Python:**
- Snake_case for functions and variables: `apply_custom_resource`, `find_and_replace`
- Type hints used in `scripts/configure`

**Ansible tasks:**
- Sentence-case task names: `"Download k3s binary"`, `"Copy k3s binary to nodes"`, `"Ensure config directories exist"`
- All tasks use FQCN module names: `ansible.builtin.get_url`, `ansible.builtin.template`, `kubernetes.core.k8s`

## Helm values.yaml Structure

All apps follow this structure — upstream chart config is nested under the chart dependency alias:

```yaml
# For app-template charts (bjw-s):
app-template:
  controllers:
    <app-name>:
      containers:
        app:
          image:
            repository: <registry>/<image>
            tag: <version>
          env: {}
          resources:
            requests:
              cpu: 10m
              memory: 100Mi
            limits:
              memory: 100Mi
  service:
    app:
      controller: <app-name>
      ports:
        http:
          port: 80
  ingress:
    main:
      enabled: true
      className: nginx
      annotations:
        cert-manager.io/cluster-issuer: letsencrypt-prod
        external-dns.alpha.kubernetes.io/target: "homelab-tunnel.meyeringh.org"
        external-dns.alpha.kubernetes.io/cloudflare-proxied: "true"
```

**Resource requests are always set** with `cpu: 10m` as the default floor for non-intensive workloads. Memory limits match memory requests.

**Probes:** liveness, readiness, and startup probes are set explicitly with `custom: true` for app-template charts, using `httpGet` against `/` on the service port.

## Ingress Annotations (standard set)

All public ingress resources use this annotation set:

```yaml
annotations:
  cert-manager.io/cluster-issuer: letsencrypt-prod
  external-dns.alpha.kubernetes.io/target: "homelab-tunnel.meyeringh.org"
  external-dns.alpha.kubernetes.io/cloudflare-proxied: "true"
```

TLS always uses a named `secretName` matching `<app>-tls-certificate`, with the host anchored using YAML anchors:

```yaml
hosts:
  - host: &host vault.meyeringh.org
    paths: [...]
tls:
  - hosts:
      - *host
    secretName: vaultwarden-tls-certificate
```

## Secrets Pattern

Secrets are never stored in the repo. The pattern is:

1. `ExternalSecret` resource in `templates/secret.yaml` references `ClusterSecretStore/global-secrets`
2. Secret key path follows `<app>.<app>` convention for app secrets
3. SSO client secrets use `dex.<app>` path with property `client_secret`
4. Env vars reference secrets via `valueFrom.secretKeyRef`

**ExternalSecret with single key:**
```yaml
spec:
  data:
    - secretKey: ADMIN_TOKEN
      remoteRef:
        key: vaultwarden.vaultwarden
        property: ADMIN_TOKEN
```

**ExternalSecret with bulk extract:**
```yaml
spec:
  dataFrom:
    - extract:
        key: webtrees.webtrees
```

## PostgreSQL Pattern

Apps with PostgreSQL use CloudNative-PG. The `postgres-cluster.yaml` template is consistent:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: "{{ .Release.Name }}-postgres"
  namespace: "{{ .Release.Namespace }}"
spec:
  instances: 1
  imageName: ghcr.io/cloudnative-pg/postgresql:17.5
  bootstrap:
    initdb:
      database: <app>
      owner: <app>
  storage:
    size: 2Gi
  monitoring:
    enablePodMonitor: true
```

Database connection via secret `<release>-postgres-app` (created by CloudNative-PG operator) with key `uri`.

## YAML Style

Governed by `.yamllint.yaml`:
- Extends `default` ruleset
- `document-start: disable` (no leading `---` required, but `---` is used to separate multiple documents in one file)
- `line-length: disable` (no line length limit)
- `templates/` directory is excluded from yamllint

## OpenTofu Conventions

- Module-per-provider pattern: `external/modules/cloudflare/`, `external/modules/ntfy/`, `external/modules/extra-secrets/`
- `sensitive = true` on all credential variables in `variables.tf`
- Kubernetes secrets created by Tofu get `"app.kubernetes.io/managed-by" = "Terraform"` annotation
- Version constraints use `~>` pessimistic constraint operator

## Python Script Conventions

- Shebang: `#!/usr/bin/env python` (not `python3`)
- Use `argparse` for CLI arguments with `required=True` on all required args
- Use kubernetes Python client (`from kubernetes import client, config`)
- Error handling via `ApiException` with status code checks
- No inline comments on obvious code; comments only on non-obvious behavior

## Shell Script Conventions

- Shebang: `#!/bin/sh` (POSIX sh, not bash)
- Short scripts only (under ~20 lines) — longer scripts are Python
- Variable assignment from command output: `host="$(kubectl get ...)"`
- Export env vars at top of script

## Comments

- Used sparingly — only for non-obvious decisions
- In Go: short inline comments for error context: `// Secret not found, create a new one`
- In Go test files: explain non-obvious patterns with a URL: `// https://github.com/golang/go/wiki/CommonMistakes`
- TODO comments include a URL to the upstream issue when waiting on external fixes

## Makefile Conventions

- All Makefiles start with `.POSIX:`
- `default` target defined explicitly
- Sub-directory makes use `make -C <dir>` from root Makefile
- KUBECONFIG exported as environment variable at the top of relevant Makefiles

---

*Convention analysis: 2026-03-15*
