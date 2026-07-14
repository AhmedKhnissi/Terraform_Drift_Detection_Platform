// Package output renders drift reports for human (table) and machine (JSON)
// consumption.
package output

import (
	"encoding/json"
	"fmt"
	"os"

	"driftdetect/internal/model"
)

// ToJSON marshals a report with indentation for readability.
func ToJSON(report model.DriftReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// WriteJSON writes the report as indented JSON to the given file path.
func WriteJSON(path string, report model.DriftReport) error {
	b, err := ToJSON(report)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// WriteJSONStdout writes the report as JSON to stdout (convenience for tests/CLI).
func WriteJSONStdout(report model.DriftReport) error {
	b, err := ToJSON(report)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(b))
	return nil
}
