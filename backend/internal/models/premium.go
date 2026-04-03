package models

import (
	"time"

	"github.com/google/uuid"
)

// PremiumTier represents the subscription tier level
type PremiumTier string

const (
	TierFree    PremiumTier = "free"
	TierBasic   PremiumTier = "basic"   // $2.99/month - 2 server boosts
	TierPremium PremiumTier = "premium" // $9.99/month - 2 boosts + premium features
)

// SubStatus represents the subscription status
type SubStatus string

const (
	SubStatusActive   SubStatus = "active"
	SubStatusCanceled SubStatus = "canceled"
	SubStatusExpired  SubStatus = "expired"
	SubStatusPastDue  SubStatus = "past_due"
	SubStatusTrialing SubStatus = "trialing"
)

// PaymentMethodType represents the type of payment method
type PaymentMethodType string

const (
	PaymentMethodCard   PaymentMethodType = "card"
	PaymentMethodPayPal PaymentMethodType = "paypal"
	PaymentMethodBank   PaymentMethodType = "bank_transfer"
)

// PaymentMethod represents a payment method on file
type PaymentMethod struct {
	ID        string            `json:"id" db:"id"`
	Type      PaymentMethodType `json:"type" db:"type"`
	Last4     string            `json:"last4" db:"last4"`
	Brand     string            `json:"brand,omitempty" db:"brand"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty" db:"expires_at"`
	IsDefault bool              `json:"is_default" db:"is_default"`
}

// Subscription represents a user's premium subscription
type Subscription struct {
	ID                   uuid.UUID      `json:"id" db:"id"`
	UserID               uuid.UUID      `json:"user_id" db:"user_id"`
	Tier                 PremiumTier    `json:"tier" db:"tier"`
	Status               SubStatus      `json:"status" db:"status"`
	BoostsUsed           int            `json:"boosts_used" db:"boosts_used"`
	BoostsTotal          int            `json:"boosts_total" db:"boosts_total"`
	NextBilling          *time.Time     `json:"next_billing,omitempty" db:"next_billing"`
	CurrentPeriodStart   *time.Time     `json:"current_period_start,omitempty" db:"current_period_start"`
	CurrentPeriodEnd     *time.Time     `json:"current_period_end,omitempty" db:"current_period_end"`
	CanceledAt           *time.Time     `json:"canceled_at,omitempty" db:"canceled_at"`
	PaymentMethod        *PaymentMethod `json:"payment_method,omitempty"`
	StripeSubscriptionID string         `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	ExternalID           string         `json:"external_id,omitempty" db:"external_id"` // Stripe customer ID
	CreatedAt            time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at" db:"updated_at"`
}

// ServerBoost represents a user's boost applied to a server
type ServerBoost struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	ServerID  uuid.UUID  `json:"server_id" db:"server_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Active    bool       `json:"active" db:"active"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

// ServerPerks represents the perks available at a given boost level
type ServerPerks struct {
	Level           int   `json:"level"`             // 0, 1, 2, 3
	BoostCount      int   `json:"boost_count"`       // Current boost count
	BoostsRequired  int   `json:"boosts_required"`   // For next level
	EmojiLimit      int   `json:"emoji_limit"`       // 50, 100, 150, 250
	FileUploadLimit int64 `json:"file_upload_limit"` // 8MB, 50MB, 100MB, 500MB
	VoiceBitrate    int   `json:"voice_bitrate"`     // 96k, 128k, 256k, 384k
	HasVanityURL    bool  `json:"has_vanity_url"`
	HasAnimatedIcon bool  `json:"has_animated_icon"`
	HasBanner       bool  `json:"has_banner"`
	HasSplashScreen bool  `json:"has_splash_screen"`
}

