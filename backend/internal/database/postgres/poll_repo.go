package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// PollRepository handles poll data access
type PollRepository struct {
	db *sqlx.DB
}

// NewPollRepository creates a new poll repository
func NewPollRepository(db *sqlx.DB) *PollRepository {
	return &PollRepository{db: db}
}

// Create inserts a new poll with its options
func (r *PollRepository) Create(ctx context.Context, poll *models.Poll) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Insert poll
	pollQuery := `
		INSERT INTO polls (id, channel_id, creator_id, question, is_multiple, end_time, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.ExecContext(ctx, pollQuery,
		poll.ID, poll.ChannelID, poll.CreatorID, poll.Question,
		poll.IsMultiple, poll.EndTime, poll.CreatedAt, poll.UpdatedAt)
	if err != nil {
		return err
	}

	// Insert options
	optQuery := `
		INSERT INTO poll_options (id, poll_id, text, votes, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, opt := range poll.Options {
		_, err = tx.ExecContext(ctx, optQuery, opt.ID, poll.ID, opt.Text, opt.Votes, time.Now())
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByID retrieves a poll with its options
func (r *PollRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Poll, error) {
	var poll models.Poll
	pollQuery := `
		SELECT id, channel_id, creator_id, question, is_multiple, end_time, created_at, updated_at
		FROM polls WHERE id = $1
	`
	err := r.db.GetContext(ctx, &poll, pollQuery, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Get options
	var options []models.PollOption
	optQuery := `SELECT id, poll_id, text, votes, created_at FROM poll_options WHERE poll_id = $1`
	err = r.db.SelectContext(ctx, &options, optQuery, id)
	if err != nil {
		return nil, err
	}
	poll.Options = options

	return &poll, nil
}

// Update updates an existing poll
func (r *PollRepository) Update(ctx context.Context, poll *models.Poll) error {
	query := `
		UPDATE polls
		SET question = $2, is_multiple = $3, end_time = $4, updated_at = $5
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, poll.ID, poll.Question, poll.IsMultiple, poll.EndTime, poll.UpdatedAt)
	return err
}

// Delete removes a poll and its options/votes (cascades)
func (r *PollRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM polls WHERE id = $1`, id)
	return err
}

// GetByChannelID retrieves all polls for a channel
func (r *PollRepository) GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Poll, error) {
	var polls []*models.Poll
	query := `
		SELECT id, channel_id, creator_id, question, is_multiple, end_time, created_at, updated_at
		FROM polls WHERE channel_id = $1 ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &polls, query, channelID)
	if err != nil {
		return nil, err
	}

	// Load options for each poll
	for _, poll := range polls {
		var options []models.PollOption
		err = r.db.SelectContext(ctx, &options, `SELECT id, poll_id, text, votes, created_at FROM poll_options WHERE poll_id = $1`, poll.ID)
		if err != nil {
			return nil, err
		}
		poll.Options = options
	}

	return polls, nil
}

// GetByGuildID retrieves all polls for a server (via channel guild mapping)
func (r *PollRepository) GetByGuildID(ctx context.Context, guildID uuid.UUID) ([]*models.Poll, error) {
	var polls []*models.Poll
	query := `
		SELECT p.id, p.channel_id, p.creator_id, p.question, p.is_multiple, p.end_time, p.created_at, p.updated_at
		FROM polls p
		INNER JOIN channels c ON p.channel_id = c.id
		WHERE c.server_id = $1
		ORDER BY p.created_at DESC
	`
	err := r.db.SelectContext(ctx, &polls, query, guildID)
	if err != nil {
		return nil, err
	}

	// Load options for each poll
	for _, poll := range polls {
		var options []models.PollOption
		err = r.db.SelectContext(ctx, &options, `SELECT id, poll_id, text, votes, created_at FROM poll_options WHERE poll_id = $1`, poll.ID)
		if err != nil {
			return nil, err
		}
		poll.Options = options
	}

	return polls, nil
}

// VoteForOption casts a vote for a user on a poll option.
// It is called AFTER the service has verified the user hasn't already voted.
// The vote is recorded in a transaction.
func (r *PollRepository) VoteForOption(ctx context.Context, pollID, optionID, userID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Insert the new vote - trigger handles vote count increment
	voteQuery := `
		INSERT INTO poll_votes (id, poll_id, option_id, user_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.ExecContext(ctx, voteQuery, uuid.New(), pollID, optionID, userID, time.Now())
	if err != nil {
		return err
	}

	return tx.Commit()
}

// CheckUserVote checks if a user has already voted on a poll
func (r *PollRepository) CheckUserVote(ctx context.Context, pollID, userID uuid.UUID) (*models.PollOptionVote, error) {
	var vote models.PollOptionVote
	query := `
		SELECT id, poll_id, option_id, user_id, created_at
		FROM poll_votes
		WHERE poll_id = $1 AND user_id = $2
	`
	err := r.db.GetContext(ctx, &vote, query, pollID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &vote, nil
}
