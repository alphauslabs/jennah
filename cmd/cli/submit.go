package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	Use:   "submit <job.json>",
	Short: "Submit a job",
	Long:  "jennah submit <job.json> [--wait]\n\nReads job parameters from a JSON file and submits the job.\nUse --wait to stream status changes until the job completes.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wait, _ := cmd.Flags().GetBool("wait")

		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", args[0], err)
		}

		var body map[string]interface{}
		if err := json.Unmarshal(data, &body); err != nil {
			return fmt.Errorf("invalid JSON in %s: %w", args[0], err)
		}

		gw, err := newGatewayClient(cmd)
		if err != nil {
			return err
		}

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
		// Helper to read both camelCase and snake_case keys
		getField := func(camel, snake string) interface{} {
			if v, ok := body[camel]; ok && v != nil && v != "" {
				return v
			}
			return body[snake]
		}
		resourceProfile := getField("resourceProfile", "resource_profile")

		// Normalise keys to camelCase for the gateway
		if _, hasCamel := body["imageUri"]; !hasCamel {
			if v, ok := body["image_uri"]; ok {
				body["imageUri"] = v
				delete(body, "image_uri")
			}
		}
		if _, hasCamel := body["resourceProfile"]; !hasCamel {
			if v, ok := body["resource_profile"]; ok {
				body["resourceProfile"] = v
				delete(body, "resource_profile")
			}
		}
		if _, hasCamel := body["envVars"]; !hasCamel {
			if v, ok := body["env_vars"]; ok {
				body["envVars"] = v
				delete(body, "env_vars")
			}
		}

		// Print header info
		fmt.Printf("Gateway URL:      %s\n", gw.baseURL)
		fmt.Printf("User ID:          %s\n", gw.userID)
		fmt.Printf("Tenant ID:        %s\n", gw.tenantID)
		if resourceProfile != nil && resourceProfile != "" {
			fmt.Printf("Resource Profile: %v\n", resourceProfile)
		}
		fmt.Println()

		// Print full request payload as formatted JSON
		payloadJSON, _ := json.MarshalIndent(body, "", "  ")
		fmt.Println("Request Payload:")
		fmt.Println(string(payloadJSON))
		fmt.Println()
		fmt.Println("Submitting job...")

		statusCode, rawResp, err := gw.postRaw("/jennah.v1.DeploymentService/SubmitJob", body)
		if err != nil {
			return fmt.Errorf("submit failed: %w", err)
		}
		fmt.Printf("HTTP Status: %d\n", statusCode)
		if statusCode != 200 {
			var errResp struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if json.Unmarshal(rawResp, &errResp) == nil && errResp.Message != "" {
				return fmt.Errorf("%s: %s", errResp.Code, errResp.Message)
			}
			return fmt.Errorf("gateway error %d: %s", statusCode, string(rawResp))
		}

		// Pretty-print response
		var prettyResp interface{}
		json.Unmarshal(rawResp, &prettyResp)
		respJSON, _ := json.MarshalIndent(prettyResp, "", "  ")
		fmt.Println()
		fmt.Println("Response:")
		fmt.Println(string(respJSON))
		fmt.Println()

		var result struct {
			JobID          string `json:"jobId"`
			Status         string `json:"status"`
			WorkerAssigned string `json:"workerAssigned"`
		}
		json.Unmarshal(rawResp, &result)

		fmt.Println("✅ Job submitted successfully!")
		fmt.Printf("Job ID: %s\n", result.JobID)

		if !wait {
			fmt.Println()
			fmt.Println("Done!")
			return nil
		}

		fmt.Printf("✓ Job submitted successfully\n")
		fmt.Printf("  Job ID:  %s\n", jobID)
		fmt.Printf("  Status:  %s\n", status)
		fmt.Printf("  Tenant:  %s\n", tenantID)
		return nil
		fmt.Println()
		fmt.Println("Streaming status...")
		fmt.Println("============================================")

		// Handle Ctrl+C gracefully
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		lastStatus := result.Status
		fmt.Printf("  [%s]  %s\n", time.Now().Format("15:04:05"), lastStatus)

		terminalStates := map[string]bool{
			"SUCCEEDED": true,
			"FAILED":    true,
			"CANCELLED": true,
			"DELETED":   true,
		}

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				fmt.Println()
				return nil
			case <-ticker.C:
				jobs, err := fetchJobs(gw)
				if err != nil {
					fmt.Printf("  [%s]  polling error: %v\n", time.Now().Format("15:04:05"), err)
					continue
				}
				job := findJob(jobs, result.JobID)
				if job == nil {
					// Job no longer in list — it has completed
					fmt.Println("============================================")
					fmt.Println("Done!")
					return nil
				}
				if job.Status != lastStatus {
					fmt.Printf("  [%s]  %s → %s\n", time.Now().Format("15:04:05"), lastStatus, job.Status)
					lastStatus = job.Status
				}
				if terminalStates[lastStatus] {
					fmt.Println("============================================")
					fmt.Println("Done!")
					return nil
				}
			}
		}
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
	submitCmd.Flags().Bool("wait", false, "Stream status changes until the job completes")
}

