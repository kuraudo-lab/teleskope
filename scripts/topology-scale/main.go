package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/report"
	"github.com/kuraudo-lab/teleskope/internal/topology"
)

func main() {
	profileName := flag.String("profile", "small", "scale profile: small, medium, or large")
	output := flag.String("output", "", "write the self-contained acceptance page to this path")
	listen := flag.String("serve", "", "serve the acceptance page at this address, for example 127.0.0.1:8092")
	flag.Parse()

	profile, ok := topology.LookupScaleProfile(*profileName)
	if !ok {
		log.Fatalf("unknown profile %q (want small, medium, or large)", *profileName)
	}
	page, err := renderScalePage(profile)
	if err != nil {
		log.Fatal(err)
	}
	if *output == "" && *listen == "" {
		log.Fatal("set -output or -serve")
	}
	if *output != "" {
		if err := os.WriteFile(*output, []byte(page), 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Println(*output)
	}
	if *listen != "" {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(page))
		})
		fmt.Printf("topology scale %s: http://%s\n", profile.Name, *listen)
		log.Fatal(http.ListenAndServe(*listen, handler))
	}
}

func renderScalePage(profile topology.ScaleProfile) (string, error) {
	graph := topology.BuildScaleFixture(profile, 1)
	snapshot := &inventory.Snapshot{
		SchemaVersion: "teleskope.io/snapshot/v1alpha1",
		CollectedAt:   graph.GeneratedAt,
		Source:        inventory.Source{Tool: "teleskope", Version: "scale-fixture", Mode: "synthetic/scale"},
		Kubernetes:    inventory.Kubernetes{Context: graph.ClusterID},
	}
	return report.RenderUI(report.UIRenderOptions{
		Mode:     report.UIModeOffline,
		Snapshot: snapshot,
		Topology: &graph,
		Target:   graph.ClusterID,
		Scripts:  []string{scaleHarness(profile)},
	})
}

func scaleHarness(profile topology.ScaleProfile) string {
	return fmt.Sprintf(`(() => {
  const profile = {name:%q,nodes:%d,edges:%d,samples:5};
  const samples = [];
  topologyMode = 'graph';
  syncTopologyURL();
  const publish = () => {
    const ordered = [...samples].sort((a,b) => a-b);
    const evidence = {
      profile,
      rendered: {nodes:Number(byId('topology').dataset.renderNodes),edges:Number(byId('topology').dataset.renderEdges),mode:byId('topology').dataset.renderMode},
      renderMs: {samples,median:ordered[Math.floor(ordered.length/2)],worst:ordered[ordered.length-1]},
      heapBytes: performance.memory?.usedJSHeapSize ?? null,
      graphPolicy: topologyGraphPolicy,
      complexityMessage: byId('topologyComplexityNotice')?.textContent || '',
      viewport: {width:innerWidth,height:innerHeight},
      userAgent: navigator.userAgent
    };
    window.__teleskopeScaleEvidence = evidence;
    let output = byId('topologyScaleEvidence');
    if (!output) {
      output = document.createElement('pre');
      output.id = 'topologyScaleEvidence';
      output.setAttribute('aria-label', 'Topology scale acceptance evidence');
      output.style.cssText = 'position:fixed;right:12px;bottom:12px;z-index:20;width:min(340px,calc(100vw - 24px));max-height:180px;overflow:auto;margin:0;padding:10px;border:1px solid var(--line);border-radius:10px;background:color-mix(in srgb,var(--panel) 94%%,transparent);box-shadow:0 8px 28px rgba(0,0,0,.18);font-size:10px;line-height:1.35;white-space:pre-wrap;pointer-events:none';
      document.body.appendChild(output);
    }
    output.textContent = JSON.stringify(evidence, null, 2);
  };
  const measure = () => {
    renderTopology();
    requestAnimationFrame(() => requestAnimationFrame(() => {
      samples.push(Number(byId('topology').dataset.renderMs));
      if (samples.length < profile.samples) measure(); else publish();
    }));
  };
  measure();
})();`, profile.Name, profile.Nodes, profile.Edges)
}

func profileNames() string {
	var names []string
	for _, profile := range topology.ScaleProfiles() {
		names = append(names, profile.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
