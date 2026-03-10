package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alphauslabs/jennah/internal/database"
)

const (
	// How often the SSE loop polls Spanner for new notifications.
	ssePollInterval = 3 * time.Second
	// SSE keepalive comment interval to prevent proxies from closing idle connections.
	sseKeepaliveInterval = 15 * time.Second
)

// sseNotification is the JSON payload sent over the SSE stream.
type sseNotification struct {
	ID              string `json:"id"`
	JobID           string `json:"job_id"`
	JobName         string `json:"job_name,omitempty"`
	FinalStatus     string `json:"final_status"`
	ServiceTier     string `json:"service_tier,omitempty"`
	AssignedService string `json:"assigned_service,omitempty"`
	OccurredAt      int64  `json:"occurred_at"`
	ErrorMessage    string `json:"error_message,omitempty"`
	IsRead          bool   `json:"is_read"`
}

func dbNotifToSSE(n *database.Notification) sseNotification {
	s := sseNotification{
		ID:          n.NotificationId,
		JobID:       n.JobId,
		FinalStatus: n.FinalStatus,
		OccurredAt:  n.OccurredAt.Unix(),
		IsRead:      n.IsRead,
	}
	if n.JobName != nil {
		s.JobName = *n.JobName
	}
	if n.ServiceTier != nil {
		s.ServiceTier = *n.ServiceTier
	}
	if n.AssignedService != nil {
		s.AssignedService = *n.AssignedService
	}
	if n.ErrorMessage != nil {
		s.ErrorMessage = *n.ErrorMessage
	}
	return s
}

// SSENotificationsHandler returns an http.Handler that streams real-time
// notifications to the authenticated frontend client via Server-Sent Events.
//
// The handler resolves the tenant from OAuth headers, then polls Spanner for
// new notifications and writes them as SSE "notification" events. A keepalive
// comment is sent periodically to prevent connection timeouts.
//
// The stream ends when the client disconnects or the server shuts down.
func (s *GatewayService) SSENotificationsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		// Authenticate and resolve tenant.
		tenantID, err := s.resolveTenantFromHTTP(r)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		// Set SSE headers.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
		flusher.Flush()

		log.Printf("SSE stream opened for tenant %s", tenantID)
		defer log.Printf("SSE stream closed for tenant %s", tenantID)

		ctx := r.Context()

		// Start polling from now; on reconnect the frontend should refetch
		// the full list via ListNotifications and then open a new stream.
		cursor := time.Now().UTC()

		pollTicker := time.NewTicker(ssePollInterval)
		defer pollTicker.Stop()

		keepaliveTicker := time.NewTicker(sseKeepaliveInterval)
		defer keepaliveTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-keepaliveTicker.C:
				// SSE comment line keeps the connection alive.
				if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
					return
				}
				flusher.Flush()

			case <-pollTicker.C:
				notifications, err := s.dbClient.ListNotificationsSince(ctx, tenantID, cursor, 50)
				if err != nil {
					log.Printf("SSE poll error for tenant %s: %v", tenantID, err)
					continue
				}
				if len(notifications) == 0 {
					continue
				}

				for _, n := range notifications {
					payload := dbNotifToSSE(n)
					data, err := json.Marshal(payload)
					if err != nil {
						log.Printf("SSE marshal error: %v", err)
						continue
					}

					// Write SSE event with "notification" event type and id for
					// reconnect deduplication by the frontend.
					if _, err := fmt.Fprintf(w, "id: %s\nevent: notification\ndata: %s\n\n", payload.ID, data); err != nil {
						return
					}
				}
				flusher.Flush()

				// Advance cursor to the newest notification's CreatedAt.
				cursor = notifications[len(notifications)-1].CreatedAt
			}
		}
	})
}

// resolveTenantFromHTTP extracts OAuth headers from an *http.Request (not a
// ConnectRPC request) and resolves or creates the tenant. This mirrors
// resolveTenant but works with raw HTTP handlers.
func (s *GatewayService) resolveTenantFromHTTP(r *http.Request) (string, error) {
	oauthUser, err := extractOAuthUser(r.Header)
	if err != nil {
		return "", err
	}
	return s.getOrCreateTenant(oauthUser)
}
