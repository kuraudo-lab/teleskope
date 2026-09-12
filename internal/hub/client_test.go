package hub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRemoteWriterPostsEnvelopeWithToken(t *testing.T) {
	got := make(chan Envelope, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/clusters" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		var env Envelope
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			t.Fatal(err)
		}
		got <- env
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	writer, err := NewRemoteWriter(RemoteWriterOptions{URL: server.URL, Token: "secret", Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer.Start(ctx)
	writer.Publish(testEnvelope("prod-a", 7))

	select {
	case env := <-got:
		if env.Cluster.ID != "prod-a" || env.Revision != 7 {
			t.Fatalf("env = %+v", env)
		}
	case <-time.After(time.Second):
		t.Fatal("hub did not receive envelope")
	}
}

func TestRemoteWriterRetriesLatestEnvelope(t *testing.T) {
	var calls atomic.Int32
	got := make(chan Envelope, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "try again", http.StatusBadGateway)
			return
		}
		var env Envelope
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			t.Fatal(err)
		}
		got <- env
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	writer, err := NewRemoteWriter(RemoteWriterOptions{URL: server.URL, Client: server.Client(), Retry: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer.Start(ctx)
	writer.Publish(testEnvelope("prod-a", 1))
	writer.Publish(testEnvelope("prod-a", 2))

	select {
	case env := <-got:
		if env.Revision != 2 {
			t.Fatalf("revision = %d, want latest revision", env.Revision)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("hub did not receive retried envelope")
	}
}

func TestHubEndpointValidation(t *testing.T) {
	if _, err := hubEndpoint("ftp://example.com"); err == nil {
		t.Fatal("accepted unsupported scheme")
	}
	got, err := hubEndpoint("https://hub.example/base/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://hub.example/base/api/clusters" {
		t.Fatalf("endpoint = %q", got)
	}
}
