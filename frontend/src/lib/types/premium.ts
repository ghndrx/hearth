// Premium subscription tiers
export type PremiumTier = 'free' | 'basic' | 'premium';
export type SubStatus = 'active' | 'canceled' | 'expired' | 'past_due' | 'trialing';

// Payment method types
export type PaymentMethodType = 'card' | 'paypal' | 'bank_transfer';

export interface PaymentMethod {
  id: string;
  type: PaymentMethodType;
  last4: string;
  brand?: string;
  expires_at?: string;
  is_default: boolean;
}

// Subscription model
export interface Subscription {
  id: string;
  user_id: string;
  tier: PremiumTier;
  status: SubStatus;
  boosts_used: number;
  boosts_total: number;
  next_billing?: string;
  canceled_at?: string;
  created_at: string;
  updated_at: string;
}

// Server boost model
export interface ServerBoost {
  id: string;
  server_id: string;
  user_id: string;
  active: boolean;
  created_at: string;
  expires_at?: string;
}

// Server perks based on boost level
export interface ServerPerks {
  level: number;
  boost_count: number;
  boosts_required: number;
  emoji_limit: number;
  file_upload_limit: number;
  voice_bitrate: number;
  has_vanity_url: boolean;
  has_animated_icon: boolean;
  has_banner: boolean;
  has_splash_screen: boolean;
}

// Premium features for each tier
export interface PremiumFeatures {
  tier: PremiumTier;
  monthly_price: number;
  server_boosts: number;
  file_upload_size: number;
  cross_server_emojis: boolean;
  high_quality_video: boolean;
  custom_discriminator: boolean;
  early_access: boolean;
  premium_badge: boolean;
  priority_support: boolean;
  no_ads: boolean;
}

// User's premium status
export interface PremiumStatus {
  user_id: string;
  tier: PremiumTier;
  status: SubStatus;
  boosts_used: number;
  boosts_total: number;
  boosts_available: number;
  features: PremiumFeatures;
  subscription?: Subscription;
  expires_at?: string;
}

// Billing invoice
export interface BillingInvoice {
  id: string;
  user_id: string;
  external_id: string;
  amount: number;
  currency: string;
  status: string;
  description?: string;
  paid_at?: string;
  created_at: string;
}

// Tier comparison data
export interface TierComparison {
  tier: PremiumTier;
  name: string;
  price: number;
  features: string[];
  recommended?: boolean;
}

export const TIER_COMPARISONS: TierComparison[] = [
  {
    tier: 'free',
    name: 'Free',
    price: 0,
    features: [
      '8MB file uploads',
      '50 custom emojis',
      'Standard voice quality (96kbps)',
      'Basic server features',
    ]
  },
  {
    tier: 'basic',
    name: 'Basic',
    price: 4.99,
    features: [
      '50MB file uploads',
      '100 custom emojis',
      'Enhanced voice quality (128kbps)',
      '2 server boosts',
      'Vanity URL for servers',
      'Animated server icons',
      'Priority support',
    ]
  },
  {
    tier: 'premium',
    name: 'Premium',
    price: 9.99,
    features: [
      '500MB file uploads',
      '250 custom emojis',
      'High-quality voice (384kbps)',
      '2 server boosts',
      'Cross-server emojis',
      '1080p60 video streaming',
      'Custom discriminator (#1234)',
      'Early access to new features',
      'Premium badge on profile',
      'No ads',
      'Priority support',
    ],
    recommended: true
  }
];
