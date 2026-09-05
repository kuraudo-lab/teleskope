// Package render writes inventory snapshots in supported presentation formats.
package render

import (
	"encoding/json"
	"io"

	"github.com/kuraudo-lab/teleskope/internal/inventory"
)

// JSON writes a pretty JSON snapshot.
func JSON(w io.Writer, snapshot *inventory.Snapshot) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(snapshot)
}
