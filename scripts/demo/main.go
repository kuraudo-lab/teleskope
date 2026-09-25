// Command demo generates public, credential-free examples using production renderers.
package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/advisor"
	"github.com/kuraudo-lab/teleskope/internal/analysis"
	"github.com/kuraudo-lab/teleskope/internal/hub"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/live"
	"github.com/kuraudo-lab/teleskope/internal/report"
)

//go:embed fixtures/*.json
var fixtures embed.FS

func decode(name string, dst any) error {
	b, err := fixtures.ReadFile("fixtures/" + name + ".json")
	if err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("%s: trailing JSON", name)
	}
	return nil
}

func seededHub() (*hub.Store, error) {
	s := new(hub.Store)
	for _, name := range []string{"source", "target"} {
		var snap inventory.Snapshot
		if err := decode(name, &snap); err != nil {
			return nil, err
		}
		at := snap.CollectedAt
		_, _, err := s.Put(hub.Envelope{Cluster: hub.Cluster{ID: name, Name: name + "-demo", Provider: "eks"}, Revision: 1, CollectedAt: at, Snapshot: &snap, Sources: map[string]live.Status{
			"kubernetes": {State: "ready", Mode: live.SourceModeRecorded, LastSuccess: &at},
			"aws":        {State: "partial", Mode: live.SourceModeRecorded, LastSuccess: &at, Coverage: snap.Coverage[len(snap.Coverage)-1:]},
		}})
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func write(root, name string, data []byte) error {
	if filepath.Ext(name) == ".md" {
		data = append(bytes.TrimRight(data, "\n"), '\n')
	}
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
func writeJSON(root, name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return write(root, name, append(b, '\n'))
}

const shell = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}} · Teleskope</title><style>body{font:16px/1.65 system-ui,sans-serif;max-width:1040px;margin:48px auto;padding:0 24px;color:#173239;background:#f5f8f7}a{color:#00695c}h1{line-height:1.2}section{background:white;border:1px solid #cddbd6;border-radius:16px;padding:24px;margin:24px 0}table{border-collapse:collapse;width:100%}th,td{text-align:left;padding:12px;border-bottom:1px solid #dce5e0}pre{white-space:pre-wrap;overflow-wrap:anywhere}small{color:#48605b}</style><header><small>TELESKOPE / PUBLIC SYNTHETIC DEMO</small><h1>{{.Title}}</h1><p>Fixed fictional data. No AWS, Kubernetes, or AI credentials required. Not a live operational assessment.</p></header>{{.Body}}</html>`

func page(root, name, title string, body template.HTML) error {
	var b bytes.Buffer
	err := template.Must(template.New("page").Parse(shell)).Execute(&b, struct {
		Title string
		Body  template.HTML
	}{title, body})
	if err != nil {
		return err
	}
	return write(root, name, b.Bytes())
}

func generate(root string) error {
	store, err := seededHub()
	if err != nil {
		return err
	}
	for _, name := range []string{"source", "target"} {
		env, _ := store.Get(name)
		html, err := report.HTML(env.Snapshot)
		if err != nil {
			return err
		}
		if err = write(root, name+"/index.html", []byte(html)); err != nil {
			return err
		}
		if err = writeJSON(root, name+"/snapshot.json", env.Snapshot); err != nil {
			return err
		}
		if err = writeJSON(root, name+"/advisor.json", advisor.Analyze(env.Snapshot)); err != nil {
			return err
		}
		if err = write(root, name+"/summary.md", []byte(report.Markdown(env.Snapshot))); err != nil {
			return err
		}
		if err = writeJSON(root, "hub/"+name+"-envelope.json", env); err != nil {
			return err
		}
	}
	fleet := store.Fleet()
	if err = writeJSON(root, "hub/fleet.json", fleet); err != nil {
		return err
	}
	if err = write(root, "hub/summary.md", []byte(regexp.MustCompile(`(?m)^- Generated at:.*$`).ReplaceAllString(hub.Markdown(fleet), "- Demo timestamp: `"+fleet.Clusters[0].CollectedAt.Format(time.RFC3339)+"`"))); err != nil {
		return err
	}
	var b bytes.Buffer
	t := template.Must(template.New("fleet").Parse(`<p><a href="../index.html">All demos</a> · <a href="fleet.json">Fleet JSON</a> · <a href="summary.md">Markdown</a></p><section><h2>Recorded fleet</h2><table><tr><th>Cluster</th><th>Kubernetes</th><th>Nodes</th><th>Workloads</th><th>Collection</th></tr>{{range .Clusters}}<tr><td><a href="../{{.Cluster.ID}}/index.html">{{.Cluster.Name}}</a></td><td>{{.KubernetesVersion}}</td><td>{{.Nodes}}</td><td>{{.Workloads}}</td><td>{{.State}}</td></tr>{{end}}</table><p>Partial collection is intentional: EKS upgrade insights are omitted. Recorded status describes this fixture, not current infrastructure.</p></section><p>For the full Hub UI with search, filters, exports, and cluster drill-down, run <code>go run ./scripts/demo -serve</code>, then open <a href="http://127.0.0.1:8091">localhost:8091</a>.</p>`))
	if err = t.Execute(&b, fleet); err != nil {
		return err
	}
	if err = page(root, "hub/index.html", "Hub fleet sample", template.HTML(b.String())); err != nil {
		return err
	}
	var ai analysis.Result
	if err = decode("analysis", &ai); err != nil {
		return err
	}
	if err = writeJSON(root, "ai/analysis.json", ai); err != nil {
		return err
	}
	md := analysis.Markdown(ai)
	if err = write(root, "ai/analysis.md", []byte(md)); err != nil {
		return err
	}
	b.Reset()
	aiTemplate := template.Must(template.New("ai").Parse(`<p><a href="../index.html">All demos</a> · <a href="analysis.json">Structured JSON</a> · <a href="analysis.md">Markdown</a></p><section><h2>Hand-authored AI output example</h2><p>{{.Summary}}</p><small>Use case: {{.UseCase}} · Provider: {{.Provider}} · Model: {{.Model}}</small></section>{{range .Sections}}<section><h2>{{.Title}}</h2>{{range .Items}}<article><h3>{{.Summary}}</h3><small>{{.Severity}} · {{.Basis}} · confidence: {{.Confidence}}</small><p>{{.Detail}}</p>{{if .Recommendation}}<p><strong>Next step:</strong> {{.Recommendation}}</p>{{end}}<ul>{{range .Resources}}<li>{{.Kind}} / {{.Namespace}} / {{.Name}}</li>{{end}}</ul><details open><summary>Evidence</summary><ul>{{range .Evidence}}<li><code>{{.}}</code></li>{{end}}</ul></details></article>{{end}}</section>{{end}}<section><h2>Limitations</h2><ul>{{range .Limitations}}<li>{{.}}</li>{{end}}</ul></section>`))
	if err = aiTemplate.Execute(&b, ai); err != nil {
		return err
	}
	if err = page(root, "ai/index.html", "AI analysis sample", template.HTML(b.String())); err != nil {
		return err
	}

	return page(root, "index.html", "Explore Teleskope", `<section><h2>Start with a cluster report</h2><p><a href="source/index.html">Open source-demo report →</a> · <a href="target/index.html">Open target-demo report →</a></p><ul><li><strong>EKS:</strong> select the EKS project for cluster version, managed nodegroup, and CSI add-on evidence. Upgrade insights are deliberately missing.</li><li><strong>Gateway:</strong> select Kubernetes → Network to follow HTTPRoute checkout → Gateway public → Service checkout-api. Inventory references do not establish traffic health.</li><li><strong>Storage:</strong> select Kubernetes → Storage to inspect orders-data → demo-orders-pv → demo-block. Source allows expansion; target does not.</li></ul><p>Use Overview topology, Search, detail drawers, and the theme control to explore the embedded product UI.</p></section><section><h2>Fleet and analysis</h2><p><a href="hub/index.html">Hub fleet sample →</a> — two recorded clusters, exported envelopes, and a full local Hub preview.</p><p><a href="ai/index.html">AI analysis sample →</a> — clearly labeled illustrative output with evidence and limitations.</p></section><section><h2>Reproduce and share</h2><p>Run <code>go run ./scripts/demo</code> from the repository root. Open this file directly or serve <code>docs/demo</code> with any static HTTP server. The source and target directories contain snapshot JSON, deterministic Advisor JSON, Markdown, and self-contained HTML.</p><p>All input is authored in <code>scripts/demo/fixtures</code>. Domains use <code>example.invalid</code>, the fictional AWS account is <code>000000000000</code>, and no real scans, credentials, or Secret values are included.</p></section>`)
}

// previewHandler allows only reads, so the demo cannot load local AI credentials
// or accept new remote-write data through the production Hub's POST endpoints.
func previewHandler(s *hub.Store) http.Handler {
	h := s.Handler(hub.HandlerOptions{})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Synthetic demo is read-only; AI execution and ingestion are disabled.", http.StatusMethodNotAllowed)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func main() {
	out := flag.String("out", "docs/demo", "generated artifact directory")
	serve := flag.Bool("serve", false, "serve the seeded production Hub on loopback port 8091")
	flag.Parse()
	if *serve {
		s, err := seededHub()
		if err != nil {
			log.Fatal(err)
		}
		log.Print("Synthetic recorded Hub: http://127.0.0.1:8091 (Ctrl-C to stop; AI provider disabled)")
		server := &http.Server{Addr: "127.0.0.1:8091", Handler: previewHandler(s), ReadHeaderTimeout: 5 * time.Second}
		log.Fatal(server.ListenAndServe())
	}
	if err := generate(*out); err != nil {
		log.Fatal(err)
	}
	fmt.Println(filepath.Join(*out, "index.html"))
}
