package clicommand

import (
	"context"
	"fmt"
	"time"

	"github.com/buildkite/agent/v3/api"
	"github.com/buildkite/roko"
	"github.com/urfave/cli"
)

const metaDataKeysHelpDescription = `Usage:

   buildkite-agent meta-data keys [options...]

Description:

   Lists all meta-data keys that have been previously set, delimited by a newline
   and terminated with a trailing newline.

Example:

   $ buildkite-agent meta-data keys`

type MetaDataKeysConfig struct {
	Job   string `cli:"job"`
	Build string `cli:"build"`

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

var MetaDataKeysCommand = cli.Command{
	Name:        "keys",
	Usage:       "Lists all meta-data keys that have been previously set",
	Description: metaDataKeysHelpDescription,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "job",
			Value:  "",
			Usage:  "Which job's build should the meta-data be checked for",
			EnvVar: "BUILDKITE_JOB_ID",
		},
		cli.StringFlag{
			Name:   "build",
			Value:  "",
			Usage:  "Which build should the meta-data be retrieved from. --build will take precedence over --job",
			EnvVar: "BUILDKITE_METADATA_BUILD_ID",
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
		cfg, l, _, done := setupLoggerAndConfig[MetaDataKeysConfig](c)
		defer done()

		// Create the API client
		client := api.NewClient(l, loadAPIClientConfig(cfg, "AgentAccessToken"))

		// Find the meta data keys
		var keys []string
		var resp *api.Response

		scope := "job"
		id := cfg.Job

		if cfg.Build != "" {
			scope = "build"
			id = cfg.Build
		}

		if err := roko.NewRetrier(
			roko.WithMaxAttempts(10),
			roko.WithStrategy(roko.Constant(5*time.Second)),
		).DoWithContext(ctx, func(r *roko.Retrier) error {
			var err error
			keys, resp, err = client.MetaDataKeys(ctx, scope, id)
			if resp != nil && (resp.StatusCode == 401 || resp.StatusCode == 404) {
				r.Break()
			}
			if err != nil {
				l.Warn("%s (%s)", err, r)
				return err
			}
			return nil
		}); err != nil {
			l.Fatal("Failed to find meta-data keys: %s", err)
		}

		for _, key := range keys {
			fmt.Fprintf(c.App.Writer, "%s\n", key)
		}
	},
}
