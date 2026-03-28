package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// StorageRepository implements services.StorageRepository for PostgreSQL
type StorageRepository struct {
	db *sqlx.DB
}

// NewStorageRepository creates a new storage repository
func NewStorageRepository(db *sqlx.DB) *StorageRepository {
	return &StorageRepository{db: db}
}

// GetUserStorageUsage returns the total bytes used by a user across all servers
func (r *StorageRepository) GetUserStorageUsage(ctx context.Context, userID uuid.UUID) (int64, error) {
	var total sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT total_bytes FROM user_storage_totals WHERE user_id = $1`,
		userID,
	).Scan(&total)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

// GetUserStorageInfo returns detailed storage info for a user
func (r *StorageRepository) GetUserStorageInfo(ctx context.Context, userID uuid.UUID) (*UserStorageInfo, error) {
	query := `
		SELECT user_id, total_bytes, file_count, last_updated 
		FROM user_storage_totals 
		WHERE user_id = $1
	`
	var info UserStorageInfo
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&info.UserID,
		&info.TotalBytes,
		&info.FileCount,
		&info.LastUpdated,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &UserStorageInfo{
				UserID:      userID,
				TotalBytes:  0,
				FileCount:   0,
				LastUpdated: time.Now().Format(time.RFC3339),
			}, nil
		}
		return nil, err
	}
	return &info, nil
}

// GetServerStorageUsage returns the total bytes used by a server
func (r *StorageRepository) GetServerStorageUsage(ctx context.Context, serverID uuid.UUID) (int64, error) {
	var total sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(used_bytes), 0) FROM storage_usage WHERE server_id = $1`,
		serverID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

// UpdateUserStorage adds to a user's storage usage (after successful upload)
// serverID is optional (nil for DM/shared storage)
func (r *StorageRepository) UpdateUserStorage(ctx context.Context, userID uuid.UUID, serverID *uuid.UUID, bytesDelta int64, fileCountDelta int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update storage_usage for the specific server
	if serverID != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO storage_usage (user_id, server_id, used_bytes, file_count, last_updated)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (user_id, server_id) 
			DO UPDATE SET 
				used_bytes = storage_usage.used_bytes + $3,
				file_count = storage_usage.file_count + $4,
				last_updated = NOW()
		`, userID, *serverID, bytesDelta, fileCountDelta)
		if err != nil {
			return err
		}
	}

	// Update user_storage_totals (always)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_storage_totals (user_id, total_bytes, file_count, last_updated)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			total_bytes = user_storage_totals.total_bytes + $2,
			file_count = user_storage_totals.file_count + $3,
			last_updated = NOW()
	`, userID, bytesDelta, fileCountDelta)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DecrementUserStorage removes from a user's storage usage (after successful delete)
func (r *StorageRepository) DecrementUserStorage(ctx context.Context, userID uuid.UUID, serverID *uuid.UUID, bytesDelta int64, fileCountDelta int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update storage_usage for the specific server
	if serverID != nil {
		_, err = tx.ExecContext(ctx, `
			UPDATE storage_usage 
			SET used_bytes = GREATEST(0, used_bytes - $3),
			    file_count = GREATEST(0, file_count - $4),
			    last_updated = NOW()
			WHERE user_id = $1 AND server_id = $2
		`, userID, *serverID, bytesDelta, fileCountDelta)
		if err != nil {
			return err
		}
	}

	// Update user_storage_totals (always)
	_, err = tx.ExecContext(ctx, `
		UPDATE user_storage_totals 
		SET total_bytes = GREATEST(0, total_bytes - $2),
		    file_count = GREATEST(0, file_count - $3),
		    last_updated = NOW()
		WHERE user_id = $1
	`, userID, bytesDelta, fileCountDelta)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// UserStorageInfo represents a user's total storage usage
type UserStorageInfo struct {
	UserID      uuid.UUID `db:"user_id"`
	TotalBytes  int64     `db:"total_bytes"`
	FileCount   int       `db:"file_count"`
	LastUpdated string    `db:"last_updated"`
}
