package main

import (
	_ "net/http/pprof"

	"cloud.google.com/go/profiler"
	"github.com/iotaledger/goshimmer/plugins"
	"github.com/iotaledger/hive.go/node"
)

func main() {
	cfg := profiler.Config{
		Service:        "goshimmer",
		ServiceVersion: "1.0.0",
		// ProjectID must be set if not running on GCP.
		// ProjectID: "my-project",

		// For OpenCensus users:
		// To see Profiler agent spans in APM backend,
		// set EnableOCTelemetry to true
		// EnableOCTelemetry: true,
	}

	// Profiler initialization, best done as early as possible.
	if err := profiler.Start(cfg); err != nil {
		// TODO: Handle error.
	}

	node.Run(
		plugins.Core,
		plugins.Research,
		plugins.UI,
		plugins.WebAPI,
	)
}
