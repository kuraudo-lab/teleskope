package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/live"
	"github.com/kuraudo-lab/teleskope/internal/report"
	"github.com/kuraudo-lab/teleskope/internal/topology"
)

const defaultFixture = "testdata/recorded/eks-demo-snapshot.json"

type liveDemo struct {
	mu               sync.Mutex
	store            *live.Store
	snapshot         inventory.Snapshot
	page             string
	lastTopologyKind topology.UpdateKind
	lastSince        uint64
}

func main() {
	fixture := flag.String("fixture", defaultFixture, "de-identified snapshot fixture")
	listen := flag.String("serve", "127.0.0.1:8093", "local listen address")
	flag.Parse()

	snapshot, err := loadSnapshot(*fixture)
	if err != nil {
		log.Fatal(err)
	}
	demo, err := newLiveDemo(snapshot)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("topology live continuity: http://%s\n", *listen)
	log.Fatal(http.ListenAndServe(*listen, demo))
}

func loadSnapshot(path string) (inventory.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return inventory.Snapshot{}, err
	}
	var snapshot inventory.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return inventory.Snapshot{}, err
	}
	return snapshot, nil
}

func newLiveDemo(snapshot inventory.Snapshot) (*liveDemo, error) {
	var kubernetesCoverage []inventory.CoverageItem
	for _, item := range snapshot.Coverage {
		if item.Area == "kubernetes" {
			kubernetesCoverage = append(kubernetesCoverage, item)
		}
	}
	snapshot.Coverage = kubernetesCoverage
	store, err := live.New([]live.Source{{Name: "kubernetes", PublicationOnly: true}})
	if err != nil {
		return nil, err
	}
	demo := &liveDemo{store: store, snapshot: snapshot}
	demo.page = strings.Replace(report.LiveHTML(), "</body>", "<script>"+liveContinuityHarness()+"</script></body>", 1)
	demo.publish(false)
	return demo, nil
}

func (d *liveDemo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = strings.NewReader(d.page).WriteTo(w)
	case "/demo/advance":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		d.publish(true)
		w.WriteHeader(http.StatusNoContent)
	case "/demo/gap":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		for range 17 {
			d.publish(false)
		}
		w.WriteHeader(http.StatusNoContent)
	case "/demo/status":
		view, err := d.store.View()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		d.mu.Lock()
		status := map[string]any{"revision": view.Revision, "lastTopologyKind": d.lastTopologyKind, "lastSince": d.lastSince}
		d.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	case "/api/topology":
		since, _ := strconv.ParseUint(r.URL.Query().Get("since"), 10, 64)
		if update, err := d.store.TopologyUpdate(since); err == nil {
			d.mu.Lock()
			d.lastTopologyKind, d.lastSince = update.Kind, since
			d.mu.Unlock()
		}
		d.store.Handler().ServeHTTP(w, r)
	default:
		d.store.Handler().ServeHTTP(w, r)
	}
}

func (d *liveDemo) publish(changeUnselectedWorkload bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snapshot.CollectedAt = d.snapshot.CollectedAt.Add(time.Second)
	for index := range d.snapshot.Coverage {
		d.snapshot.Coverage[index].CollectedAt = d.snapshot.CollectedAt
	}
	if changeUnselectedWorkload && len(d.snapshot.Kubernetes.Workloads) > 0 {
		workload := &d.snapshot.Kubernetes.Workloads[0]
		if workload.ReadyReplicas > 0 {
			workload.ReadyReplicas--
		} else {
			workload.ReadyReplicas++
		}
	}
	d.store.PublishSource(live.SourcePublication{Source: "kubernetes", Mode: live.SourceModeRecorded, Snapshot: &d.snapshot})
}

func liveContinuityHarness() string {
	return `(() => {
  const evidence = {fixture:'recorded EKS',phase:'waiting for initial revision'};
  let output;
  const publish = () => {
    window.__teleskopeLiveEvidence = evidence;
    if (!output) {
      output = document.createElement('pre');
      output.id = 'topologyLiveEvidence';
      output.setAttribute('aria-label', 'Topology live continuity evidence');
      document.querySelector('.topology-workspace .toolbar')?.appendChild(output);
    }
    output.textContent = JSON.stringify(evidence, null, 2);
  };
  const waitFor = async (predicate, label) => {
    const deadline = Date.now() + 10000;
    while (!predicate()) {
      if (Date.now() > deadline) throw new Error('timed out waiting for ' + label);
      await new Promise(resolve => setTimeout(resolve, 50));
    }
  };
  const position = id => {
    const node = [...document.querySelectorAll('[data-node-id]')].find(item => item.dataset.nodeId === id);
    const card = node?.querySelector('.node-card');
    return card ? {x:card.getAttribute('x'),y:card.getAttribute('y')} : null;
  };
  const status = () => fetch('/demo/status', {cache:'no-store'}).then(response => response.json());
  publish();
  (async () => {
    await waitFor(() => liveRevision >= 1 && topologyGraph?.nodes?.length, 'initial graph');
    const selected = topologyGraph.nodes.find(node => node.name === 'aws-node') || topologyGraph.nodes[0];
    topologySemanticView = 'application';
    topologySearch = selected.name;
    byId('topology-search').value = topologySearch;
    selectTopologyNode(selected.id);
    const before = {revision:topologyGraph.revision,selected:topologySelectedNodeId,position:position(selected.id)};

    await fetch('/demo/advance', {method:'POST'});
    await refreshLivePage();
    await waitFor(() => topologyGraph?.revision >= 2, 'accepted delta');
    const deltaStatus = await status();
    const afterDelta = {revision:topologyGraph.revision,selected:topologySelectedNodeId,position:position(selected.id)};

    await fetch('/demo/gap', {method:'POST'});
    await refreshLivePage();
    await waitFor(() => topologyGraph?.revision > 2, 'full reset');
    const resetStatus = await status();

    Object.assign(evidence, {
      phase:'complete',
      selectedObject:selected.name,
      before,
      delta:{responseKind:deltaStatus.lastTopologyKind,accepted:deltaStatus.lastTopologyKind==='delta',revision:afterDelta.revision},
      selectionStable:before.selected===afterDelta.selected,
      layoutStable:JSON.stringify(before.position)===JSON.stringify(afterDelta.position),
      revisionGap:{requestedSince:resetStatus.lastSince,responseKind:resetStatus.lastTopologyKind,fullReset:resetStatus.lastTopologyKind==='full',revision:topologyGraph.revision},
      nodeProbe:false
    });
    publish();
  })().catch(error => { evidence.phase='failed'; evidence.error=error.message; publish(); });
})();`
}
