package clicommand

import (
	"context"
	"fmt"
	"os"

	"github.com/buildkite/agent/v3/lock"
	"github.com/urfave/cli"
)

const lockReleaseHelpDescription = `Usage:

   buildkite-agent lock release [key] [token]

Description:
   Releases the lock for the given key. This should only be called by the
   process that acquired the lock. To help prevent different processes unlocking
   each other unintentionally, the output from ′lock acquire′ is required as the
   second argument.

   Note that this subcommand is only available when an agent has been started
   with the ′agent-api′ experiment enabled.

Examples:

   $ token=$(buildkite-agent lock acquire llama)
   $ critical_section()
   $ buildkite-agent lock release llama "${token}"

`

type LockReleaseConfig struct {
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

var LockReleaseCommand = cli.Command{
	Name:        "release",
	Usage:       "Releases a previously-acquired lock",
	Description: lockReleaseHelpDescription,
	Flags:       append(globalFlags(), lockCommonFlags...),
	Action:      lockReleaseAction,
}

func lockReleaseAction(c *cli.Context) error {
	if c.NArg() != 2 {
		fmt.Fprint(c.App.ErrWriter, lockReleaseHelpDescription)
		os.Exit(1)
	}
	key, token := c.Args()[0], c.Args()[1]

	ctx := context.Background()
	cfg, l, _, done := setupLoggerAndConfig[LockReleaseConfig](c)
	defer done()

	if cfg.LockScope != "machine" {
		l.Fatal("Only 'machine' scope for locks is supported in this version.")
	}

	client, err := lock.NewClient(ctx, cfg.SocketsPath)
	if err != nil {
		l.Fatal(lockClientErrMessage, err)
	}

	if err := client.Unlock(ctx, key, token); err != nil {
		l.Fatal("Could not release lock: %v", err)
	}

	return nil
}
