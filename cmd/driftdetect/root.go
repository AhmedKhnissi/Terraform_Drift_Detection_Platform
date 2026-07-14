package driftdetect

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command for the CLI.
var rootCmd = &cobra.Command{
	Use:           "driftdetect",
	Short:         "Detect Terraform configuration drift against live cloud infrastructure",
	Long:          "driftdetect compares Terraform state with actual cloud resources to surface deleted, modified, and tag-drifted infrastructure — without running terraform plan or apply.",
	SilenceUsage:  true,
	SilenceErrors: false,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to YAML config file (defaults applied if omitted)")
	rootCmd.PersistentFlags().StringVar(&stateOverride, "state", "", "override the Terraform state source (local path or s3://bucket/key)")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(scheduleCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(historyCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
