package database

import (
	"time"

	"cloud.google.com/go/spanner"
)

// Tenant represents an organization/team using the platform
type Tenant struct {
	TenantId      string             `spanner:"TenantId"`
	UserEmail     spanner.NullString `spanner:"UserEmail"`
	OAuthProvider spanner.NullString `spanner:"OAuthProvider"`
	OAuthUserId   spanner.NullString `spanner:"OAuthUserId"`
	CreatedAt     time.Time          `spanner:"CreatedAt"`
	UpdatedAt     time.Time          `spanner:"UpdatedAt"`
}

// Job represents a deployment job
type Job struct {
	TenantId             string     `spanner:"TenantId"`
	JobId                string     `spanner:"JobId"`
	Status               string     `spanner:"Status"`
	ImageUri             string     `spanner:"ImageUri"`
	Commands             []string   `spanner:"Commands"`
	CreatedAt            time.Time  `spanner:"CreatedAt"`
	UpdatedAt            time.Time  `spanner:"UpdatedAt"`
	ScheduledAt          *time.Time `spanner:"ScheduledAt"`
	StartedAt            *time.Time `spanner:"StartedAt"`
	CompletedAt          *time.Time `spanner:"CompletedAt"`
	RetryCount           int64      `spanner:"RetryCount"`
	MaxRetries           int64      `spanner:"MaxRetries"`
	ErrorMessage         *string    `spanner:"ErrorMessage"`
	CloudJobResourcePath *string    `spanner:"CloudJobResourcePath"`
}

// JobStateTransition tracks state changes for audit trail
type JobStateTransition struct {
	TenantId       string    `spanner:"TenantId"`
	JobId          string    `spanner:"JobId"`
	TransitionId   string    `spanner:"TransitionId"`
	FromStatus     *string   `spanner:"FromStatus"`
	ToStatus       string    `spanner:"ToStatus"`
	TransitionedAt time.Time `spanner:"TransitionedAt"`
	Reason         *string   `spanner:"Reason"`
}

// JobStatus constants
const (
	JobStatusPending   = "PENDING"
	JobStatusScheduled = "SCHEDULED"
	JobStatusRunning   = "RUNNING"
	JobStatusCompleted = "COMPLETED"
	JobStatusFailed    = "FAILED"
	JobStatusCancelled = "CANCELLED"
)
