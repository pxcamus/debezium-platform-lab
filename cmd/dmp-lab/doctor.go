package main

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	"dbz-mage/ko/preflight"
	"dbz-mage/ko/preflight/render"
)

// exitFindingsFailed is returned when the machine is not ready. It is distinct from 1, so
// CI can tell "the preflight found problems" apart from "the preflight itself broke".
const exitFindingsFailed = 2

func newDoctorCmd() *cobra.Command {
	var (
		asJSON  bool
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check whether this machine can deploy the platform",
		Long: "Inspects the machine and reports what is missing or misconfigured, with the exact " +
			"command to fix each finding.\n\n" +
			"Nothing is installed and nothing is changed. Every finding carries a stable " +
			"identifier and a link to the section of the documentation that explains it.\n\n" +
			"Exit codes: 0 if the machine is ready (warnings do not block), 2 if a check failed.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			report := preflight.Run(ctx, preflight.Catalog())

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

	cmd.Flags().BoolVar(&asJSON, "json", false, "emit findings as JSON for scripting and CI")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "budget for the whole run, including network probes")

	return cmd
}
