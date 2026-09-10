package advisory

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

func TestHandlerServesPageAndComparesUploads(t *testing.T) {
	h := Handler()
	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("page status = %d", page.Code)
	}
	for _, want := range []string{"migration advisory", `name="source"`, `name="target"`, "/api/compare"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("page missing %q", want)
		}
	}

	body, contentType := multipartSnapshots(t, advisorySnapshot("source", "v1.31.0"), advisorySnapshot("target", "v1.30.0"))
	req := httptest.NewRequest(http.MethodPost, "/api/compare", body)
	req.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("compare status = %d body=%s", response.Code, response.Body.String())
	}
	var out comparePayload
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Markdown, "# Teleskope migration comparison") {
		t.Fatalf("markdown missing comparison heading:\n%s", out.Markdown)
	}
	if !strings.Contains(out.Report.Summary, "finding") {
		t.Fatalf("summary = %q", out.Report.Summary)
	}
}

func TestHandlerRejectsInvalidCompareRequests(t *testing.T) {
	h := Handler()
	for _, item := range []struct {
		method string
		path   string
		code   int
	}{
		{http.MethodPost, "/", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/compare", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/compare", http.StatusBadRequest},
		{http.MethodGet, "/missing", http.StatusNotFound},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(item.method, item.path, nil))
		if w.Code != item.code {
			t.Fatalf("%s %s = %d, want %d", item.method, item.path, w.Code, item.code)
		}
	}
}

type comparePayload struct {
	Report struct {
		Summary string `json:"summary"`
	} `json:"report"`
	Markdown string `json:"markdown"`
}

func multipartSnapshots(t *testing.T, source, target inventory.Snapshot) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writeSnapshotPart(t, writer, "source", source)
	writeSnapshotPart(t, writer, "target", target)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, writer.FormDataContentType()
}

func writeSnapshotPart(t *testing.T, writer *multipart.Writer, field string, snapshot inventory.Snapshot) {
	t.Helper()
	part, err := writer.CreateFormFile(field, field+"-snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(part).Encode(snapshot); err != nil {
		t.Fatal(err)
	}
}

func advisorySnapshot(name, version string) inventory.Snapshot {
	return inventory.Snapshot{
		SchemaVersion: "teleskope.io/snapshot/v1alpha1",
		CollectedAt:   time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		Source:        inventory.Source{Tool: "teleskope", Version: "test", Mode: "test"},
		EKS:           inventory.EKSInventory{Cluster: inventory.Cluster{Name: name, Version: strings.TrimPrefix(version, "v")}},
		Kubernetes: inventory.Kubernetes{
			Context: name,
			Version: inventory.KubernetesVersion{GitVersion: version},
		},
	}
}