// PremiumFeatures represents features available for a premium tier
type PremiumFeatures struct {
	// Tier info
	Tier         PremiumTier `json:"tier"`
	MonthlyPrice float64     `json:"monthly_price"`

	// Basic Tier ($2.99/month) and Premium ($9.99/month)
	ServerBoosts        int   `json:"server_boosts"`        // 2 boosts
	FileUploadSize      int64 `json:"file_upload_size"`     // 50MB basic, 100MB premium vs 8MB free
	CrossServerEmojis   bool  `json:"cross_server_emojis"`  // Use emojis across servers
	HighQualityVideo    bool  `json:"high_quality_video"`   // 1080p60 vs 720p30
	CustomDiscriminator bool  `json:"custom_discriminator"` // Choose your #1234
	EarlyAccess         bool  `json:"early_access"`         // Beta features
	PrioritySupport     bool  `json:"priority_support"`

	// Premium Tier ($9.99/month) only
	PremiumBadge       bool `json:"premium_badge"` // Profile badge
	NoAds              bool `json:"no_ads"`
	MessageEditHistory bool `json:"message_edit_history"` // Full edit history access
	PremiumStickers    bool `json:"premium_stickers"`     // Premium sticker packs
	CustomStatusEmoji  bool `json:"custom_status_emoji"`  // Emoji in custom status
	HDScreenShare      bool `json:"hd_screen_share"`      // HD screen sharing
}

// PremiumStatus represents a user's overall premium status
type PremiumStatus struct {
	UserID          uuid.UUID       `json:"user_id"`
	Tier            PremiumTier     `json:"tier"`
	Status          SubStatus       `json:"status"`
	BoostsUsed      int             `json:"boosts_used"`
	BoostsTotal     int             `json:"boosts_total"`
	BoostsAvailable int             `json:"boosts_available"`
	Features        PremiumFeatures `json:"features"`
	Subscription    *Subscription   `json:"subscription,omitempty"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
}

// BillingInvoice represents a billing invoice
type BillingInvoice struct {
	ID          string     `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	ExternalID  string     `json:"external_id,omitempty" db:"external_id"`
	Amount      int        `json:"amount" db:"amount"` // cents
	Currency    string     `json:"currency" db:"currency"`
	Status      string     `json:"status" db:"status"`
	Description string     `json:"description" db:"description"`
	PaidAt      *time.Time `json:"paid_at,omitempty" db:"paid_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// Customer represents a billing customer record
type Customer struct {
	ID         string    `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Email      string    `json:"email" db:"email"`
	ExternalID string    `json:"external_id,omitempty" db:"external_id"` // Stripe customer ID
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// ServerBoostPerks is a map of boost levels to their perks
var ServerBoostPerks = map[int]ServerPerks{
	0: { // No boosts
		EmojiLimit:      50,
		FileUploadLimit: 8 * 1024 * 1024, // 8MB
		VoiceBitrate:    96000,
		HasVanityURL:    false,
	},
	1: { // 2 boosts
		EmojiLimit:      100,
		FileUploadLimit: 50 * 1024 * 1024, // 50MB
		VoiceBitrate:    128000,
		HasVanityURL:    true,
		HasAnimatedIcon: true,
	},
	2: { // 15 boosts
		EmojiLimit:      150,
		FileUploadLimit: 100 * 1024 * 1024, // 100MB
		VoiceBitrate:    256000,
		HasBanner:       true,
	},
	3: { // 30 boosts
		EmojiLimit:      250,
		FileUploadLimit: 500 * 1024 * 1024, // 500MB
		VoiceBitrate:    384000,
		HasSplashScreen: true,
	},
}

// PremiumTierPricing defines pricing for each tier
var PremiumTierPricing = map[PremiumTier]float64{
	TierFree:    0,
	TierBasic:   2.99,
	TierPremium: 9.99,
}

// TierBoostsTotal defines how many boosts each tier provides
var TierBoostsTotal = map[PremiumTier]int{
	TierFree:    0,
	TierBasic:   2,
	TierPremium: 2,
}

// LevelBoostsRequired defines how many boosts are needed for each level
var LevelBoostsRequired = map[int]int{
	0: 0,
	1: 2,
	2: 15,
	3: 30,
}

// GetPremiumFeatures returns the features for a given tier
func GetPremiumFeatures(tier PremiumTier) PremiumFeatures {
	features := PremiumFeatures{
		Tier:         tier,
		MonthlyPrice: PremiumTierPricing[tier],
	}

	switch tier {
	case TierBasic:
		features.ServerBoosts = 2
		features.FileUploadSize = 50 * 1024 * 1024 // 50MB
		features.CrossServerEmojis = true
		features.HighQualityVideo = true
		features.CustomDiscriminator = true
		features.PrioritySupport = true
		features.EarlyAccess = true
	case TierPremium:
		features.ServerBoosts = 2
		features.FileUploadSize = 100 * 1024 * 1024 // 100MB
		features.CrossServerEmojis = true
		features.HighQualityVideo = true
		features.CustomDiscriminator = true
		features.EarlyAccess = true
		features.PremiumBadge = true
		features.PrioritySupport = true
		features.NoAds = true
		features.MessageEditHistory = true
		features.PremiumStickers = true
		features.CustomStatusEmoji = true
		features.HDScreenShare = true
	default: // Free
		features.ServerBoosts = 0
		features.FileUploadSize = 8 * 1024 * 1024 // 8MB
	}

	return features
}

// CalculateServerLevel returns the boost level for a given boost count
func CalculateServerLevel(boostCount int) int {
	if boostCount >= LevelBoostsRequired[3] {
		return 3
	}
	if boostCount >= LevelBoostsRequired[2] {
		return 2
	}
	if boostCount >= LevelBoostsRequired[1] {
		return 1
	}
	return 0
}

// GetServerPerks returns the perks for a server at a given boost level
func GetServerPerks(boostCount int) ServerPerks {
	level := CalculateServerLevel(boostCount)
	perks := ServerBoostPerks[level]
	perks.BoostCount = boostCount
	perks.Level = level

	// Calculate boosts required for next level
	if level < 3 {
		perks.BoostsRequired = LevelBoostsRequired[level+1]
	} else {
		perks.BoostsRequired = 0
	}

	return perks
}

// SubscriptionTierFromString converts a string to PremiumTier
func SubscriptionTierFromString(s string) PremiumTier {
	switch s {
	case "basic":
		return TierBasic
	case "premium":
		return TierPremium
	default:
		return TierFree
	}
}

// SubscriptionPlan represents a purchasable subscription plan
type SubscriptionPlan struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Price         float64     `json:"price"`       // dollars per month
	PriceCents    int         `json:"price_cents"` // cents per month
	Currency      string      `json:"currency"`
	Tier          PremiumTier `json:"tier"`
	Features      []string    `json:"features"`
	IsBestValue   bool        `json:"is_best_value"`
	StripePriceID string      `json:"stripe_price_id,omitempty"`
	Active        bool        `json:"active"`
	SortOrder     int         `json:"sort_order"`
}

