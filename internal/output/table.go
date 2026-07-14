package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"driftdetect/internal/model"
)

// RenderTable writes a human-readable drift report to w: a summary header
// followed by one row per drift item.
func RenderTable(w io.Writer, report model.DriftReport) {
	fmt.Fprintf(w, "Scan %s  @ %s  (%d ms)\n", report.ScanID, report.StartedAt.Format("2006-01-02 15:04:05"), report.DurationMs)
	fmt.Fprintf(w, "Resources scanned: %d   Drift detected: %d\n", report.ResourceCount, report.DriftCount)
	fmt.Fprintln(w, "Summary:")
	for _, dt := range model.AllDriftTypes {
		if n, ok := report.Summary[dt]; ok && n > 0 {
			fmt.Fprintf(w, "  %-12s %d\n", dt, n)
		}
	}
	if len(report.Items) == 0 {
		fmt.Fprintln(w, "No drift detected. Infrastructure matches Terraform state.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "TYPE\tNAME\tID\tDRIFT\tDETAIL\tEXPECTED -> ACTUAL")
	for _, it := range report.Items {
		detail := it.Attribute
		if it.Message != "" {
			detail = it.Message
		}
		ea := fmt.Sprintf("%v -> %v", it.Expected, it.Actual)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			it.Type, it.Name, it.ID, it.DriftType, detail, ea)
	}
	_ = tw.Flush()
}

// RenderHistory writes a list of scan summaries to w.
func RenderHistory(w io.Writer, scans []model.ScanSummary) {
	if len(scans) == 0 {
		fmt.Fprintln(w, "No scans recorded yet.")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "SCAN ID\tSTARTED\tDURATION(ms)\tRESOURCES\tDRIFT")
	for _, s := range scans {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\n",
			s.ScanID, s.StartedAt.Format("2006-01-02 15:04:05"), s.DurationMs, s.ResourceCount, s.DriftCount)
	}
	_ = tw.Flush()
}
