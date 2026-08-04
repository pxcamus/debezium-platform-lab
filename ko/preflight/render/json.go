package render

import (
	"encoding/json"
	"io"

	"dbz-mage/ko/preflight"
)

// JSON writes the report as the machine-readable contract consumed by CI. It is the same
// data the text renderer receives, so the two can never disagree about what was found.
func JSON(w io.Writer, report preflight.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
