package clicommand

import (
	"context"
	"fmt"
	"time"

	"github.com/buildkite/agent/v3/api"
	"github.com/buildkite/roko"
	"github.com/urfave/cli"
)

const stepGetHelpDescription = `Usage:

   buildkite-agent step get <attribute> [options...]

Description:

   Retrieve the value of an attribute in a step. If no attribute is passed, the
   entire step will be returned.

   In the event a complex object is returned (an object or an array),
   you'll need to supply the --format option to tell the agent how it should
   output the data (currently only JSON is supported).

Example:

   $ buildkite-agent step get "label" --step "key"
   $ buildkite-agent step get --format json
   $ buildkite-agent step get "retry" --format json
   $ buildkite-agent step get "state" --step "my-other-step"`

type StepGetConfig struct {
	Attribute string `cli:"arg:0" label:"step attribute"`
	StepOrKey string `cli:"step" validate:"required"`
	Build     string `cli:"build"`
	Format    string `cli:"format"`

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

var StepGetCommand = cli.Command{
	Name:        "get",
	Usage:       "Get the value of an attribute",
	Description: stepGetHelpDescription,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "step",
			Value:  "",
			Usage:  "The step to get. Can be either its ID (BUILDKITE_STEP_ID) or key (BUILDKITE_STEP_KEY)",
			EnvVar: "BUILDKITE_STEP_ID",
		},
		cli.StringFlag{
			Name:   "build",
			Value:  "",
			Usage:  "The build to look for the step in. Only required when targeting a step using its key (BUILDKITE_STEP_KEY)",
			EnvVar: "BUILDKITE_BUILD_ID",
		},
		cli.StringFlag{
			Name:   "format",
			Value:  "",
			Usage:  "The format to output the attribute value in (currently only JSON is supported)",
			EnvVar: "BUILDKITE_STEP_GET_FORMAT",
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
	Action: func(c *cli.Context) error {
		ctx := context.Background()
		cfg, l, _, done := setupLoggerAndConfig[StepGetConfig](c)
		defer done()

		// Create the API client
		client := api.NewClient(l, loadAPIClientConfig(cfg, "AgentAccessToken"))

		// Create the request
		stepExportRequest := &api.StepExportRequest{
			Build:     cfg.Build,
			Attribute: cfg.Attribute,
			Format:    cfg.Format,
		}

		// Find the step attribute
		var resp *api.Response
		var stepExportResponse *api.StepExportResponse
		if err := roko.NewRetrier(
			roko.WithMaxAttempts(10),
			roko.WithStrategy(roko.Constant(5*time.Second)),
		).DoWithContext(ctx, func(r *roko.Retrier) error {
			var err error
			stepExportResponse, resp, err = client.StepExport(ctx, cfg.StepOrKey, stepExportRequest)
			// Don't bother retrying if the response was one of these statuses
			if resp != nil && (resp.StatusCode == 401 || resp.StatusCode == 404 || resp.StatusCode == 400) {
				r.Break()
			}
			if err != nil {
				l.Warn("%s (%s)", err, r)
				return err
			}
			return nil
		}); err != nil {
			l.Fatal("Failed to get step: %s", err)
		}

		// Output the value to STDOUT
		_, err := fmt.Println(c.App.Writer, stepExportResponse.Output)
		return err
	},
}
