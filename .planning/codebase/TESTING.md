# Testing Patterns

**Analysis Date:** 2026-03-15

## Test Framework

**Runner:**
- Go testing with `gotestsum` for formatted output
- Config: `test/Makefile` (single target)
- Framework: `github.com/gruntwork-io/terratest v0.46.1` (pinned to a fork: `github.com/khuedoan/terratest`)
- Go version: 1.21 (module `git.khuedoan.com/ops/homelab`)

**Assertion Library:**
- `github.com/stretchr/testify v1.8.1` (indirect, via terratest)
- Terratest helper assertions via `t.FailNow()` and `shell.RunCommand`

**Run Commands:**
```bash
# From /test directory:
make                          # Run all tests (30min timeout)
make filter=Smoke             # Run only smoke tests
make filter=TestArgoCDCheck   # Run specific test

# From repo root:
make smoke-test               # Runs filter=Smoke
make test                     # Runs all tests
```

## Test File Organization

**Location:** All test files co-located in `test/` (flat, not co-located with source)

**Naming:**
- `smoke_test.go` — availability checks for core services
- `integration_test.go` — integration-level checks (ArgoCD, etc.)
- `external_test.go` — validates OpenTofu/Terraform configuration
- `tools_test.go` — dev toolchain version constraints

**Package:** All tests in `package test`

## Test Structure

**All tests are parallel at both suite and subtest level:**

```go
func TestSmoke(t *testing.T) {
    t.Parallel()

    var mainApps = []struct {
        name      string
        namespace string
    }{
        {"argocd-server", "argocd"},
        {"grafana", "grafana"},
        {"kanidm", "kanidm"},
    }

    for _, app := range mainApps {
        app := app // capture loop var for goroutine safety
        t.Run(app.name, func(t *testing.T) {
            t.Parallel()
            // test body
        })
    }
}
```

**Note:** Loop variable capture pattern `app := app` is used explicitly with a comment linking to the Go wiki. This is required in Go < 1.22.

## Kubernetes Ingress Tests

Pattern for testing a service is reachable via its ingress:

```go
options := k8s.NewKubectlOptions("", "", app.namespace)  // kubeconfig from $KUBECONFIG env var

// Wait for ingress to be available (retries, timeout)
k8s.WaitUntilIngressAvailable(t, options, app.name, 30, 60*time.Second)

// Fetch ingress object to get the hostname
ingress := k8s.GetIngress(t, options, app.name)

// TLS config — conditionally skip verify based on env var
tlsConfig := tls.Config{
    InsecureSkipVerify: os.Getenv("INSECURE_SKIP_VERIFY") != "",
}

// HTTP GET with retry until 200
http_helper.HttpGetWithRetryWithCustomValidation(
    t,
    fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host),
    &tlsConfig,
    30,
    60*time.Second,
    func(statusCode int, body string) bool {
        return statusCode == 200
    },
)
```

**Timeout parameters:** `(retries int, sleepBetweenRetries time.Duration)`
- Smoke tests: 30 retries × 60s = 30 minutes max wait per service
- Integration tests: 10–30 retries × 1–30s

## Terraform/OpenTofu Validation Tests

Pattern in `test/external_test.go`:

```go
func TestTerraformExternal(t *testing.T) {
    t.Parallel()

    // Copy to temp dir to allow parallel runs against same module
    exampleFolder := test_structure.CopyTerraformFolderToTemp(t, "../external", ".")

    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: exampleFolder,
    })

    terraform.Init(t, terraformOptions)
    terraform.Validate(t, terraformOptions)  // Only validates, does NOT apply
}
```

This test only validates HCL syntax and provider schema — it does not apply or plan against live infrastructure.

## Tool Version Tests

Pattern in `test/tools_test.go`:

```go
var tools = []struct {
    binaryPath        string
    versionArg        string
    versionConstraint string
}{
    {"helm", "version", ">= 3.9.4, < 4.0.0"},
    {"tofu", "--version", ">= 1.7.0, < 1.9.0"},
}

for _, tool := range tools {
    tool := tool
    t.Run(tool.binaryPath, func(t *testing.T) {
        t.Parallel()
        params := version_checker.CheckVersionParams{
            BinaryPath:        tool.binaryPath,
            VersionConstraint: tool.versionConstraint,
            VersionArg:        tool.versionArg,
            WorkingDir:        ".",
        }
        version_checker.CheckVersion(t, params)
    })
}
```

## Nix Shell Test

```go
func TestToolsNixShell(t *testing.T) {
    t.Parallel()

    projectRoot, err := filepath.Abs("../")
    if err != nil {
        t.FailNow()
    }

    command := shell.Command{
        Command:    "nix",
        Args:       []string{"develop", "--experimental-features", "nix-command flakes", "--command", "true"},
        WorkingDir: projectRoot,
    }

    shell.RunCommand(t, command)
}
```

This verifies the Nix dev environment builds successfully.

## Mocking

**No mocking framework is used.** Tests are integration/smoke tests that run against a live or staging Kubernetes cluster. There are no unit tests with mocked dependencies.

**What is tested:**
- Live Kubernetes ingresses are reachable and return HTTP 200
- OpenTofu configuration is valid (no apply)
- Dev tool binaries exist and meet version constraints
- Nix flake evaluates successfully

**What is NOT tested:**
- Individual Helm chart rendering
- Kubernetes resource configuration correctness
- Application-level functionality beyond HTTP 200
- Ansible playbook idempotency

## Test Configuration

**Kubeconfig:** Tests use `$KUBECONFIG` environment variable (set to `metal/kubeconfig.yaml` by root Makefile via `.EXPORT_ALL_VARIABLES`).

**TLS:** `INSECURE_SKIP_VERIFY` environment variable controls TLS verification — empty = verify, non-empty = skip (for staging/sandbox environments).

## Test Types

**Smoke Tests (`TestSmoke`):**
- Scope: Core platform services — ArgoCD, Grafana, Kanidm
- Checks: Ingress available + HTTP 200 response
- Timeout: 30 min total (30 retries × 60s)
- Triggered by: `make smoke-test` in CI/deploy flow

**Integration Tests (`TestArgoCDCheck`):**
- Scope: ArgoCD server specifically
- Checks: Ingress available + HTTP 200
- Shorter timeout: 10 retries × 1s + 30 retries × 30s

**Infrastructure Validation (`TestTerraformExternal`):**
- Scope: `external/` OpenTofu module
- Checks: `tofu init` + `tofu validate` only
- No live cloud API calls

**Toolchain Tests (`TestToolsVersions`, `TestToolsNixShell`):**
- Scope: Dev environment
- Checks: Binary version constraints, Nix shell builds

## Coverage

**Requirements:** None enforced — no coverage targets or reporting configured.

**Coverage command:** Not configured.

## Benchmark Tests

Not Go benchmarks. Manual benchmark YAML manifests exist in `test/benchmark/`:
- `test/benchmark/security/kube-bench.yaml` — CIS Kubernetes benchmark job
- `test/benchmark/storage/dbench-rwo.yaml` — block storage throughput test
- `test/benchmark/storage/dbench-rwx.yaml` — shared storage throughput test

These are ad-hoc Kubernetes jobs, not automated test suite entries.

---

*Testing analysis: 2026-03-15*
