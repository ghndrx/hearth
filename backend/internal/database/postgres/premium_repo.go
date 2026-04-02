package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

type PremiumRepository struct {
	db *sqlx.DB
}

func NewPremiumRepository(db *sqlx.DB) *PremiumRepository {
	return &PremiumRepository{db: db}
}

// Subscription operations

func (r *PremiumRepository) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	query := `
		INSERT INTO subscriptions (id, user_id, tier, status, boosts_used, boosts_total,
			next_billing, current_period_start, current_period_end, canceled_at,
			stripe_subscription_id, external_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.UserID, sub.Tier, sub.Status, sub.BoostsUsed, sub.BoostsTotal,
		sub.NextBilling, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.CanceledAt,
		sub.StripeSubscriptionID, sub.ExternalID, sub.CreatedAt, sub.UpdatedAt,
	)
	return err
}

func (r *PremiumRepository) GetSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	var sub models.Subscription
	query := `
		SELECT id, user_id, tier, status, boosts_used, boosts_total, next_billing,
			current_period_start, current_period_end, canceled_at,
			stripe_subscription_id, external_id, created_at, updated_at
		FROM subscriptions WHERE user_id = $1
	`
	err := r.db.GetContext(ctx, &sub, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &sub, err
}

func (r *PremiumRepository) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*models.Subscription, error) {
	var sub models.Subscription
	query := `
		SELECT id, user_id, tier, status, boosts_used, boosts_total, next_billing,
			current_period_start, current_period_end, canceled_at,
			stripe_subscription_id, external_id, created_at, updated_at
		FROM subscriptions WHERE stripe_subscription_id = $1
	`
	err := r.db.GetContext(ctx, &sub, query, stripeSubID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &sub, err
}

func (r *PremiumRepository) UpdateSubscription(ctx context.Context, sub *models.Subscription) error {
	query := `
		UPDATE subscriptions SET
			tier = $2, status = $3, boosts_used = $4, boosts_total = $5,
			next_billing = $6, current_period_start = $7, current_period_end = $8,
			canceled_at = $9, stripe_subscription_id = $10, updated_at = $11
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.Tier, sub.Status, sub.BoostsUsed, sub.BoostsTotal,
		sub.NextBilling, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CanceledAt, sub.StripeSubscriptionID, time.Now(),
	)
	return err
}

func (r *PremiumRepository) DeleteSubscription(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE user_id = $1`, userID)
	return err
}

// Server boost operations

func (r *PremiumRepository) CreateServerBoost(ctx context.Context, boost *models.ServerBoost) error {
	query := `
		INSERT INTO server_boosts (id, server_id, user_id, active, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (server_id, user_id) DO UPDATE SET
			active = $4, expires_at = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		boost.ID, boost.ServerID, boost.UserID, boost.Active, boost.CreatedAt, boost.ExpiresAt,
	)
	return err
}

func (r *PremiumRepository) GetServerBoost(ctx context.Context, serverID, userID uuid.UUID) (*models.ServerBoost, error) {
	var boost models.ServerBoost
	query := `
		SELECT id, server_id, user_id, active, created_at, expires_at
		FROM server_boosts WHERE server_id = $1 AND user_id = $2
	`
	err := r.db.GetContext(ctx, &boost, query, serverID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &boost, err
}

func (r *PremiumRepository) GetServerBoosts(ctx context.Context, serverID uuid.UUID) ([]*models.ServerBoost, error) {
	query := `
		SELECT id, server_id, user_id, active, created_at, expires_at
		FROM server_boosts WHERE server_id = $1 AND active = true
		ORDER BY created_at DESC
	`
	var boosts []*models.ServerBoost
	err := r.db.SelectContext(ctx, &boosts, query, serverID)
	return boosts, err
}

func (r *PremiumRepository) GetServerBoostCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM server_boosts WHERE server_id = $1 AND active = true`
	err := r.db.GetContext(ctx, &count, query, serverID)
	return count, err
}

func (r *PremiumRepository) DeactivateServerBoost(ctx context.Context, serverID, userID uuid.UUID) error {
	query := `UPDATE server_boosts SET active = false WHERE server_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, serverID, userID)
	return err
}

func (r *PremiumRepository) GetUserBoosts(ctx context.Context, userID uuid.UUID) ([]*models.ServerBoost, error) {
	query := `
		SELECT id, server_id, user_id, active, created_at, expires_at
		FROM server_boosts WHERE user_id = $1 AND active = true
		ORDER BY created_at DESC
	`
	var boosts []*models.ServerBoost
	err := r.db.SelectContext(ctx, &boosts, query, userID)
	return boosts, err
}

func (r *PremiumRepository) GetUserActiveBoostCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM server_boosts WHERE user_id = $1 AND active = true`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

// Server boost level operations

func (r *PremiumRepository) UpdateServerBoostLevel(ctx context.Context, serverID uuid.UUID, boostCount int) error {
	query := `
		INSERT INTO server_boost_levels (server_id, boost_count, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (server_id) DO UPDATE SET boost_count = $2, updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, serverID, boostCount)
	return err
}

