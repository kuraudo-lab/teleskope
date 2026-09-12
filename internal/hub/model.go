// Package hub stores cluster envelopes published by live collectors.
package hub

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/advisor"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/live"
)

// Cluster identifies one remote cluster in a hub fleet.
type Cluster struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider,omitempty"`
	Region   string `json:"region,omitempty"`
}

// Envelope is the remote-write document exchanged between serve collectors and a hub.
type Envelope struct {
	Cluster     Cluster                `json:"cluster"`
	Revision    uint64                 `json:"revision"`
	CollectedAt time.Time              `json:"collectedAt"`
	Snapshot    *inventory.Snapshot    `json:"snapshot,omitempty"`
	Advisor     *advisor.Report        `json:"advisor,omitempty"`
	Sources     map[string]live.Status `json:"sources,omitempty"`
	Events      []live.Event           `json:"events,omitempty"`
}

// Summary is the hub's compact fleet row for one cluster.
type Summary struct {
	Cluster     Cluster                `json:"cluster"`
	Revision    uint64                 `json:"revision"`
	CollectedAt time.Time              `json:"collectedAt"`
	State       string                 `json:"state"`
	Nodes       int                    `json:"nodes"`
	Workloads   int                    `json:"workloads"`
	Pods        int                    `json:"pods"`
	Images      int                    `json:"images"`
	Events      int                    `json:"events"`
	Advisor     string                 `json:"advisor,omitempty"`
	Sources     map[string]live.Status `json:"sources,omitempty"`
}

// FleetEvent records one recent event annotated with the cluster that reported it.
type FleetEvent struct {
	Cluster Cluster    `json:"cluster"`
	Event   live.Event `json:"event"`
}

// DeriveCluster creates stable cluster metadata from explicit input and snapshot evidence.
func DeriveCluster(snapshot *inventory.Snapshot, explicitID, explicitName, provider string) Cluster {
	cluster := Cluster{
		ID:       strings.TrimSpace(explicitID),
		Name:     strings.TrimSpace(explicitName),
		Provider: strings.TrimSpace(provider),
	}
	if snapshot == nil {
		if cluster.Name == "" {
			cluster.Name = cluster.ID
		}
		return cluster
	}
	if cluster.Provider == "" && snapshot.EKS.Cluster.Name != "" {
		cluster.Provider = "eks"
	}
	if cluster.Provider == "" {
		cluster.Provider = "kubernetes"
	}
	if cluster.Name == "" {
		cluster.Name = firstNonEmpty(snapshot.EKS.Cluster.Name, snapshot.Kubernetes.Context, cluster.ID)
	}
	cluster.Region = snapshot.AWS.Region
	if cluster.ID == "" {
		cluster.ID = firstNonEmpty(snapshot.EKS.Cluster.ARN, snapshot.EKS.Cluster.Name)
	}
	if cluster.ID == "" {
		cluster.ID = kubernetesFallbackID(snapshot.Kubernetes)
	}
	return cluster
}

// CompleteEnvelope fills derived fields and advisor data before publication.
func CompleteEnvelope(env Envelope) (Envelope, error) {
	env.Cluster.ID = strings.TrimSpace(env.Cluster.ID)
	env.Cluster.Name = strings.TrimSpace(env.Cluster.Name)
	env.Cluster.Provider = strings.TrimSpace(env.Cluster.Provider)
	env.Cluster.Region = strings.TrimSpace(env.Cluster.Region)
	if env.Snapshot != nil {
		derived := DeriveCluster(env.Snapshot, env.Cluster.ID, env.Cluster.Name, env.Cluster.Provider)
		env.Cluster = mergeCluster(env.Cluster, derived)
		if env.CollectedAt.IsZero() {
			env.CollectedAt = env.Snapshot.CollectedAt
		}
		if env.Advisor == nil {
			report := advisor.Analyze(env.Snapshot)
			if state := sourceFreshness(env.Sources); state != "" {
				report.SetFreshness(state)
			}
			env.Advisor = &report
		}
	}
	if env.Cluster.ID == "" {
		return Envelope{}, fmt.Errorf("cluster.id is required")
	}
	if env.Cluster.Name == "" {
		env.Cluster.Name = env.Cluster.ID
	}
	if env.Revision == 0 {
		return Envelope{}, fmt.Errorf("revision must be positive")
	}
	if env.CollectedAt.IsZero() {
		env.CollectedAt = time.Now().UTC()
	}
	return env, nil
}

func mergeCluster(base, derived Cluster) Cluster {
	if base.ID == "" {
		base.ID = derived.ID
	}
	if base.Name == "" {
		base.Name = derived.Name
	}
	if base.Provider == "" {
		base.Provider = derived.Provider
	}
	if base.Region == "" {
		base.Region = derived.Region
	}
	return base
}

func kubernetesFallbackID(k inventory.Kubernetes) string {
	seed := firstNonEmpty(k.Server, k.Context)
	if seed == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(seed))
	return "kubernetes:" + hex.EncodeToString(hash[:])[:16]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sourceFreshness(sources map[string]live.Status) string {
	if len(sources) == 0 {
		return ""
	}
	states := make([]string, 0, len(sources))
	for _, status := range sources {
		states = append(states, status.State)
	}
	sort.Strings(states)
	for _, state := range []string{"stale", "partial", "error", "loading"} {
		for _, got := range states {
			if got == state {
				return state
			}
		}
	}
	return "ready"
}

func summarize(env Envelope) Summary {
	summary := Summary{
		Cluster:     env.Cluster,
		Revision:    env.Revision,
		CollectedAt: env.CollectedAt,
		State:       sourceFreshness(env.Sources),
		Events:      len(env.Events),
		Sources:     env.Sources,
	}
	if summary.State == "" {
		summary.State = "ready"
	}
	if env.Snapshot != nil {
		k := env.Snapshot.Kubernetes
		summary.Nodes = len(k.Nodes)
		summary.Workloads = len(k.Workloads)
		summary.Pods = len(k.Pods)
		summary.Images = len(k.RunningImages)
	}
	if env.Advisor != nil {
		summary.Advisor = env.Advisor.Summary
	}
	return summary
}
