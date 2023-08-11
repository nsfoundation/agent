package clicommand

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/buildkite/agent/v3/lock"
	"github.com/urfave/cli"
)

const lockDoHelpDescription = `Usage:

   buildkite-agent lock do [key]

Description:
   Begins a do-once lock. Do-once can be used by multiple processes to
   wait for completion of some shared work, where only one process should do
   the work.

   Note that this subcommand is only available when an agent has been started
   with the ′agent-api′ experiment enabled.

   ′lock do′ will do one of two things:

   - Print 'do'. The calling process should proceed to do the work and then
     call ′lock done′.
   - Wait until the work is marked as done (with ′lock done′) and print 'done'.

   If ′lock do′ prints 'done' immediately, the work was already done.

Examples:

   #!/bin/bash
   if [ $(buildkite-agent lock do llama) = 'do' ] ; then
      setup_code()
      buildkite-agent lock done llama
   fi

`

type LockDoConfig struct {
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

func lockDoFlags() []cli.Flag {
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

var LockDoCommand = cli.Command{
	Name:        "do",
	Usage:       "Begins a do-once lock",
	Description: lockDoHelpDescription,
	Flags:       lockDoFlags(),
	Action:      lockDoAction,
}

func lockDoAction(c *cli.Context) error {
	if c.NArg() != 1 {
		fmt.Fprint(c.App.ErrWriter, lockDoHelpDescription)
		os.Exit(1)
	}
	key := c.Args()[0]

	ctx := context.Background()
	cfg, l, _, done := setupLoggerAndConfig[LockDoConfig](c)
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

	do, err := client.DoOnceStart(ctx, key)
	if err != nil {
		l.Fatal("Couldn't start do-once lock: %v\n", err)
	}

	if do {
		fmt.Fprintln(c.App.Writer, "do")
		return nil
	}
	fmt.Fprintln(c.App.Writer, "done")
	return nil
}