// GetSubscriptionPlans returns all available subscription plans
func GetSubscriptionPlans() []*SubscriptionPlan {
	return []*SubscriptionPlan{
		{
			ID:          "plan_basic",
			Name:        "Hearth+ Basic",
			Description: "Essential premium features for an enhanced experience",
			Price:       2.99,
			PriceCents:  299,
			Currency:    "USD",
			Tier:        TierBasic,
			Features: []string{
				"Custom emoji across servers",
				"50MB file uploads",
				"HD video streaming",
				"Custom tag (#0001-#9999)",
				"Priority customer support",
				"Early access to new features",
			},
			IsBestValue:   false,
			StripePriceID: "price_basic_monthly",
			Active:        true,
			SortOrder:     1,
		},
		{
			ID:          "plan_premium",
			Name:        "Hearth+ Premium",
			Description: "The complete premium experience with server boosts",
			Price:       9.99,
			PriceCents:  999,
			Currency:    "USD",
			Tier:        TierPremium,
			Features: []string{
				"Everything in Basic",
				"100MB file uploads",
				"2 server boost slots",
				"Premium sticker packs",
				"Custom status with emoji",
				"HD screen sharing",
				"256kbps voice quality",
				"Message editing history",
				"No advertisements",
			},
			IsBestValue:   true,
			StripePriceID: "price_premium_monthly",
			Active:        true,
			SortOrder:     2,
		},
	}
}
