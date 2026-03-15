# Roadmap: SSH Bastion Host

## Overview

Deliver a persistent SSH dev environment in the homelab K3s cluster, accessible from any device. Phase 1 builds the container image with all tooling in the bastion repo. Phase 2 deploys it to Kubernetes with secrets, storage, networking, and DNS in the homelab repo. Phase 3 adds reliability (backups, graceful shutdown) once the service is functional end-to-end.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Container Image + CI** - Dockerfile with all dev tooling and GitHub Actions pipeline pushing to GHCR (completed 2026-03-15)
- [ ] **Phase 2: Kubernetes Deployment + External Access** - Helm chart, Deployment+PVC, LoadBalancer, DNS exposure
- [ ] **Phase 3: Reliability + Hardening** - VolSync backups, graceful shutdown, health checks

## Phase Details

### Phase 1: Container Image + CI
**Goal**: A tested, GHCR-hosted container image with all SRE tooling ready for SSH access
**Depends on**: Nothing (first phase, work happens in bastion repo)
**Requirements**: IMG-01, IMG-02, IMG-03, IMG-04, TOOL-01, TOOL-02, TOOL-03, TOOL-04, TOOL-05, SSH-03
**Success Criteria** (what must be TRUE):
  1. `docker run` of the image drops into a shell as user meyeringh (UID 1000) with kubectl, helm, git, vim, tmux, gh, op, node, python3, and claude all on PATH
  2. `docker run` starts sshd that accepts key-based connections on port 2222 and rejects password auth
  3. GitHub Actions builds and pushes a tagged image to ghcr.io/meyeringh/bastion on every push to main
  4. Processes spawned in the container are reaped properly (tini as PID 1)
**Plans**: 2 plans

Plans:
- [x] 01-01-PLAN.md — Dockerfile with SRE tooling, sshd config, entrypoint, local build + verification
- [x] 01-02-PLAN.md — GitHub Actions CI pipeline to build and push image to GHCR

### Phase 2: Kubernetes Deployment + External Access
**Goal**: User can SSH into bastion.meyeringh.org:2222 from any device with their SSH key
**Depends on**: Phase 1 (image must exist on GHCR)
**Requirements**: K8S-01, K8S-02, K8S-03, K8S-04, K8S-05, SSH-01, SSH-02, NET-01, NET-02
**Success Criteria** (what must be TRUE):
  1. `ssh meyeringh@bastion.meyeringh.org -p 2222` connects successfully from an external device
  2. Files written to /home/meyeringh survive pod restarts (PVC persistence)
  3. SSH host key fingerprint remains stable across pod restarts (no warnings)
  4. ArgoCD shows bastion app healthy and synced
  5. Authorized SSH keys managed manually on PVC (External-Secrets deferred)
**Plans**: 2 plans

Plans:
- [ ] 02-01-PLAN.md — Bastion Helm chart (Chart.yaml + values.yaml) and cloudflare-ddns DNS config
- [ ] 02-02-PLAN.md — Push to git, verify ArgoCD sync, end-to-end SSH verification

### Phase 3: Reliability + Hardening
**Goal**: The bastion survives failures gracefully and home directory is backed up
**Depends on**: Phase 2 (service must be running end-to-end)
**Requirements**: SSH-04, REL-01, REL-02, REL-03
**Success Criteria** (what must be TRUE):
  1. Pod auto-restarts if sshd process crashes (liveness probe)
  2. VolSync creates scheduled backups of the /home PVC
  3. Active SSH sessions receive a wall notice before pod termination, with enough grace period to save work
**Plans**: TBD

Plans:
- [ ] 03-01: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Container Image + CI | 2/2 | Complete    | 2026-03-15 |
| 2. Kubernetes Deployment + External Access | 0/2 | In Progress | - |
| 3. Reliability + Hardening | 0/? | Not started | - |
