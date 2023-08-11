package clicommand

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/buildkite/agent/v3/lock"
	"github.com/urfave/cli"
)

const lockAcquireHelpDescription = `Usage:

   buildkite-agent lock acquire [key]

Description:
   Acquires the lock for the given key. ′lock acquire′ will wait (potentially
   forever) until it can acquire the lock, if the lock is already held by
   another process. If multiple processes are waiting for the same lock, there
   is no ordering guarantee of which one will be given the lock next.

   To prevent separate processes unlocking each other, the output from ′lock
   acquire′ should be stored, and passed to ′lock release′.

   Note that this subcommand is only available when an agent has been started
   with the ′agent-api′ experiment enabled.

Examples:

   $ token=$(buildkite-agent lock acquire llama)
   $ critical_section()
   $ buildkite-agent lock release llama "${token}"

`

type LockAcquireConfig struct {
	// Common config options
	LockScope   string `cli:"lock-scope"`
	SocketsPath string `cli:"sockets-path" normalize:"filepath"`

	LockWaitTimeout time.Duration `cli:"lock-wait-timeout"`

	// Global flags
	Debug       bool     `cli:"debug"`
	LogLevel    string   `cli:"log-level"`
	NoColor     bool     `cli:"no-color"`
	Experiments []string `cli:"experiment" normalize:"list"`
	Profile     string   `cli:"profile"`
}

func lockAcquireFlags() []cli.Flag {
	flags := append(
		[]cli.Flag{
			cli.DurationFlag{
				Name:   "lock-wait-timeout",
				Usage:  "If specified, sets a maximum duration to wait for a lock before giving up",
				EnvVar: "BUILDKITE_LOCK_WAIT_TIMEOUT",
			},
		},
		lockCommonFlags...,
	)
	return append(flags, globalFlags()...)
}

var LockAcquireCommand = cli.Command{
	Name:        "acquire",
	Usage:       "Acquires a lock from the agent leader",
	Description: lockAcquireHelpDescription,
	Flags:       lockAcquireFlags(),
	Action:      lockAcquireAction,
}

func lockAcquireAction(c *cli.Context) error {
	if c.NArg() != 1 {
		fmt.Fprint(c.App.ErrWriter, lockAcquireHelpDescription)
		os.Exit(1)
	}
	key := c.Args()[0]

	ctx := context.Background()
	cfg, l, _, done := setupLoggerAndConfig[LockAcquireConfig](c)
	defer done()

	if cfg.LockScope != "machine" {
		l.Fatal("Only 'machine' scope for locks is supported in this version.")
	}

	if cfg.LockWaitTimeout != 0 {
		cctx, canc := context.WithTimeout(ctx, cfg.LockWaitTimeout)
		defer canc()
		ctx = cctx
	}

	client, err := lock.NewClient(ctx, cfg.SocketsPath)
	if err != nil {
		l.Fatal(lockClientErrMessage, err)
	}

	token, err := client.Lock(ctx, key)
	if err != nil {
		l.Fatal("Could not acquire lock: %v\n", err)
	}

	fmt.Fprintln(c.App.Writer, token)
	return nil
}
