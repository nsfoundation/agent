package clicommand

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/buildkite/agent/v3/env"
	"github.com/buildkite/agent/v3/jobapi"
	"github.com/urfave/cli"
)

const envClientErrMessage = `Could not create Job API client: %v
This command can only be used from hooks or plugins running under a job executor
where the "job-api" experiment is enabled.
`

const envGetHelpDescription = `Usage:

   buildkite-agent env get [variables]

Description:
   Retrieves environment variables and their current values from the current job
   execution environment.

   Note that this subcommand is only available from within the job executor with
   the ′job-api′ experiment enabled.

   Changes to the job environment only apply to the environments of subsequent
   phases of the job. However, ′env get′ can be used to inspect the changes made
   with ′env set′ and ′env unset′.

Examples:
   Getting all variables in key=value format:

   $ buildkite-agent env get
   ALPACA=Geronimo the Incredible
   BUILDKITE=true
   LLAMA=Kuzco
   ...

   Getting the value of one variable:

   $ buildkite-agent env get LLAMA
   Kuzco

   Getting multiple specific variables:

   $ buildkite-agent env get LLAMA ALPACA
   ALPACA=Geronimo the Incredible
   LLAMA=Kuzco

   Getting variables as a JSON object:

   $ buildkite-agent env get --format=json-pretty
   {
     "ALPACA": "Geronimo the Incredible",
     "BUILDKITE": "true",
     "LLAMA": "Kuzco",
     ...
   }
`

type EnvGetConfig struct {
	Format string `cli:"format"`

	// Global flags
	Debug       bool     `cli:"debug"`
	LogLevel    string   `cli:"log-level"`
	NoColor     bool     `cli:"no-color"`
	Experiments []string `cli:"experiment" normalize:"list"`
	Profile     string   `cli:"profile"`
}

var EnvGetCommand = cli.Command{
	Name:        "get",
	Usage:       "Gets variables from the job execution environment",
	Description: envGetHelpDescription,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "format",
			Usage:  "Output format: plain, json, or json-pretty",
			EnvVar: "BUILDKITE_AGENT_ENV_GET_FORMAT",
			Value:  "plain",
		},

		// Global flags
		NoColorFlag,
		DebugFlag,
		LogLevelFlag,
		ExperimentsFlag,
		ProfileFlag,
	},
	Action: envGetAction,
}

func envGetAction(c *cli.Context) error {
	ctx := context.Background()
	cfg, l, _, done := setupLoggerAndConfig[EnvGetConfig](c)
	defer done()

	client, err := jobapi.NewDefaultClient(ctx)
	if err != nil {
		l.Fatal(envClientErrMessage, err)
	}

	envMap, err := client.EnvGet(ctx)
	if err != nil {
		l.Fatal("Couldn't fetch the job executor environment variables: %v\n", err)
	}

	notFound := false

	// Filter envMap by any remaining args.
	if len(c.Args()) > 0 {
		em := make(map[string]string)
		for _, arg := range c.Args() {
			v, ok := envMap[arg]
			if !ok {
				notFound = true
				l.Warn("%q is not set", arg)
				continue
			}
			em[arg] = v
		}
		envMap = em
	}

	switch cfg.Format {
	case "plain":
		if len(c.Args()) == 1 {
			// Just print the value.
			for _, v := range envMap {
				fmt.Fprintln(c.App.Writer, v)
			}
			break
		}

		// Print everything.
		for _, v := range env.FromMap(envMap).ToSlice() {
			fmt.Fprintln(c.App.Writer, v)
		}

	case "json", "json-pretty":
		enc := json.NewEncoder(c.App.Writer)
		if c.String("format") == "json-pretty" {
			enc.SetIndent("", "  ")
		}
		if err := enc.Encode(envMap); err != nil {
			l.Fatal("Error marshalling JSON: %v\n", err)
		}

	default:
		l.Error("Invalid output format %q\n", cfg.Format)
	}

	if notFound {
		done()
		os.Exit(1)
	}

	return nil
}
