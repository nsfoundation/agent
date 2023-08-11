package clicommand

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/buildkite/agent/v3/env"
	"github.com/buildkite/agent/v3/jobapi"
	"github.com/urfave/cli"
)

const envSetHelpDescription = `Usage:

   buildkite-agent env set [variable]

Description:
   Sets environment variables in the current job execution environment.
   Changes to the job environment variables only apply to subsequent phases of the job.
   This command cannot unset Buildkite read-only variables.

   To read the new values of variables from within the current phase, use ′env get′.

   Note that this subcommand is only available from within the job executor with the job-api experiment enabled.

Examples:
   Setting the variables ′LLAMA′ and ′ALPACA′:

   $ buildkite-agent env set LLAMA=Kuzco "ALPACA=Geronimo the Incredible"
   Added:
   + LLAMA
   Updated:
   ~ ALPACA

   Setting the variables ′LLAMA′ and ′ALPACA′ using a JSON object supplied over standard input:

   $ echo '{"ALPACA":"Geronimo the Incredible","LLAMA":"Kuzco"}' | buildkite-agent env set --input-format=json --output-format=quiet -
`

type EnvSetConfig struct {
	InputFormat  string `cli:"input-format"`
	OutputFormat string `cli:"output-format"`

	// Global flags
	Debug       bool     `cli:"debug"`
	LogLevel    string   `cli:"log-level"`
	NoColor     bool     `cli:"no-color"`
	Experiments []string `cli:"experiment" normalize:"list"`
	Profile     string   `cli:"profile"`
}

var EnvSetCommand = cli.Command{
	Name:        "set",
	Usage:       "Sets variables in the job execution environment",
	Description: envSetHelpDescription,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "input-format",
			Usage:  "Input format: plain or json",
			EnvVar: "BUILDKITE_AGENT_ENV_SET_INPUT_FORMAT",
			Value:  "plain",
		},
		cli.StringFlag{
			Name:   "output-format",
			Usage:  "Output format: quiet (no output), plain, json, or json-pretty",
			EnvVar: "BUILDKITE_AGENT_ENV_SET_OUTPUT_FORMAT",
			Value:  "plain",
		},

		// Global flags
		NoColorFlag,
		DebugFlag,
		LogLevelFlag,
		ExperimentsFlag,
		ProfileFlag,
	},
	Action: envSetAction,
}

func envSetAction(c *cli.Context) error {
	ctx := context.Background()
	cfg, l, _, done := setupLoggerAndConfig[EnvSetConfig](c)
	defer done()

	client, err := jobapi.NewDefaultClient(ctx)
	if err != nil {
		l.Fatal(envClientErrMessage, err)
	}

	req := &jobapi.EnvUpdateRequest{
		Env: make(map[string]string),
	}

	var parse func(string) error

	switch cfg.InputFormat {
	case "plain":
		parse = func(input string) error {
			e, v, ok := env.Split(input)
			if !ok {
				return fmt.Errorf("%q is not in key=value format", input)
			}
			req.Env[e] = v
			return nil
		}

	case "json":
		// Parse directly into the map
		parse = func(input string) error {
			return json.Unmarshal([]byte(input), &req.Env)
		}

	default:
		l.Fatal("Invalid input format %q\n", c.String("input-format"))
	}

	// Inspect each arg, which could either be "-" for stdin, or "KEY=value"
	for _, arg := range c.Args() {
		if arg == "-" {
			// TODO: replace with c.App.Reader (or something like that) when we upgrade to urfave/cli v3
			sc := bufio.NewScanner(os.Stdin)
			line := 1
			for sc.Scan() {
				if err := parse(sc.Text()); err != nil {
					l.Fatal("Couldn't parse input line %d: %v\n", line, err)
				}
				line++
			}
			if err := sc.Err(); err != nil {
				l.Fatal("Couldn't scan the input buffer: %v\n", err)
			}
			continue
		}
		// Parse args directly
		if err := parse(arg); err != nil {
			l.Fatal("Couldn't parse the command-line argument %q: %v\n", arg, err)
		}
	}

	resp, err := client.EnvUpdate(ctx, req)
	if err != nil {
		l.Error("Couldn't update the job executor environment: %v\n", err)
	}

	switch cfg.OutputFormat {
	case "quiet":
		return nil

	case "plain":
		if len(resp.Added) > 0 {
			fmt.Fprintln(c.App.Writer, "Added:")
			for _, a := range resp.Added {
				fmt.Fprintf(c.App.Writer, "+ %s\n", a)
			}
		}
		if len(resp.Updated) > 0 {
			fmt.Fprintln(c.App.Writer, "Updated:")
			for _, u := range resp.Updated {
				fmt.Fprintf(c.App.Writer, "~ %s\n", u)
			}
		}
		if len(resp.Added) == 0 && len(resp.Updated) == 0 {
			fmt.Fprintln(c.App.Writer, "No variables added or updated.")
		}

	case "json", "json-pretty":
		enc := json.NewEncoder(c.App.Writer)
		if c.String("output-format") == "json-pretty" {
			enc.SetIndent("", "  ")
		}
		if err := enc.Encode(resp); err != nil {
			l.Fatal("Error marshalling JSON: %v\n", err)
		}

	default:
		l.Fatal("Invalid output format %q\n", c.String("output-format"))
	}

	return nil
}
