package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RemoteWriter asynchronously publishes latest envelopes to a hub.
type RemoteWriter struct {
	endpoint string
	token    string
	client   *http.Client
	log      func(format string, args ...any)
	retry    time.Duration

	once   sync.Once
	latest chan Envelope
}

// RemoteWriterOptions configures collector-to-hub publication.
type RemoteWriterOptions struct {
	URL    string
	Token  string
	Client *http.Client
	Log    func(format string, args ...any)
	Retry  time.Duration
}

// NewRemoteWriter validates the hub URL and prepares an asynchronous writer.
func NewRemoteWriter(opts RemoteWriterOptions) (*RemoteWriter, error) {
	endpoint, err := hubEndpoint(opts.URL)
	if err != nil {
		return nil, err
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	retry := opts.Retry
	if retry <= 0 {
		retry = 5 * time.Second
	}
	return &RemoteWriter{
		endpoint: endpoint,
		token:    opts.Token,
		client:   client,
		log:      opts.Log,
		retry:    retry,
		latest:   make(chan Envelope, 1),
	}, nil
}

// Start runs the retry loop until ctx is cancelled.
func (w *RemoteWriter) Start(ctx context.Context) {
	w.once.Do(func() {
		go w.run(ctx)
	})
}

// Publish queues an envelope without blocking collection. If the queue is full, it replaces the older pending envelope.
func (w *RemoteWriter) Publish(env Envelope) {
	select {
	case w.latest <- env:
	default:
		select {
		case <-w.latest:
		default:
		}
		select {
		case w.latest <- env:
		default:
		}
	}
}

func (w *RemoteWriter) run(ctx context.Context) {
	var pending *Envelope
	for ctx.Err() == nil {
		if pending == nil {
			select {
			case env := <-w.latest:
				pending = &env
			case <-ctx.Done():
				return
			}
		}
		if err := w.send(ctx, *pending); err != nil {
			w.logf("hub publish failed cluster=%s revision=%d error=%s", pending.Cluster.ID, pending.Revision, err)
			timer := time.NewTimer(w.retry)
			select {
			case env := <-w.latest:
				pending = &env
				timer.Stop()
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			}
			continue
		}
		w.logf("hub publish succeeded cluster=%s revision=%d", pending.Cluster.ID, pending.Revision)
		pending = nil
	}
}

func (w *RemoteWriter) send(ctx context.Context, env Envelope) error {
	env, err := CompleteEnvelope(env)
	if err != nil {
		return err
	}
	body, err := json.Marshal(env)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.token != "" {
		req.Header.Set("Authorization", "Bearer "+w.token)
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (w *RemoteWriter) logf(format string, args ...any) {
	if w.log != nil {
		w.log(format, args...)
	}
}

func hubEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("hub URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse hub URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("hub URL must use http or https")
	}
	if u.Host == "" {
		return "", fmt.Errorf("hub URL requires a host")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/clusters"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}
