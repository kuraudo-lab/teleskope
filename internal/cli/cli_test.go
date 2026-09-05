package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"--version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if got, want := strings.TrimSpace(stdout.String()), "teleskope dev"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunRootHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("stdout did not include help usage: %s", stdout.String())
	}
}

func TestScanEKSRequiresCluster(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"scan", "eks"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("Run returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "--cluster is required") {
		t.Fatalf("stderr = %q, want missing cluster error", stderr.String())
	}
}

func TestScanEKSRejectsUnknownOutputBeforeAWSLoad(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"scan", "eks", "--cluster", "demo", "--output", "yaml"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("Run returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "unsupported output") {
		t.Fatalf("stderr = %q, want unsupported output error", stderr.String())
	}
}

func TestScanK8sRejectsUnknownOutputBeforeKubeLoad(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"scan", "k8s", "--output", "yaml"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("Run returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "unsupported output") {
		t.Fatalf("stderr = %q, want unsupported output error", stderr.String())
	}
}

func TestScanK8sMissingKubeconfigReturnsCoverage(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"scan", "k8s", "--kubeconfig", "/private/tmp/teleskope-missing-kubeconfig-for-test", "--timeout", "1s"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "kubernetes  kubeconfig") {
		t.Fatalf("stdout = %q, want kubeconfig coverage", stdout.String())
	}
	if !strings.Contains(stdout.String(), "unavailable") && !strings.Contains(stdout.String(), "xx") {
		t.Fatalf("stdout = %q, want unavailable coverage marker", stdout.String())
	}
}
