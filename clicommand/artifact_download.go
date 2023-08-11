package clicommand

import (
	"context"

	"github.com/buildkite/agent/v3/agent"
	"github.com/buildkite/agent/v3/api"
	"github.com/urfave/cli"
)

const downloadHelpDescription = `Usage:

   buildkite-agent artifact download [options] <query> <destination>

Description:

   Downloads artifacts matching <query> from Buildkite to <destination>
   directory on the local machine.

   Note: You need to ensure that your search query is surrounded by quotes if
   using a wild card as the built-in shell path globbing will expand the wild
   card and break the query.

   If the last path component of <destination> matches the first path component
   of your <query>, the last component of <destination> is dropped from the
   final path. For example, a query of 'app/logs/*' with a destination of
   'foo/app' will write any matched artifact files to 'foo/app/logs/', relative
   to the current working directory.

   You can also change working directory to the intended destination and use a
   <destination> of '.' to always create a directory hierarchy matching the
   artifact paths.

Example:

   $ buildkite-agent artifact download "pkg/*.tar.gz" . --build xxx

   This will search across all the artifacts for the build with files that match that part.
   The first argument is the search query, and the second argument is the download destination.

   If you're trying to download a specific file, and there are multiple artifacts from different
   jobs, you can target the particular job you want to download the artifact from:

   $ buildkite-agent artifact download "pkg/*.tar.gz" . --step "tests" --build xxx

   You can also use the step's jobs id (provided by the environment variable $BUILDKITE_JOB_ID)`

type ArtifactDownloadConfig struct {
	Query              string `cli:"arg:0" label:"artifact search query" validate:"required"`
	Destination        string `cli:"arg:1" label:"artifact download path" validate:"required"`
	Step               string `cli:"step"`
	Build              string `cli:"build" validate:"required"`
	IncludeRetriedJobs bool   `cli:"include-retried-jobs"`

	// Global flags
	Debug       bool     `cli:"debug"`
	LogLevel    string   `cli:"log-level"`
	NoColor     bool     `cli:"no-color"`
	Experiments []string `cli:"experiment" normalize:"list"`
	Profile     string   `cli:"profile"`

	// API config
	DebugHTTP        bool   `cli:"debug-http"`
	AgentAccessToken string `cli:"agent-access-token" validate:"required"`
	Endpoint         string `cli:"endpoint" validate:"required"`
	NoHTTP2          bool   `cli:"no-http2"`
}

var ArtifactDownloadCommand = cli.Command{
	Name:        "download",
	Usage:       "Downloads artifacts from Buildkite to the local machine",
	Description: downloadHelpDescription,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "step",
			Value: "",
			Usage: "Scope the search to a particular step by using either its name or job ID",
		},
		cli.StringFlag{
			Name:   "build",
			Value:  "",
			EnvVar: "BUILDKITE_BUILD_ID",
			Usage:  "The build that the artifacts were uploaded to",
		},
		cli.BoolFlag{
			Name:   "include-retried-jobs",
			EnvVar: "BUILDKITE_AGENT_INCLUDE_RETRIED_JOBS",
			Usage:  "Include artifacts from retried jobs in the search",
		},

		// API Flags
		AgentAccessTokenFlag,
		EndpointFlag,
		NoHTTP2Flag,
		DebugHTTPFlag,

		// Global flags
		NoColorFlag,
		DebugFlag,
		LogLevelFlag,
		ExperimentsFlag,
		ProfileFlag,
	},
	Action: func(c *cli.Context) {
		ctx := context.Background()
		cfg, l, _, done := setupLoggerAndConfig[ArtifactDownloadConfig](c)
		defer done()

		// Create the API client
		client := api.NewClient(l, loadAPIClientConfig(cfg, "AgentAccessToken"))

		// Setup the downloader
		downloader := agent.NewArtifactDownloader(l, client, agent.ArtifactDownloaderConfig{
			Query:              cfg.Query,
			Destination:        cfg.Destination,
			BuildID:            cfg.Build,
			Step:               cfg.Step,
			IncludeRetriedJobs: cfg.IncludeRetriedJobs,
			DebugHTTP:          cfg.DebugHTTP,
		})

		// Download the artifacts
		if err := downloader.Download(ctx); err != nil {
			l.Fatal("Failed to download artifacts: %s", err)
		}
	},
}
