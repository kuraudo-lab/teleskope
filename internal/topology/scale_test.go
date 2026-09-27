package topology

import (
	"testing"
)

func TestTopologyScaleFixturesAreDeterministicAndValid(t *testing.T) {
	for _, profile := range ScaleProfiles() {
		t.Run(profile.Name, func(t *testing.T) {
			first := BuildScaleFixture(profile, 1)
			second := BuildScaleFixture(profile, 1)
			if err := first.Validate(); err != nil {
				t.Fatal(err)
			}
			if len(first.Nodes) != profile.Nodes || len(first.Edges) != profile.Edges {
				t.Fatalf("fixture shape = %d nodes/%d edges", len(first.Nodes), len(first.Edges))
			}
			firstJSON, err := first.MarshalDeterministic()
			if err != nil {
				t.Fatal(err)
			}
			secondJSON, err := second.MarshalDeterministic()
			if err != nil {
				t.Fatal(err)
			}
			if string(firstJSON) != string(secondJSON) {
				t.Fatal("scale fixture is not deterministic")
			}
		})
	}
}

func BenchmarkTopologyScaleDiff(b *testing.B) {
	for _, profile := range ScaleProfiles() {
		profile := profile
		b.Run(profile.Name, func(b *testing.B) {
			base := BuildScaleFixture(profile, 1)
			next := BuildScaleFixture(profile, 2)
			markScaleChanges(&next)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := Diff(&base, next); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTopologyScaleApply(b *testing.B) {
	for _, profile := range ScaleProfiles() {
		profile := profile
		b.Run(profile.Name, func(b *testing.B) {
			base := BuildScaleFixture(profile, 1)
			next := BuildScaleFixture(profile, 2)
			markScaleChanges(&next)
			update, err := Diff(&base, next)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := Apply(&base, update); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func markScaleChanges(graph *Graph) {
	for i := 0; i < 10 && i < len(graph.Nodes); i++ {
		graph.Nodes[i].Metadata = map[string]string{"phase": "changed"}
	}
}
