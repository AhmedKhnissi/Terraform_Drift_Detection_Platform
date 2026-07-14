// Package driftdetect implements the command-line interface for the Terraform
// drift detection platform (scan, serve, schedule, report, history).
package driftdetect

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"

	"driftdetect/internal/cloud"
	awsprovider "driftdetect/internal/cloud/aws"
	"driftdetect/internal/config"
	"driftdetect/internal/drift"
	"driftdetect/internal/engine"
	"driftdetect/internal/state"
	"driftdetect/internal/storage"
)

// cfgFile and stateOverride are the persistent global flags.
var (
	cfgFile       string
	stateOverride string
)

// loadConfig loads the configuration, honoring the --config flag.
func loadConfig() *config.Config {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		// A missing/invalid config file is fatal for every command.
		panic(err)
	}
	return cfg
}

// effectiveState returns the state source: the --state override if present,
// otherwise the configured default.
func effectiveState(cfg *config.Config) string {
	if stateOverride != "" {
		return stateOverride
	}
	return cfg.State.Source
}

// buildAWSConfig resolves the AWS SDK configuration from the platform config.
func buildAWSConfig(ctx context.Context, cfg *config.Config) (aws.Config, error) {
	return awsprovider.LoadConfig(ctx, cfg.AWS)
}

// buildSource builds the state source for the current config.
func buildSource(ctx context.Context, cfg *config.Config, awsCfg aws.Config) (state.StateSource, error) {
	return state.NewSource(effectiveState(cfg), awsCfg)
}

// buildProvider returns the AWS cloud provider.
func buildProvider(awsCfg aws.Config) cloud.CloudProvider {
	return awsprovider.New(awsCfg)
}

// buildStore opens the configured SQLite store.
func buildStore(cfg *config.Config) (storage.Store, error) {
	return storage.OpenSQLite(cfg.Storage.Path)
}

// buildOptions maps the drift config to comparator options.
func buildOptions(cfg *config.Config) drift.DriftOptions {
	return drift.DriftOptions{
		CompareAttributes: cfg.Drift.CompareAttributes,
		CompareTags:       cfg.Drift.CompareTags,
		DetectOrphans:     cfg.Drift.DetectOrphans,
	}
}

// buildRunner constructs a scan runner for the current config.
func buildRunner(ctx context.Context, cfg *config.Config, awsCfg aws.Config) (*engine.Runner, error) {
	src, err := buildSource(ctx, cfg, awsCfg)
	if err != nil {
		return nil, err
	}
	return &engine.Runner{
		Source:   src,
		Provider: buildProvider(awsCfg),
		Options:  buildOptions(cfg),
	}, nil
}
