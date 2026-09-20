package analysis

import "testing"

func TestClientRequestValidateRevision(t *testing.T) {
	for _, revision := range []uint64{0, 7} {
		if err := (ClientRequest{Revision: revision}).ValidateRevision(7); err != nil {
			t.Fatalf("revision %d should be accepted: %v", revision, err)
		}
	}
	if err := (ClientRequest{Revision: 6}).ValidateRevision(7); err == nil {
		t.Fatal("stale revision was accepted")
	}
}
