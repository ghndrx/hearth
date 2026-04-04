package models

import "testing"

func TestSubscriptionTierFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected PremiumTier
	}{
		{"basic", TierBasic},
		{"premium", TierPremium},
		{"free", TierFree},
		{"unknown", TierFree},
		{"", TierFree},
		{"BASIC", TierFree}, // case-sensitive
		{"Premium", TierFree}, // case-sensitive
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := SubscriptionTierFromString(tc.input)
			if result != tc.expected {
				t.Errorf("SubscriptionTierFromString(%q) = %v; want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestGetPremiumFeatures(t *testing.T) {
	// Test TierFree
	freeFeatures := GetPremiumFeatures(TierFree)
	if freeFeatures.Tier != TierFree {
		t.Errorf("TierFree features tier = %v; want %v", freeFeatures.Tier, TierFree)
	}
	if freeFeatures.MonthlyPrice != 0 {
		t.Errorf("TierFree MonthlyPrice = %v; want 0", freeFeatures.MonthlyPrice)
	}
	if freeFeatures.ServerBoosts != 0 {
		t.Errorf("TierFree ServerBoosts = %v; want 0", freeFeatures.ServerBoosts)
	}
	if freeFeatures.FileUploadSize != 8*1024*1024 {
		t.Errorf("TierFree FileUploadSize = %v; want 8MB", freeFeatures.FileUploadSize)
	}
	if freeFeatures.CrossServerEmojis {
		t.Error("TierFree CrossServerEmojis should be false")
	}

	// Test TierBasic
	basicFeatures := GetPremiumFeatures(TierBasic)
	if basicFeatures.Tier != TierBasic {
		t.Errorf("TierBasic features tier = %v; want %v", basicFeatures.Tier, TierBasic)
	}
	if basicFeatures.MonthlyPrice != 2.99 {
		t.Errorf("TierBasic MonthlyPrice = %v; want 2.99", basicFeatures.MonthlyPrice)
	}
	if basicFeatures.ServerBoosts != 2 {
		t.Errorf("TierBasic ServerBoosts = %v; want 2", basicFeatures.ServerBoosts)
	}
	if basicFeatures.FileUploadSize != 50*1024*1024 {
		t.Errorf("TierBasic FileUploadSize = %v; want 50MB", basicFeatures.FileUploadSize)
	}
	if !basicFeatures.CrossServerEmojis {
		t.Error("TierBasic CrossServerEmojis should be true")
	}
	if !basicFeatures.HighQualityVideo {
		t.Error("TierBasic HighQualityVideo should be true")
	}
	if !basicFeatures.CustomDiscriminator {
		t.Error("TierBasic CustomDiscriminator should be true")
	}
	if !basicFeatures.PrioritySupport {
		t.Error("TierBasic PrioritySupport should be true")
	}
	if !basicFeatures.EarlyAccess {
		t.Error("TierBasic EarlyAccess should be true")
	}
	// TierBasic should NOT have premium-only features
	if basicFeatures.PremiumBadge {
		t.Error("TierBasic PremiumBadge should be false")
	}
	if basicFeatures.NoAds {
		t.Error("TierBasic NoAds should be false")
	}

	// Test TierPremium
	premiumFeatures := GetPremiumFeatures(TierPremium)
	if premiumFeatures.Tier != TierPremium {
		t.Errorf("TierPremium features tier = %v; want %v", premiumFeatures.Tier, TierPremium)
	}
	if premiumFeatures.MonthlyPrice != 9.99 {
		t.Errorf("TierPremium MonthlyPrice = %v; want 9.99", premiumFeatures.MonthlyPrice)
	}
	if premiumFeatures.ServerBoosts != 2 {
		t.Errorf("TierPremium ServerBoosts = %v; want 2", premiumFeatures.ServerBoosts)
	}
	if premiumFeatures.FileUploadSize != 100*1024*1024 {
		t.Errorf("TierPremium FileUploadSize = %v; want 100MB", premiumFeatures.FileUploadSize)
	}
	// Premium-only features
	if !premiumFeatures.PremiumBadge {
		t.Error("TierPremium PremiumBadge should be true")
	}
	if !premiumFeatures.NoAds {
		t.Error("TierPremium NoAds should be true")
	}
	if !premiumFeatures.MessageEditHistory {
		t.Error("TierPremium MessageEditHistory should be true")
	}
	if !premiumFeatures.PremiumStickers {
		t.Error("TierPremium PremiumStickers should be true")
	}
	if !premiumFeatures.CustomStatusEmoji {
		t.Error("TierPremium CustomStatusEmoji should be true")
	}
	if !premiumFeatures.HDScreenShare {
		t.Error("TierPremium HDScreenShare should be true")
	}
}

func TestCalculateServerLevel(t *testing.T) {
	tests := []struct {
		boostCount int
		expected   int
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{14, 1},
		{15, 2},
		{29, 2},
		{30, 3},
		{100, 3},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			result := CalculateServerLevel(tc.boostCount)
			if result != tc.expected {
				t.Errorf("CalculateServerLevel(%d) = %d; want %d", tc.boostCount, result, tc.expected)
			}
		})
	}
}

