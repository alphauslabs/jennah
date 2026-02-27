package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	jennahv1 "github.com/alphauslabs/jennah/gen/proto"
	"github.com/alphauslabs/jennah/gen/proto/jennahv1connect"
	"github.com/alphauslabs/jennah/internal/database"
	"github.com/spf13/cobra"
)

// jobFile is the schema for a job.json submission file.
type jobFile struct {
	ImageUri        string            `json:"imageUri"`
	EnvVars         map[string]string `json:"envVars"`
	ResourceProfile string            `json:"resourceProfile"`
	MachineType     string            `json:"machineType"`
	Commands        []string          `json:"commands"`
}

var submitCmd = &cobra.Command{
	Use:   "submit [job.json]",
	Short: "Submit a new job",
	Long: `Submit a job to the gateway via a JSON file:
  jennah submit job.json [--gateway <url>] [--email <email>] [--user-id <id>]

Or write directly to Spanner (dev/debug):
  jennah submit --tenant-id <id> [--status PENDING]`,
	RunE: func(cmd *cobra.Command, args []string) error {

		// ── File-based gateway submission ──────────────────────────────────
		if len(args) == 1 {
			return submitViaGateway(cmd, args[0])
		}

		// ── Legacy direct-Spanner submission ──────────────────────────────
		tenantID, _ := cmd.Flags().GetString("tenant-id")
		status, _ := cmd.Flags().GetString("status")

		if tenantID == "" {
			return fmt.Errorf("provide a job.json file  OR  --tenant-id for direct submission")
		}
		if !validStatuses[status] {
			return fmt.Errorf("invalid status %q: must be PENDING, SCHEDULED, RUNNING, COMPLETED, FAILED, or CANCELLED", status)
		}

		db, closeDB, err := newDBClient(cmd)
		if err != nil {
			return err
		}
		defer closeDB()

		jobID := newJobID()
		if err := db.InsertJobFull(context.Background(), &database.Job{
			TenantId:   tenantID,
			JobId:      jobID,
			Status:     status,
			ImageUri:   "",
			RetryCount: 0,
			MaxRetries: 3,
		}); err != nil {
			if strings.Contains(err.Error(), "Parent row") || strings.Contains(err.Error(), "missing") {
				return fmt.Errorf("tenant %q not found — create it first with: jennah tenant create --email <email>", tenantID)
			}
			return fmt.Errorf("failed to submit job: %w", err)
		}

		fmt.Printf("✓ Job submitted successfully\n")
		fmt.Printf("  Job ID:  %s\n", jobID)
		fmt.Printf("  Status:  %s\n", status)
		fmt.Printf("  Tenant:  %s\n", tenantID)
		return nil
	},
}

func submitViaGateway(cmd *cobra.Command, filePath string) error {
	// ── Read job.json ──────────────────────────────────────────────────────
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filePath, err)
	}

	var jf jobFile
	if err := json.Unmarshal(data, &jf); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", filePath, err)
	}

	// ── Gateway URL ────────────────────────────────────────────────────────
	gatewayURL, _ := cmd.Flags().GetString("gateway")
	if gatewayURL == "" {
		gatewayURL = os.Getenv("JENNAH_GATEWAY_URL")
	}
	if gatewayURL == "" {
		return fmt.Errorf("gateway URL required: use --gateway or set JENNAH_GATEWAY_URL")
	}
	gatewayURL = strings.TrimRight(gatewayURL, "/")

	// ── OAuth headers ──────────────────────────────────────────────────────
	email, _ := cmd.Flags().GetString("email")
	userID, _ := cmd.Flags().GetString("user-id")
	provider, _ := cmd.Flags().GetString("provider")

	if email == "" {
		email = os.Getenv("JENNAH_EMAIL")
	}
	if userID == "" {
		userID = os.Getenv("JENNAH_USER_ID")
	}
	if email == "" || userID == "" {
		return fmt.Errorf("OAuth identity required: use --email and --user-id (or JENNAH_EMAIL / JENNAH_USER_ID)")
	}

	// ── Pretty-print request payload ───────────────────────────────────────
	resourceProfile := jf.ResourceProfile
	if resourceProfile == "" {
		resourceProfile = "default"
	}
	fmt.Printf("Gateway URL: %s\n", gatewayURL)
	fmt.Printf("Resource Profile: %s\n", resourceProfile)
	fmt.Println()
	fmt.Println("Request Payload:")
	pretty, _ := json.MarshalIndent(jf, "", "  ")
	fmt.Println(string(pretty))
	fmt.Println()

	// ── Build ConnectRPC client ────────────────────────────────────────────
	httpClient := &http.Client{}
	client := jennahv1connect.NewDeploymentServiceClient(httpClient, gatewayURL)

	req := connect.NewRequest(&jennahv1.SubmitJobRequest{
		ImageUri:        jf.ImageUri,
		EnvVars:         jf.EnvVars,
		ResourceProfile: resourceProfile,
		MachineType:     jf.MachineType,
		Commands:        jf.Commands,
	})
	req.Header().Set("X-OAuth-Email", email)
	req.Header().Set("X-OAuth-UserId", userID)
	req.Header().Set("X-OAuth-Provider", provider)

	// ── Submit ─────────────────────────────────────────────────────────────
	fmt.Println("Submitting job...")
	resp, err := client.SubmitJob(context.Background(), req)
	if err != nil {
		var connectErr *connect.Error
		if ok := strings.Contains(err.Error(), "connect"); ok {
			_ = connectErr
		}
		return fmt.Errorf("gateway error: %w", err)
	}

	// ── Print response ─────────────────────────────────────────────────────
	fmt.Printf("HTTP Status: 200\n\n")
	fmt.Println("Response:")
	respJSON, _ := json.MarshalIndent(map[string]string{
		"jobId":          resp.Msg.JobId,
		"status":         resp.Msg.Status,
		"workerAssigned": resp.Msg.WorkerAssigned,
	}, "", "  ")
	fmt.Println(string(respJSON))
	fmt.Println()
	fmt.Printf("✅ Job submitted successfully!\n")
	fmt.Printf("Job ID: %s\n", resp.Msg.JobId)
	fmt.Println()
	fmt.Println("Done!")
	return nil
}

func init() {
	submitCmd.Flags().String("status", "PENDING", "Job status (direct Spanner mode)")
	submitCmd.Flags().String("email", "", "OAuth email (or JENNAH_EMAIL env var)")
	submitCmd.Flags().String("user-id", "", "OAuth user ID (or JENNAH_USER_ID env var)")
	submitCmd.Flags().String("provider", "google", "OAuth provider (default: google)")
}

