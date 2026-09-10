package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
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

func TestScanK8sMissingKubeconfigReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"scan", "k8s", "--kubeconfig", "/private/tmp/teleskope-missing-kubeconfig-for-test", "--timeout", "1s", "--output", "human"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("Run returned %d, want 1", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty output", stdout.String())
	}
	if !strings.Contains(stderr.String(), "connect Kubernetes cluster") || !strings.Contains(stderr.String(), "kubeconfig") {
		t.Fatalf("stderr = %q, want Kubernetes connection error", stderr.String())
	}
}

func TestScanK8sDefaultMissingKubeconfigDoesNotWriteReportDirectory(t *testing.T) {
	var stdout, stderr bytes.Buffer
	outputDir := t.TempDir()

	code := Run([]string{
		"scan", "k8s",
		"--kubeconfig", "/private/tmp/teleskope-missing-kubeconfig-for-test",
		"--timeout", "1s",
		"--output-dir", outputDir,
	}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("Run returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "collecting Kubernetes API inventory") {
		t.Fatalf("stderr = %q, want progress", stderr.String())
	}
	if !strings.Contains(stderr.String(), "connect Kubernetes cluster") {
		t.Fatalf("stderr = %q, want Kubernetes connection error", stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want no report path", stdout.String())
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("read output dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("output dir contains %d entries, want none", len(entries))
	}
}

func TestProgressRendersNumberedStepsAndDetails(t *testing.T) {
	var stderr bytes.Buffer
	progress := newProgress(&stderr)

	progress.Step("collecting EKS inventory for %s", "demo")
	progress.Detail("managed add-ons=%d", 3)
	progress.Step("writing report artifacts")
	progress.Done("report ready: %s", "demo-20260908-120000")

	got := stderr.String()
	for _, want := range []string{
		"◆ 01 collecting EKS inventory for demo\n",
		"  ├─ managed add-ons=3\n",
		"◆ 02 writing report artifacts\n",
		"✓ report ready: demo-20260908-120000\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress output missing %q:\n%s", want, got)
		}
	}
}

func TestServeValidation(t *testing.T) {
	for _, args := range [][]string{
		{"serve", "eks"},
		{"serve", "k8s", "--interval", "0s"},
		{"serve", "k8s", "--timeout", "-1s"},
		{"serve", "eks", "--cluster", "demo", "--aws-interval", "0s"},
		{"serve", "k8s", "unexpected"},
		{"serve", "k8s", "--listen", "invalid-address"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 1 {
			t.Fatalf("%v: code=%d", args, code)
		}
	}
}

func TestAdvisoryValidation(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"advisory", "--listen", "invalid-address"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("Run returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "listen") {
		t.Fatalf("stderr = %q, want listen error", stderr.String())
	}
}

func TestCompareCommandReadsReportDirectories(t *testing.T) {
	base := t.TempDir()
	sourceDir := writeCLISnapshot(t, base, "source", "v1.31.0")
	targetDir := writeCLISnapshot(t, base, "target", "v1.30.0")
	var stdout, stderr bytes.Buffer

	code := Run([]string{"compare", sourceDir, targetDir, "--output", "markdown"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	for _, want := range []string{"# Teleskope migration comparison", "Kubernetes versions differ"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if !strings.Contains(stderr.String(), "loading source scan result") || !strings.Contains(stderr.String(), "analyzing migration differences") {
		t.Fatalf("stderr missing progress:\n%s", stderr.String())
	}
}

func TestCompareCommandValidatesInput(t *testing.T) {
	for _, args := range [][]string{
		{"compare"},
		{"compare", "--source", "one"},
		{"compare", "--source", "one", "--target", "two", "--output", "yaml"},
		{"compare", "one", "two", "three"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 1 {
			t.Fatalf("%v: code=%d stdout=%s stderr=%s", args, code, stdout.String(), stderr.String())
		}
	}
}

func writeCLISnapshot(t *testing.T, base, name, version string) string {
	t.Helper()
	dir := filepath.Join(base, name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	snapshot := inventory.Snapshot{
		SchemaVersion: "teleskope.io/snapshot/v1alpha1",
		CollectedAt:   time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		Source:        inventory.Source{Tool: "teleskope", Version: "test", Mode: "test"},
		EKS:           inventory.EKSInventory{Cluster: inventory.Cluster{Name: name, Version: strings.TrimPrefix(version, "v")}},
		Kubernetes: inventory.Kubernetes{
			Context: name,
			Version: inventory.KubernetesVersion{GitVersion: version},
		},
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "snapshot.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
