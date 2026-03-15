# Requirements: SSH Bastion Host

**Defined:** 2026-03-15
**Core Value:** A consistent, always-available dev environment reachable from anywhere via SSH

## v1 Requirements

### SSH & Authentication

- [ ] **SSH-01**: User can SSH into bastion on port 2222 with key-based authentication only
- [ ] **SSH-02**: SSH host keys persist across pod restarts (no fingerprint warnings)
- [x] **SSH-03**: Password authentication and root login are disabled
- [ ] **SSH-04**: sshd health check auto-restarts pod if SSH server crashes

### Dev Tooling

- [x] **TOOL-01**: User has kubectl, helm, git, vim, tmux, and gh CLI available immediately after login
- [x] **TOOL-02**: User can run Claude Code via session authentication (no API key required)
- [x] **TOOL-03**: User can create and use Python 3 virtual environments for project work
- [x] **TOOL-04**: User has Node.js runtime available for JS/TS development
- [x] **TOOL-05**: User has 1Password CLI (op) available for secret retrieval

### Container Image

- [x] **IMG-01**: Custom Docker image based on Ubuntu 24.04 with all tools pre-installed
- [x] **IMG-02**: Image uses tini as PID 1 for proper signal handling and zombie reaping
- [x] **IMG-03**: Image runs as non-root user meyeringh (UID 1000)
- [ ] **IMG-04**: GitHub Actions CI pipeline builds and pushes image to ghcr.io/meyeringh/bastion

### Kubernetes Deployment

- [ ] **K8S-01**: StatefulSet with PVC for /home/meyeringh persistence (Rook-Ceph)
- [ ] **K8S-02**: LoadBalancer service exposing port 2222 via Cilium L2 (no Cloudflare tunnel)
- [ ] **K8S-03**: Authorized SSH keys injected via External-Secrets operator
- [ ] **K8S-04**: Helm wrapper chart in apps/bastion/ using bjw-s app-template
- [ ] **K8S-05**: initContainer sets correct PVC ownership (UID 1000)

### Networking & DNS

- [ ] **NET-01**: bastion.meyeringh.org resolves to home IP via cloudflare-ddns (proxied: false)
- [ ] **NET-02**: User can SSH from any device (phone, laptop) to bastion.meyeringh.org:2222

### Reliability

- [ ] **REL-01**: VolSync backs up /home PVC on a schedule
- [ ] **REL-02**: Pod shutdown sends wall notice to active sessions before terminating
- [ ] **REL-03**: Extended terminationGracePeriodSeconds allows graceful session cleanup

## v2 Requirements

### Polish

- **POL-01**: MOTD displays hostname, tool versions, cluster status on login
- **POL-02**: Dotfiles auto-clone from git repo on first login
- **POL-03**: tmux auto-attach on SSH login
- **POL-04**: Resource limits tuned based on observed Claude Code usage

### Security

- **SEC-01**: Cilium NetworkPolicy for SSH connection rate limiting
- **SEC-02**: Monitoring/alerting for failed SSH login attempts

## Out of Scope

| Feature | Reason |
|---------|--------|
| Web terminal (ttyd, code-server) | Pure SSH access is the goal |
| Multi-user support | Single-user homelab |
| Tailscale SSH | Requires Tailscale client, limits device compatibility |
| Docker-in-Docker | Security risk, use kubectl instead |
| fail2ban | Needs NET_ADMIN, can't see real IPs behind Cilium LB |
| Mosh | Requires UDP port range, tmux covers disconnect resilience |
| IDE server (VS Code Server) | Claude Code + vim is the workflow |
| Init scripts for tools | All tools baked into Docker image |
| ANTHROPIC_API_KEY secret | Claude Code uses session auth, not API key |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SSH-01 | Phase 2 | Pending |
| SSH-02 | Phase 2 | Pending |
| SSH-03 | Phase 1 | Complete |
| SSH-04 | Phase 3 | Pending |
| TOOL-01 | Phase 1 | Complete |
| TOOL-02 | Phase 1 | Complete |
| TOOL-03 | Phase 1 | Complete |
| TOOL-04 | Phase 1 | Complete |
| TOOL-05 | Phase 1 | Complete |
| IMG-01 | Phase 1 | Complete |
| IMG-02 | Phase 1 | Complete |
| IMG-03 | Phase 1 | Complete |
| IMG-04 | Phase 1 | Pending |
| K8S-01 | Phase 2 | Pending |
| K8S-02 | Phase 2 | Pending |
| K8S-03 | Phase 2 | Pending |
| K8S-04 | Phase 2 | Pending |
| K8S-05 | Phase 2 | Pending |
| NET-01 | Phase 2 | Pending |
| NET-02 | Phase 2 | Pending |
| REL-01 | Phase 3 | Pending |
| REL-02 | Phase 3 | Pending |
| REL-03 | Phase 3 | Pending |

**Coverage:**
- v1 requirements: 23 total
- Mapped to phases: 23
- Unmapped: 0

---
*Requirements defined: 2026-03-15*
*Last updated: 2026-03-15 after roadmap creation*
