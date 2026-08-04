package main

import (
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dmp-lab",
		Short: "Deploy and inspect a Debezium Platform lab environment",
		Long: "dmp-lab sets up the Debezium Management Platform on Kubernetes — locally with Kind, " +
			"or against a cluster you already run.\n\n" +
			"Start with 'dmp-lab doctor', which reports whether this machine can deploy the " +
			"platform before anything is installed.",
		SilenceUsage: true,
	}

	cmd.AddCommand(newDoctorCmd())

	return cmd
}
