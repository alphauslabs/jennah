package database

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ListNotificationsSince returns notifications for a tenant that were created
// after the given timestamp, ordered oldest-first so the caller can stream
// them chronologically. limit ≤ 0 defaults to 50.
func (c *Client) ListNotificationsSince(ctx context.Context, tenantID string, since time.Time, limit int32) ([]*Notification, error) {
	if limit <= 0 {
		limit = 50
	}

	stmt := spanner.Statement{
		SQL: `SELECT ` + columnList(notificationColumns) + `
		      FROM Notifications
		      WHERE TenantId = @tenantId AND CreatedAt > @since
		      ORDER BY CreatedAt ASC
		      LIMIT @limit`,
		Params: map[string]interface{}{
			"tenantId": tenantID,
			"since":    since,
			"limit":    int64(limit),
		},
	}

	iter := c.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var notifications []*Notification
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list notifications since: %w", err)
		}
		var n Notification
		if err := row.ToStruct(&n); err != nil {
			return nil, fmt.Errorf("parse notification row: %w", err)
		}
		notifications = append(notifications, &n)
	}
	return notifications, nil
}
