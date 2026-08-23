package test

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/retry"
)

// routeHost returns the first hostname of the first HTTPRoute in the namespace.
// Terratest has no Gateway API helper, so this goes through kubectl.
func routeHost(t *testing.T, options *k8s.KubectlOptions) string {
	return retry.DoWithRetry(t, "get HTTPRoute hostname", 30, 10*time.Second, func() (string, error) {
		host, err := k8s.RunKubectlAndGetOutputE(t, options, "get", "httproutes", "--output", "jsonpath={.items[0].spec.hostnames[0]}")
		if err != nil {
			return "", err
		}

		host = strings.TrimSpace(host)
		if host == "" {
			return "", fmt.Errorf("no HTTPRoute with a hostname in namespace %s", options.Namespace)
		}

		return host, nil
	})
}

func TestSmoke(t *testing.T) {
	t.Parallel()

	var mainApps = []struct {
		name      string
		namespace string
	}{
		{"argocd", "argocd"},
		{"grafana", "grafana"},
		{"kanidm", "kanidm"},
	}

	for _, app := range mainApps {
		app := app // https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(app.name, func(t *testing.T) {
			t.Parallel()

			options := k8s.NewKubectlOptions("", "", app.namespace)

			// Setup a TLS configuration, ignore the certificate because we may not use cert-manager (like the sandbox environment)
			tlsConfig := tls.Config{
				InsecureSkipVerify: os.Getenv("INSECURE_SKIP_VERIFY") != "",
			}

			// Test the endpoint, this will only fail if we timeout waiting for the service to return a 200 response
			http_helper.HttpGetWithRetryWithCustomValidation(
				t,
				fmt.Sprintf("https://%s", routeHost(t, options)),
				&tlsConfig,
				30,
				60*time.Second,
				func(statusCode int, body string) bool {
					return statusCode == 200
				},
			)
		})
	}
}
