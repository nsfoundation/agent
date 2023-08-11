package clicommand

import (
	"context"
	"fmt"
	"os"

	"github.com/buildkite/agent/v3/lock"
	"github.com/urfave/cli"
)

const lockGetHelpDescription = `Usage:

   buildkite-agent lock get [key]

Description:
   Retrieves the value of a lock key. Any key not in use returns an empty
   string.

   Note that this subcommand is only available when an agent has been started
   with the ′agent-api′ experiment enabled.

   ′lock get′ is generally only useful for inspecting lock state, as the value
   can change concurrently. To acquire or release a lock, use ′lock acquire′ and
   ′lock release′.

Examples:

   $ buildkite-agent lock get llama
   Kuzco

`

type LockGetConfig struct {
	// Common config options
	LockScope   string `cli:"lock-scope"`
	SocketsPath string `cli:"sockets-path" normalize:"filepath"`

	// Global flags
	Debug       bool     `cli:"debug"`
	LogLevel    string   `cli:"log-level"`
	NoColor     bool     `cli:"no-color"`
	Experiments []string `cli:"experiment" normalize:"list"`
	Profile     string   `cli:"profile"`
}

var LockGetCommand = cli.Command{
	Name:        "get",
	Usage:       "Gets a lock value from the agent leader",
	Description: lockGetHelpDescription,
	Flags:       append(globalFlags(), lockCommonFlags...),
	Action:      lockGetAction,
}

func lockGetAction(c *cli.Context) error {
	if c.NArg() != 1 {
		fmt.Fprint(c.App.ErrWriter, lockGetHelpDescription)
		os.Exit(1)
	}
	key := c.Args()[0]

	ctx := context.Background()
	cfg, l, _, done := setupLoggerAndConfig[LockGetConfig](c)
	defer done()

	if cfg.LockScope != "machine" {
		l.Fatal("Only 'machine' scope for locks is supported in this version.")
	}

	client, err := lock.NewClient(ctx, cfg.SocketsPath)
	if err != nil {
		l.Fatal(lockClientErrMessage, err)
	}

	v, err := client.Get(ctx, key)
	if err != nil {
		l.Fatal("Couldn't get lock state: %v", err)
	}

	fmt.Fprintln(c.App.Writer, v)
	return nil
}
