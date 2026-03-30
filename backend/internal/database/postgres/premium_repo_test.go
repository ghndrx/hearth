package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func setupPremiumRepoMock(t *testing.T) (*PremiumRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewPremiumRepository(sqlxDB)
	return repo, mock
}

// Test Subscription operations

func TestPremiumRepository_CreateSubscription(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Tier:        models.TierBasic,
		Status:      models.SubStatusActive,
		BoostsUsed:  0,
		BoostsTotal: 2,
		NextBilling: &now,
		ExternalID:  "sub_123",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mock.ExpectExec("INSERT INTO subscriptions").
		WithArgs(sub.ID, sub.UserID, sub.Tier, sub.Status, sub.BoostsUsed, sub.BoostsTotal,
			sub.NextBilling, sub.CanceledAt, sub.ExternalID, sub.CreatedAt, sub.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateSubscription(ctx, sub)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetSubscription(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	now := time.Now()
	expectedSub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		Tier:        models.TierBasic,
		Status:      models.SubStatusActive,
		BoostsUsed:  1,
		BoostsTotal: 2,
		NextBilling: &now,
		ExternalID:  "sub_123",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "tier", "status", "boosts_used", "boosts_total",
		"next_billing", "canceled_at", "external_id", "created_at", "updated_at",
	}).AddRow(
		expectedSub.ID, expectedSub.UserID, expectedSub.Tier, expectedSub.Status,
		expectedSub.BoostsUsed, expectedSub.BoostsTotal, expectedSub.NextBilling,
		expectedSub.CanceledAt, expectedSub.ExternalID, expectedSub.CreatedAt, expectedSub.UpdatedAt,
	)

	mock.ExpectQuery("SELECT .+ FROM subscriptions WHERE user_id").
		WithArgs(userID).
		WillReturnRows(rows)

	sub, err := repo.GetSubscription(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, expectedSub.ID, sub.ID)
	assert.Equal(t, expectedSub.Tier, sub.Tier)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetSubscription_NotFound(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "tier", "status", "boosts_used", "boosts_total",
		"next_billing", "canceled_at", "external_id", "created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT .+ FROM subscriptions WHERE user_id").
		WithArgs(userID).
		WillReturnRows(rows)

	sub, err := repo.GetSubscription(ctx, userID)
	require.NoError(t, err)
	assert.Nil(t, sub)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_UpdateSubscription(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	sub := &models.Subscription{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Tier:        models.TierPremium,
		Status:      models.SubStatusActive,
		BoostsUsed:  2,
		BoostsTotal: 2,
		NextBilling: &now,
		UpdatedAt:   now,
	}

	mock.ExpectExec("UPDATE subscriptions SET").
		WithArgs(sub.ID, sub.Tier, sub.Status, sub.BoostsUsed, sub.BoostsTotal,
			sub.NextBilling, sub.CanceledAt, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateSubscription(ctx, sub)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_DeleteSubscription(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM subscriptions WHERE user_id").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteSubscription(ctx, userID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test Server Boost operations

func TestPremiumRepository_CreateServerBoost(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	boost := &models.ServerBoost{
		ID:        uuid.New(),
		ServerID:  uuid.New(),
		UserID:    uuid.New(),
		Active:    true,
		CreatedAt: now,
		ExpiresAt: nil,
	}

	mock.ExpectExec("INSERT INTO server_boosts").
		WithArgs(boost.ID, boost.ServerID, boost.UserID, boost.Active, boost.CreatedAt, boost.ExpiresAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateServerBoost(ctx, boost)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetServerBoost(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	now := time.Now()
	expectedBoost := &models.ServerBoost{
		ID:        uuid.New(),
		ServerID:  serverID,
		UserID:    userID,
		Active:    true,
		CreatedAt: now,
		ExpiresAt: nil,
	}

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "user_id", "active", "created_at", "expires_at",
	}).AddRow(
		expectedBoost.ID, expectedBoost.ServerID, expectedBoost.UserID,
		expectedBoost.Active, expectedBoost.CreatedAt, expectedBoost.ExpiresAt,
	)

	mock.ExpectQuery("SELECT .+ FROM server_boosts WHERE server_id").
		WithArgs(serverID, userID).
		WillReturnRows(rows)

	boost, err := repo.GetServerBoost(ctx, serverID, userID)
	require.NoError(t, err)
	assert.Equal(t, expectedBoost.ID, boost.ID)
	assert.Equal(t, expectedBoost.Active, boost.Active)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetServerBoost_NotFound(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "user_id", "active", "created_at", "expires_at",
	})

	mock.ExpectQuery("SELECT .+ FROM server_boosts WHERE server_id").
		WithArgs(serverID, userID).
		WillReturnRows(rows)

	boost, err := repo.GetServerBoost(ctx, serverID, userID)
	require.NoError(t, err)
	assert.Nil(t, boost)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetServerBoosts(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	now := time.Now()
	boosts := []*models.ServerBoost{
		{ID: uuid.New(), ServerID: serverID, UserID: uuid.New(), Active: true, CreatedAt: now},
		{ID: uuid.New(), ServerID: serverID, UserID: uuid.New(), Active: true, CreatedAt: now},
	}

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "user_id", "active", "created_at", "expires_at",
	})
	for _, b := range boosts {
		rows.AddRow(b.ID, b.ServerID, b.UserID, b.Active, b.CreatedAt, b.ExpiresAt)
	}

	mock.ExpectQuery("SELECT .+ FROM server_boosts WHERE server_id").
		WithArgs(serverID).
		WillReturnRows(rows)

	result, err := repo.GetServerBoosts(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetServerBoostCount(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	rows := sqlmock.NewRows([]string{"count"}).AddRow(5)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM server_boosts WHERE server_id").
		WithArgs(serverID).
		WillReturnRows(rows)

	count, err := repo.GetServerBoostCount(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_DeactivateServerBoost(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("UPDATE server_boosts SET active = false").
		WithArgs(serverID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeactivateServerBoost(ctx, serverID, userID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetUserBoosts(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	now := time.Now()
	boosts := []*models.ServerBoost{
		{ID: uuid.New(), ServerID: uuid.New(), UserID: userID, Active: true, CreatedAt: now},
	}

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "user_id", "active", "created_at", "expires_at",
	}).AddRow(boosts[0].ID, boosts[0].ServerID, boosts[0].UserID, boosts[0].Active, boosts[0].CreatedAt, boosts[0].ExpiresAt)

	mock.ExpectQuery("SELECT .+ FROM server_boosts WHERE user_id").
		WithArgs(userID).
		WillReturnRows(rows)

	result, err := repo.GetUserBoosts(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetUserActiveBoostCount(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"count"}).AddRow(3)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM server_boosts WHERE user_id").
		WithArgs(userID).
		WillReturnRows(rows)

	count, err := repo.GetUserActiveBoostCount(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test Server Boost Level operations

func TestPremiumRepository_UpdateServerBoostLevel(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	boostCount := 15

	mock.ExpectExec("INSERT INTO server_boost_levels").
		WithArgs(serverID, boostCount).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateServerBoostLevel(ctx, serverID, boostCount)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetServerBoostLevel(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	rows := sqlmock.NewRows([]string{"level", "boost_count"}).AddRow(2, 15)

	mock.ExpectQuery("SELECT level, boost_count FROM server_boost_levels WHERE server_id").
		WithArgs(serverID).
		WillReturnRows(rows)

	perks, err := repo.GetServerBoostLevel(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, 2, perks.Level)
	assert.Equal(t, 15, perks.BoostCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetServerBoostLevel_NotFound(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	mock.ExpectQuery("SELECT level, boost_count FROM server_boost_levels WHERE server_id").
		WithArgs(serverID).
		WillReturnError(sql.ErrNoRows)

	perks, err := repo.GetServerBoostLevel(ctx, serverID)
	require.NoError(t, err)
	// Should return level 0 perks for no record
	assert.Equal(t, 0, perks.Level)
	assert.Equal(t, 50, perks.EmojiLimit)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test Billing Customer operations

func TestPremiumRepository_CreateBillingCustomer(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	customer := &models.Customer{
		ID:         uuid.New().String(),
		UserID:     uuid.New(),
		Email:      "test@example.com",
		ExternalID: "cus_123",
		CreatedAt:  now,
	}

	mock.ExpectExec("INSERT INTO billing_customers").
		WithArgs(customer.ID, customer.UserID, customer.Email, customer.ExternalID, customer.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateBillingCustomer(ctx, customer)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetBillingCustomer(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	expected := &models.Customer{
		ID:         uuid.New().String(),
		UserID:     userID,
		Email:      "test@example.com",
		ExternalID: "cus_123",
		CreatedAt:  time.Now(),
	}

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "email", "external_id", "created_at",
	}).AddRow(expected.ID, expected.UserID, expected.Email, expected.ExternalID, expected.CreatedAt)

	mock.ExpectQuery("SELECT .+ FROM billing_customers WHERE user_id").
		WithArgs(userID).
		WillReturnRows(rows)

	customer, err := repo.GetBillingCustomer(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, expected.Email, customer.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetBillingCustomerByExternalID(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	externalID := "cus_123"

	expected := &models.Customer{
		ID:         uuid.New().String(),
		UserID:     uuid.New(),
		Email:      "test@example.com",
		ExternalID: externalID,
		CreatedAt:  time.Now(),
	}

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "email", "external_id", "created_at",
	}).AddRow(expected.ID, expected.UserID, expected.Email, expected.ExternalID, expected.CreatedAt)

	mock.ExpectQuery("SELECT .+ FROM billing_customers WHERE external_id").
		WithArgs(externalID).
		WillReturnRows(rows)

	customer, err := repo.GetBillingCustomerByExternalID(ctx, externalID)
	require.NoError(t, err)
	assert.Equal(t, expected.ExternalID, customer.ExternalID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test Billing Invoice operations

func TestPremiumRepository_CreateBillingInvoice(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	invoice := &models.BillingInvoice{
		ID:          uuid.New().String(),
		UserID:      uuid.New(),
		ExternalID:  "inv_123",
		Amount:      499,
		Currency:    "USD",
		Status:      "paid",
		Description: "Basic Plan",
		PaidAt:      &now,
		CreatedAt:   now,
	}

	mock.ExpectExec("INSERT INTO billing_invoices").
		WithArgs(invoice.ID, invoice.UserID, invoice.ExternalID, invoice.Amount, invoice.Currency,
			invoice.Status, invoice.Description, invoice.PaidAt, invoice.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateBillingInvoice(ctx, invoice)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetBillingInvoices(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	now := time.Now()
	invoices := []*models.BillingInvoice{
		{
			ID: uuid.New().String(), UserID: userID, ExternalID: "inv_1",
			Amount: 499, Currency: "USD", Status: "paid", CreatedAt: now,
		},
	}

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "external_id", "amount", "currency", "status", "description", "paid_at", "created_at",
	})
	for _, inv := range invoices {
		rows.AddRow(inv.ID, inv.UserID, inv.ExternalID, inv.Amount, inv.Currency, inv.Status, inv.Description, inv.PaidAt, inv.CreatedAt)
	}

	mock.ExpectQuery("SELECT .+ FROM billing_invoices WHERE user_id").
		WithArgs(userID, 10).
		WillReturnRows(rows)

	result, err := repo.GetBillingInvoices(ctx, userID, 10)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test Payment Method operations

func TestPremiumRepository_CreatePaymentMethod(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	now := time.Now()
	pm := &models.PaymentMethod{
		ID:        uuid.New().String(),
		Type:      models.PaymentMethodCard,
		Last4:     "4242",
		Brand:     "visa",
		ExpiresAt: &now,
		IsDefault: true,
	}

	mock.ExpectExec("INSERT INTO payment_methods").
		WithArgs(pm.ID, userID, pm.ID, pm.Type, pm.Last4, pm.Brand, pm.ExpiresAt, pm.IsDefault).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreatePaymentMethod(ctx, userID, pm)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetPaymentMethods(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	now := time.Now()
	methods := []*models.PaymentMethod{
		{ID: uuid.New().String(), Type: models.PaymentMethodCard, Last4: "4242", Brand: "visa", ExpiresAt: &now, IsDefault: true},
	}

	rows := sqlmock.NewRows([]string{
		"id", "type", "last4", "brand", "expires_at", "is_default",
	})
	for _, m := range methods {
		rows.AddRow(m.ID, m.Type, m.Last4, m.Brand, m.ExpiresAt, m.IsDefault)
	}

	mock.ExpectQuery("SELECT .+ FROM payment_methods WHERE user_id").
		WithArgs(userID).
		WillReturnRows(rows)

	result, err := repo.GetPaymentMethods(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetDefaultPaymentMethod(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	now := time.Now()
	expected := &models.PaymentMethod{
		ID: uuid.New().String(), Type: models.PaymentMethodCard, Last4: "4242", Brand: "visa", ExpiresAt: &now, IsDefault: true,
	}

	rows := sqlmock.NewRows([]string{
		"id", "type", "last4", "brand", "expires_at", "is_default",
	}).AddRow(expected.ID, expected.Type, expected.Last4, expected.Brand, expected.ExpiresAt, expected.IsDefault)

	mock.ExpectQuery("SELECT .+ FROM payment_methods WHERE user_id").
		WithArgs(userID).
		WillReturnRows(rows)

	result, err := repo.GetDefaultPaymentMethod(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, expected.Last4, result.Last4)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetDefaultPaymentMethod_NotFound(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "type", "last4", "brand", "expires_at", "is_default",
	})

	mock.ExpectQuery("SELECT .+ FROM payment_methods WHERE user_id").
		WithArgs(userID).
		WillReturnRows(rows)

	result, err := repo.GetDefaultPaymentMethod(ctx, userID)
	require.NoError(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_DeletePaymentMethod(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()
	paymentMethodID := uuid.New().String()

	mock.ExpectExec("DELETE FROM payment_methods WHERE user_id").
		WithArgs(userID, paymentMethodID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeletePaymentMethod(ctx, userID, paymentMethodID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test User Premium Tier operations

func TestPremiumRepository_UpdateUserPremiumTier(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	mock.ExpectExec("UPDATE users SET premium_tier").
		WithArgs(userID, models.TierBasic).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateUserPremiumTier(ctx, userID, models.TierBasic)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetUserPremiumTier(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"premium_tier"}).AddRow("basic")

	mock.ExpectQuery("SELECT premium_tier FROM users WHERE id").
		WithArgs(userID).
		WillReturnRows(rows)

	tier, err := repo.GetUserPremiumTier(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, models.TierBasic, tier)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumRepository_GetUserPremiumTier_NotFound(t *testing.T) {
	repo, mock := setupPremiumRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	mock.ExpectQuery("SELECT premium_tier FROM users WHERE id").
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	tier, err := repo.GetUserPremiumTier(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, models.TierFree, tier)
	assert.NoError(t, mock.ExpectationsWereMet())
}
