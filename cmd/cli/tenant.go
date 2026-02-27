package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var tenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "Manage your tenant account",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var tenantWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show your tenant info",
	Long:  "jennah tenant whoami",
	RunE: func(cmd *cobra.Command, args []string) error {
		gw, err := newGatewayClient(cmd)
		if err != nil {
			return err
		}

		var result struct {
			TenantID      string `json:"tenantId"`
			UserEmail     string `json:"userEmail"`
			OAuthProvider string `json:"oauthProvider"`
			CreatedAt     string `json:"createdAt"`
		}

		if len(tenants) == 0 {
			fmt.Println("No tenants found.")
			return nil
		}

		fmt.Printf("%-20s  %-30s  %-36s  %s\n", "NAME", "EMAIL", "TENANT ID", "CREATED")
		fmt.Println(strings.Repeat("\u2500", 96))
		for _, t := range tenants {
			fmt.Printf("%-20s  %-30s  %-36s  %s\n", t.OAuthUserId, t.UserEmail, t.TenantId, t.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		return nil
	},
}

var tenantDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a tenant and all its jobs",
	Long:  "jennah tenant delete --tenant-id <id>",
	RunE: func(cmd *cobra.Command, args []string) error {
		tenantID, _ := cmd.Flags().GetString("tenant-id")
		if tenantID == "" {
			return fmt.Errorf("--tenant-id flag is required")
		}

		db, closeDB, err := newDBClient(cmd)
		if err != nil {
			return err
		}
		defer closeDB()

		if err := db.DeleteTenant(context.Background(), tenantID); err != nil {
			return fmt.Errorf("failed to delete tenant: %w", err)
		if err := gw.post("/jennah.v1.DeploymentService/GetCurrentTenant", map[string]interface{}{}, &result); err != nil {
			return fmt.Errorf("failed to get tenant: %w", err)
		}

		fmt.Println("Tenant Info")
		fmt.Println(strings.Repeat("─", 40))
		fmt.Printf("Tenant ID: %s\n", result.TenantID)
		fmt.Printf("Email:     %s\n", result.UserEmail)
		fmt.Printf("Provider:  %s\n", result.OAuthProvider)
		fmt.Printf("Created:   %s\n", result.CreatedAt)
		return nil
	},
}

func init() {
	tenantCmd.AddCommand(tenantWhoamiCmd)
}
