package compare

import (
	"strings"
	"testing"
)

func TestDecodeSnapshotCompatibility(t *testing.T) {
	for _, input := range []string{`{"schemaVersion":"teleskope.io/snapshot/v1alpha1","newField":true}`, `{ "kubernetes": {} }`} {
		if _, err := DecodeSnapshot("fixture", strings.NewReader(input)); err != nil {
			t.Fatal(err)
		}
	}
	for _, input := range []string{`{} {}`, `{} trailing`, `{"schemaVersion":"future/v9"}`, `broken`} {
		if _, err := DecodeSnapshot("fixture", strings.NewReader(input)); err == nil {
			t.Fatalf("accepted invalid snapshot: %s", input)
		}
	}
}
