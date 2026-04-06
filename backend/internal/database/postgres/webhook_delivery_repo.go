package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"hearth/internal/models"
)

// WebhookDeliveryRepository handles webhook delivery log operations
type WebhookDeliveryRepository struct {
	db *sqlx.DB
}

// NewWebhookDeliveryRepository creates a new webhook delivery repository
func NewWebhookDeliveryRepository(db *sqlx.DB) *WebhookDeliveryRepository {
	return &WebhookDeliveryRepository{db: db}
}

// Create logs a new webhook delivery attempt
func (r *WebhookDeliveryRepository) Create(ctx context.Context, delivery *models.WebhookDelivery) error {
	query := `
		INSERT INTO webhook_deliveries (
			id, webhook_id, status_code, response_body, error_message, 
			attempt_number, delivered_at, created_at, request_payload, 
			response_headers, duration_ms
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`
	
	var requestPayload, responseHeaders interface{}
	if delivery.RequestPayload != nil {
		data, _ := json.Marshal(delivery.RequestPayload)
		requestPayload = data
	}
	if delivery.ResponseHeaders != nil {
		data, _ := json.Marshal(delivery.ResponseHeaders)
		responseHeaders = data
	}
	
	_, err := r.db.ExecContext(ctx, query,
		delivery.ID,
		delivery.WebhookID,
		delivery.StatusCode,
		delivery.ResponseBody,
		delivery.ErrorMessage,
		delivery.AttemptNumber,
		delivery.DeliveredAt,
		delivery.CreatedAt,
		requestPayload,
		responseHeaders,
		delivery.DurationMs,
	)
	return err
}

// GetByWebhookID retrieves deliveries for a webhook with pagination
func (r *WebhookDeliveryRepository) GetByWebhookID(ctx context.Context, webhookID uuid.UUID, limit, offset int) ([]*models.WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, status_code, response_body, error_message, 
			   attempt_number, delivered_at, created_at, request_payload, 
			   response_headers, duration_ms
		FROM webhook_deliveries
		WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	var deliveries []*models.WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, query, webhookID, limit, offset)
	if err != nil {
		return nil, err
	}
	return deliveries, nil
}

// GetRecentFailures retrieves recent failed deliveries for a webhook
func (r *WebhookDeliveryRepository) GetRecentFailures(ctx context.Context, webhookID uuid.UUID, limit int) ([]*models.WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, status_code, response_body, error_message, 
			   attempt_number, delivered_at, created_at, request_payload, 
			   response_headers, duration_ms
		FROM webhook_deliveries
		WHERE webhook_id = $1 
		  AND (status_code < 200 OR status_code >= 300)
		ORDER BY created_at DESC
		LIMIT $2
	`
	var deliveries []*models.WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, query, webhookID, limit)
	if err != nil {
		return nil, err
	}
	return deliveries, nil
}

// GetStats retrieves delivery statistics for a webhook
func (r *WebhookDeliveryRepository) GetStats(ctx context.Context, webhookID uuid.UUID) (*models.WebhookDeliveryStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_deliveries,
			COUNT(*) FILTER (WHERE status_code >= 200 AND status_code < 300) as successful_count,
			COUNT(*) FILTER (WHERE status_code < 200 OR status_code >= 300) as failed_count,
			AVG(duration_ms) FILTER (WHERE duration_ms IS NOT NULL) as avg_duration_ms,
			MAX(created_at) as last_delivery_at,
			MAX(created_at) FILTER (WHERE status_code < 200 OR status_code >= 300) as last_failure_at
		FROM webhook_deliveries
		WHERE webhook_id = $1
	`
	
	var stats models.WebhookDeliveryStats
	var lastDeliveryAt, lastFailureAt sql.NullTime
	var avgDuration sql.NullFloat64
	
	err := r.db.QueryRowContext(ctx, query, webhookID).Scan(
		&stats.TotalDeliveries,
		&stats.SuccessfulCount,
		&stats.FailedCount,
		&avgDuration,
		&lastDeliveryAt,
		&lastFailureAt,
	)
	if err != nil {
		return nil, err
	}
	
	if avgDuration.Valid {
		stats.AvgDurationMs = avgDuration.Float64
	}
	if lastDeliveryAt.Valid {
		stats.LastDeliveryAt = &lastDeliveryAt.Time
	}
	if lastFailureAt.Valid {
		stats.LastFailureAt = &lastFailureAt.Time
	}
	
	if stats.TotalDeliveries > 0 {
		stats.SuccessRate = float64(stats.SuccessfulCount) / float64(stats.TotalDeliveries) * 100
	}
	
	return &stats, nil
}

// GetRecentFailuresDetailed gets recent failures with full details
func (r *WebhookDeliveryRepository) GetRecentFailuresDetailed(ctx context.Context, webhookID uuid.UUID, limit int) ([]*models.WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, status_code, response_body, error_message, 
			   attempt_number, delivered_at, created_at, request_payload, 
			   response_headers, duration_ms
		FROM webhook_deliveries
		WHERE webhook_id = $1 
		  AND (status_code < 200 OR status_code >= 300)
		ORDER BY created_at DESC
		LIMIT $2
	`
	var deliveries []*models.WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, query, webhookID, limit)
	return deliveries, err
}

// CleanupOldDeliveries removes delivery logs older than the specified retention period
func (r *WebhookDeliveryRepository) CleanupOldDeliveries(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM webhook_deliveries WHERE created_at < $1`
	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetLastAttemptNumber gets the last attempt number for a webhook
func (r *WebhookDeliveryRepository) GetLastAttemptNumber(ctx context.Context, webhookID uuid.UUID) (int, error) {
	query := `
		SELECT COALESCE(MAX(attempt_number), 0)
		FROM webhook_deliveries
		WHERE webhook_id = $1
		  AND created_at > NOW() - INTERVAL '1 hour'
	`
	var maxAttempt int
	err := r.db.GetContext(ctx, &maxAttempt, query, webhookID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	return maxAttempt, nil
}
