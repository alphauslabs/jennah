package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jennah",
	Short: "Jennah CLI",
	Long: "-------------------------------------------------------------------\n" +
		"                           Jennah CLI\n" +
		"-------------------------------------------------------------------",
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	cobra.EnableCommandSorting = false

	rootCmd.PersistentFlags().String("provider", "", "OAuth provider, default: google (or JENNAH_PROVIDER env var)")
	rootCmd.PersistentFlags().String("gateway", "", "Gateway URL (or JENNAH_GATEWAY env var)")
	rootCmd.PersistentFlags().MarkHidden("provider")
	rootCmd.PersistentFlags().MarkHidden("gateway")

	// Gateway flag (can also be set via env var)
	rootCmd.PersistentFlags().String("gateway", "", "Gateway URL (or JENNAH_GATEWAY_URL env var)")

	// Spanner connection flags (can also be set via env vars)
	rootCmd.PersistentFlags().String("project", "", "GCP project ID (or JENNAH_PROJECT env var)")
	rootCmd.PersistentFlags().String("instance", "", "Spanner instance ID (or JENNAH_INSTANCE env var)")
	rootCmd.PersistentFlags().String("database", "", "Spanner database name (or JENNAH_DATABASE env var)")
	rootCmd.PersistentFlags().MarkHidden("project")
	rootCmd.PersistentFlags().MarkHidden("instance")
	rootCmd.PersistentFlags().MarkHidden("database")

	// Disable completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(submitCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(tenantCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
}