func TestGetServerPerks(t *testing.T) {
	// Note: ServerBoostPerks defines perks PER LEVEL, not cumulative.
	// Each level only lists perks that are NEW at that level.
	// So level 2 has EmojiLimit=150 and HasBanner=true, but doesn't set HasVanityURL
	// (it stays at the zero value from the struct).
	tests := []struct {
		boostCount       int
		expectedLevel    int
		expectedEmoji    int
		expectedVanity   bool
		expectedAnimated bool
		expectedBanner   bool
		expectedSplash   bool
	}{
		{0, 0, 50, false, false, false, false},
		{1, 0, 50, false, false, false, false},
		{2, 1, 100, true, true, false, false},
		{15, 2, 150, false, false, true, false},
		{30, 3, 250, false, false, false, true},
	}

	for _, tc := range tests {
		perks := GetServerPerks(tc.boostCount)
		if perks.Level != tc.expectedLevel {
			t.Errorf("GetServerPerks(%d).Level = %d; want %d", tc.boostCount, perks.Level, tc.expectedLevel)
		}
		if perks.BoostCount != tc.boostCount {
			t.Errorf("GetServerPerks(%d).BoostCount = %d; want %d", tc.boostCount, perks.BoostCount, tc.boostCount)
		}
		if perks.EmojiLimit != tc.expectedEmoji {
			t.Errorf("GetServerPerks(%d).EmojiLimit = %d; want %d", tc.boostCount, perks.EmojiLimit, tc.expectedEmoji)
		}
		if perks.HasVanityURL != tc.expectedVanity {
			t.Errorf("GetServerPerks(%d).HasVanityURL = %v; want %v", tc.boostCount, perks.HasVanityURL, tc.expectedVanity)
		}
		if perks.HasAnimatedIcon != tc.expectedAnimated {
			t.Errorf("GetServerPerks(%d).HasAnimatedIcon = %v; want %v", tc.boostCount, perks.HasAnimatedIcon, tc.expectedAnimated)
		}
		if perks.HasBanner != tc.expectedBanner {
			t.Errorf("GetServerPerks(%d).HasBanner = %v; want %v", tc.boostCount, perks.HasBanner, tc.expectedBanner)
		}
		if perks.HasSplashScreen != tc.expectedSplash {
			t.Errorf("GetServerPerks(%d).HasSplashScreen = %v; want %v", tc.boostCount, perks.HasSplashScreen, tc.expectedSplash)
		}
	}
}

func TestGetServerPerksBoostsRequiredForNextLevel(t *testing.T) {
	// Level 0 should require 2 for next level
	perks0 := GetServerPerks(0)
	if perks0.BoostsRequired != 2 {
		t.Errorf("GetServerPerks(0).BoostsRequired = %d; want 2", perks0.BoostsRequired)
	}

	// Level 1 should require 15 for next level
	perks1 := GetServerPerks(2)
	if perks1.BoostsRequired != 15 {
		t.Errorf("GetServerPerks(2).BoostsRequired = %d; want 15", perks1.BoostsRequired)
	}

	// Level 2 should require 30 for next level
	perks2 := GetServerPerks(15)
	if perks2.BoostsRequired != 30 {
		t.Errorf("GetServerPerks(15).BoostsRequired = %d; want 30", perks2.BoostsRequired)
	}

	// Level 3 (max) should require 0
	perks3 := GetServerPerks(30)
	if perks3.BoostsRequired != 0 {
		t.Errorf("GetServerPerks(30).BoostsRequired = %d; want 0", perks3.BoostsRequired)
	}
}

func TestGetSubscriptionPlans(t *testing.T) {
	plans := GetSubscriptionPlans()
	if len(plans) == 0 {
		t.Fatal("GetSubscriptionPlans() returned empty slice")
	}

	// Verify plans are sorted by price
	for i := 1; i < len(plans); i++ {
		if plans[i].Price < plans[i-1].Price {
			t.Error("Plans are not sorted by price ascending")
		}
	}

	// Verify all plans have required fields
	for i, plan := range plans {
		if plan.ID == "" {
			t.Errorf("Plan[%d] has empty ID", i)
		}
		if plan.Name == "" {
			t.Errorf("Plan[%d] has empty Name", i)
		}
		if plan.Price <= 0 {
			t.Errorf("Plan[%d] has invalid Price %v", i, plan.Price)
		}
		if plan.PriceCents <= 0 {
			t.Errorf("Plan[%d] has invalid PriceCents %v", i, plan.PriceCents)
		}
		if plan.Currency == "" {
			t.Errorf("Plan[%d] has empty Currency", i)
		}
		if plan.Tier == "" {
			t.Errorf("Plan[%d] has empty Tier", i)
		}
		if plan.PriceCents != int(plan.Price*100) {
			t.Errorf("Plan[%d] PriceCents %d doesn't match Price %v * 100 = %d", i, plan.PriceCents, plan.Price, int(plan.Price*100))
		}
	}
}
