package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"dbz-mage/ko/automation"
	"dbz-mage/ko/preflight"
	"dbz-mage/ko/preflight/render"
)

// exitFindingsFailed is returned when the machine is not ready. It is distinct from 1, so
// CI can tell "the preflight found problems" apart from "the preflight itself broke".
const exitFindingsFailed = 2

func newDoctorCmd() *cobra.Command {
	var (
		asJSON   bool
		timeout  time.Duration
		targetIn string
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check whether this machine can deploy the platform",
		Long: "Inspects the machine and reports what is missing or misconfigured, with the exact " +
			"command to fix each finding.\n\n" +
			"Nothing is installed and nothing is changed. Every finding carries a stable " +
			"identifier and a link to the section of the documentation that explains it.\n\n" +
			"Several checks depend on where the platform will run. With --target local the " +
			"workloads run on this machine, so its architecture, memory, kernel limits and free " +
			"ports all matter. With --target remote they run on a cluster this machine only " +
			"talks to, and host capacity stops being a constraint. The default is taken from " +
			"CLUSTER_TYPE.\n\n" +
			"Exit codes: 0 if the machine is ready (warnings do not block), 2 if a check failed.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := automation.LoadEnv(); err != nil {
				return fmt.Errorf("loading environment: %w", err)
			}

			target := preflight.DetectTarget()
			if targetIn != "" {
				parsed, err := preflight.ParseTarget(targetIn)
				if err != nil {
					return err
				}
				target = parsed
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			report := preflight.Run(ctx, preflight.Catalog(target))

			write := render.Text
			if asJSON {
				write = render.JSON
			}
			if err := write(cmd.OutOrStdout(), report); err != nil {
				return err
			}

			if !report.OK() {
				os.Exit(exitFindingsFailed)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&targetIn, "target", "", "where the platform will run: local or remote (default from CLUSTER_TYPE)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit findings as JSON for scripting and CI")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "budget for the whole run, including network probes")

	return cmd
}