func (r *PremiumRepository) GetServerBoostLevel(ctx context.Context, serverID uuid.UUID) (*models.ServerPerks, error) {
	var level struct {
		Level      int `db:"level"`
		BoostCount int `db:"boost_count"`
	}
	query := `SELECT level, boost_count FROM server_boost_levels WHERE server_id = $1`
	err := r.db.GetContext(ctx, &level, query, serverID)
	if err == sql.ErrNoRows {
		// Return level 0 if no record exists
		perks := models.GetServerPerks(0)
		return &perks, nil
	}
	if err != nil {
		return nil, err
	}

	perks := models.GetServerPerks(level.BoostCount)
	return &perks, nil
}

// Billing customer operations

func (r *PremiumRepository) CreateBillingCustomer(ctx context.Context, customer *models.Customer) error {
	query := `
		INSERT INTO billing_customers (id, user_id, email, external_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET email = $3, external_id = $4
	`
	_, err := r.db.ExecContext(ctx, query,
		customer.ID, customer.UserID, customer.Email, customer.ExternalID, customer.CreatedAt,
	)
	return err
}

func (r *PremiumRepository) GetBillingCustomer(ctx context.Context, userID uuid.UUID) (*models.Customer, error) {
	var customer models.Customer
	query := `SELECT id, user_id, email, external_id, created_at FROM billing_customers WHERE user_id = $1`
	err := r.db.GetContext(ctx, &customer, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &customer, err
}

func (r *PremiumRepository) GetBillingCustomerByExternalID(ctx context.Context, externalID string) (*models.Customer, error) {
	var customer models.Customer
	query := `SELECT id, user_id, email, external_id, created_at FROM billing_customers WHERE external_id = $1`
	err := r.db.GetContext(ctx, &customer, query, externalID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &customer, err
}

// Billing invoice operations

func (r *PremiumRepository) CreateBillingInvoice(ctx context.Context, invoice *models.BillingInvoice) error {
	query := `
		INSERT INTO billing_invoices (id, user_id, external_id, amount, currency, status, description, paid_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, external_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query,
		invoice.ID, invoice.UserID, invoice.ExternalID, invoice.Amount, invoice.Currency,
		invoice.Status, invoice.Description, invoice.PaidAt, invoice.CreatedAt,
	)
	return err
}

func (r *PremiumRepository) GetBillingInvoices(ctx context.Context, userID uuid.UUID, limit int) ([]*models.BillingInvoice, error) {
	query := `
		SELECT id, user_id, external_id, amount, currency, status, description, paid_at, created_at
		FROM billing_invoices WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2
	`
	var invoices []*models.BillingInvoice
	err := r.db.SelectContext(ctx, &invoices, query, userID, limit)
	return invoices, err
}

// Payment method operations

func (r *PremiumRepository) CreatePaymentMethod(ctx context.Context, userID uuid.UUID, pm *models.PaymentMethod) error {
	query := `
		INSERT INTO payment_methods (id, user_id, external_id, type, last4, brand, expires_at, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, external_id) DO UPDATE SET
			type = $4, last4 = $5, brand = $6, expires_at = $7, is_default = $8
	`
	_, err := r.db.ExecContext(ctx, query,
		pm.ID, userID, pm.ID, pm.Type, pm.Last4, pm.Brand, pm.ExpiresAt, pm.IsDefault,
	)
	return err
}

func (r *PremiumRepository) GetPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*models.PaymentMethod, error) {
	query := `
		SELECT id, type, last4, brand, expires_at, is_default
		FROM payment_methods WHERE user_id = $1 ORDER BY is_default DESC, created_at DESC
	`
	var methods []*models.PaymentMethod
	err := r.db.SelectContext(ctx, &methods, query, userID)
	return methods, err
}

func (r *PremiumRepository) GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*models.PaymentMethod, error) {
	var pm models.PaymentMethod
	query := `
		SELECT id, type, last4, brand, expires_at, is_default
		FROM payment_methods WHERE user_id = $1 AND is_default = true
	`
	err := r.db.GetContext(ctx, &pm, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &pm, err
}

func (r *PremiumRepository) DeletePaymentMethod(ctx context.Context, userID uuid.UUID, paymentMethodID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM payment_methods WHERE user_id = $1 AND id = $2`, userID, paymentMethodID)
	return err
}

// Update user's premium tier
func (r *PremiumRepository) UpdateUserPremiumTier(ctx context.Context, userID uuid.UUID, tier models.PremiumTier) error {
	query := `UPDATE users SET premium_tier = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, tier)
	return err
}

func (r *PremiumRepository) GetUserPremiumTier(ctx context.Context, userID uuid.UUID) (models.PremiumTier, error) {
	var tier string
	query := `SELECT premium_tier FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &tier, query, userID)
	if err == sql.ErrNoRows {
		return models.TierFree, nil
	}
	return models.SubscriptionTierFromString(tier), err
}
