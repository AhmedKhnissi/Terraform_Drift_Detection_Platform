package driftdetect

import (
	"github.com/spf13/cobra"

	"driftdetect/internal/output"
)

var historyLimit int

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "List previously recorded scans",
	RunE:  runHistory,
}

func init() {
	historyCmd.Flags().IntVar(&historyLimit, "limit", 20, "maximum number of scans to list")
}

func runHistory(cmd *cobra.Command, _ []string) error {
	cfg := loadConfig()
	store, err := buildStore(cfg)
	if err != nil {
		return err
	}
	defer store.Close()

	scans, err := store.ListScans(historyLimit)
	if err != nil {
		return err
	}
	output.RenderHistory(cmd.OutOrStdout(), scans)
	return nil
}
