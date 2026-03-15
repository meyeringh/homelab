---
phase: 01-container-image-ci
plan: 01
subsystem: infra
tags: [docker, ubuntu, ssh, tini, kubectl, helm, claude-code, bastion]

requires: []
provides:
  - "Bastion Docker image with all SRE tooling (kubectl, helm, gh, op, claude, k9s, etc.)"
  - "Hardened SSH server on port 2222 with key-only auth"
  - "Non-root container running as meyeringh (UID 1000)"
  - "tini as PID 1 for proper signal handling"
affects: [02-container-image-ci, 02-k8s-deployment]

tech-stack:
  added: [tini, claude-code-native-installer, nodesource-lts, 1password-cli]
  patterns: [non-root-container, pvc-backed-host-keys, entrypoint-keygen]

key-files:
  created:
    - /home/meyeringh/git/bastion/Dockerfile
    - /home/meyeringh/git/bastion/entrypoint.sh
    - /home/meyeringh/git/bastion/sshd_config
  modified: []

key-decisions:
  - "Remove default ubuntu user (UID/GID 1000) to reuse IDs for meyeringh"
  - "Claude Code installed via native installer as meyeringh user, PATH set via ENV"
  - "SSH host keys generated at runtime in configurable dir (PVC-ready)"

patterns-established:
  - "Non-root sshd on port 2222 with build-time /run/sshd setup"
  - "Entrypoint generates host keys on first boot, then execs sshd"

requirements-completed: [IMG-01, IMG-02, IMG-03, TOOL-01, TOOL-02, TOOL-03, TOOL-04, TOOL-05, SSH-03]

duration: 4min
completed: 2026-03-15
---

# Phase 1 Plan 1: Bastion Container Image Summary

**Ubuntu 24.04 bastion image with 13 SRE tools, tini init, hardened SSH on port 2222, running as non-root meyeringh user**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-15T17:45:08Z
- **Completed:** 2026-03-15T17:49:23Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Dockerfile with all 13 tools: kubectl, helm, git, vim, tmux, gh, op, node, python3, claude, jq, yq, k9s
- Hardened sshd_config (password auth disabled, root login disabled, port 2222)
- Entrypoint with SSH host key generation (PVC-ready configurable path)
- Image builds and all requirements verified via docker run

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Dockerfile, sshd_config, and entrypoint** - `15edd55` (feat)
2. **Task 2: Build image locally and verify all requirements** - `d6f1f23` (fix)

## Files Created/Modified
- `Dockerfile` - Multi-stage build with Ubuntu 24.04, all tools, non-root user, tini
- `entrypoint.sh` - SSH host key generation and sshd startup
- `sshd_config` - Hardened SSH config on port 2222

## Decisions Made
- Removed default `ubuntu` user (UID/GID 1000) in Ubuntu 24.04 to reuse those IDs for `meyeringh`
- Claude Code installed via native installer (`curl -fsSL https://claude.ai/install.sh | bash`) as meyeringh user
- SSH host key directory configurable via `SSH_HOST_KEY_DIR` env var (defaults to `/etc/ssh/host_keys/`)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed UID/GID 1000 conflict with default ubuntu user**
- **Found during:** Task 2 (Build image locally)
- **Issue:** Ubuntu 24.04 ships with `ubuntu` user at UID 1000 and GID 1000, causing groupadd/useradd to fail
- **Fix:** Added `userdel -r ubuntu` before creating meyeringh user/group
- **Files modified:** Dockerfile
- **Verification:** Image builds successfully, `id` shows uid=1000(meyeringh)
- **Committed in:** d6f1f23 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Essential fix for build to succeed. No scope creep.

## Issues Encountered
None beyond the auto-fixed UID/GID conflict.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Docker image ready for CI pipeline (Plan 02: GitHub Actions + GHCR push)
- All tool binaries verified on PATH
- sshd confirmed listening on port 2222

---
*Phase: 01-container-image-ci*
*Completed: 2026-03-15*
