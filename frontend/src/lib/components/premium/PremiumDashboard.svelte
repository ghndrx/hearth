<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { PremiumStatus, BillingInvoice, PaymentMethod } from '$lib/types/premium';
  import { TIER_COMPARISONS } from '$lib/types/premium';
  import ServerBoostManager from './ServerBoostManager.svelte';
  import FeatureComparison from './FeatureComparison.svelte';
  import PaymentMethodManager from './PaymentMethodManager.svelte';
  
  export let premiumStatus: PremiumStatus;
  export let invoices: BillingInvoice[] = [];
  export let paymentMethods: PaymentMethod[] = [];
  
  const dispatch = createEventDispatcher();
  
  $: isActive = premiumStatus.status === 'active';
  $: isCanceled = premiumStatus.status === 'canceled';
  $: isFree = premiumStatus.tier === 'free';
  
  function formatDate(dateStr?: string): string {
    if (!dateStr) return 'N/A';
    return new Date(dateStr).toLocaleDateString();
  }
  
  function formatCurrency(amount: number, currency: string): string {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency.toUpperCase()
    }).format(amount / 100);
  }
  
  function handleSubscribe(tier: 'basic' | 'premium') {
    dispatch('subscribe', tier);
  }
  
  function handleCancel() {
    dispatch('cancel');
  }
  
  function handleReactivate() {
    dispatch('reactivate');
  }
</script>

<div class="premium-dashboard">
  <!-- Current Status -->
  <section class="status-section">
    <h2>Your Subscription</h2>
    
    {#if isFree}
      <div class="status-card free">
        <div class="tier-badge">Free</div>
        <p>Upgrade to unlock premium features and support Hearth development.</p>
      </div>
    {:else if isCanceled}
      <div class="status-card canceled">
        <div class="tier-badge {premiumStatus.tier}">{premiumStatus.tier}</div>
        <p>Your subscription has been canceled. It will remain active until {formatDate(premiumStatus.expires_at)}.</p>
        <button class="btn-primary" on:click={handleReactivate}>Reactivate Subscription</button>
      </div>
    {:else}
      <div class="status-card {premiumStatus.tier}">
        <div class="tier-badge {premiumStatus.tier}">{premiumStatus.tier}</div>
        <div class="status-details">
          <div class="detail">
            <span class="label">Status</span>
            <span class="value">{premiumStatus.status}</span>
          </div>
          <div class="detail">
            <span class="label">Boosts Used</span>
            <span class="value">{premiumStatus.boosts_used} / {premiumStatus.boosts_total}</span>
          </div>
          {#if premiumStatus.subscription?.next_billing}
            <div class="detail">
              <span class="label">Next Billing</span>
              <span class="value">{formatDate(premiumStatus.subscription?.next_billing)}</span>
            </div>
          {/if}
        </div>
        <button class="btn-danger" on:click={handleCancel}>Cancel Subscription</button>
      </div>
    {/if}
  </section>
  
  <!-- Server Boosts -->
  {#if !isFree}
    <section class="boosts-section">
      <h2>Server Boosts</h2>
      <ServerBoostManager 
        boostsUsed={premiumStatus.boosts_used}
        boostsTotal={premiumStatus.boosts_total}
      />
    </section>
  {/if}
  
  <!-- Feature Comparison -->
  <section class="features-section">
    <h2>Compare Plans</h2>
    <FeatureComparison 
      currentTier={premiumStatus.tier}
      on:subscribe={(e) => handleSubscribe(e.detail)}
    />
  </section>
  
  <!-- Billing History -->
  {#if invoices.length > 0}
    <section class="billing-section">
      <h2>Billing History</h2>
      <div class="invoice-list">
        {#each invoices as invoice}
          <div class="invoice-item">
            <div class="invoice-info">
              <span class="invoice-desc">{invoice.description || 'Premium Subscription'}</span>
              <span class="invoice-date">{formatDate(invoice.created_at)}</span>
            </div>
            <div class="invoice-amount">
              {formatCurrency(invoice.amount, invoice.currency)}
            </div>
          </div>
        {/each}
      </div>
    </section>
  {/if}
  
  <!-- Payment Methods -->
  {#if !isFree && paymentMethods.length > 0}
    <section class="payment-section">
      <h2>Payment Methods</h2>
      <PaymentMethodManager methods={paymentMethods} />
    </section>
  {/if}
</div>

<style>
  .premium-dashboard {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }
  
  section {
    background: var(--bg-secondary);
    border-radius: 8px;
    padding: 20px;
  }
  
  h2 {
    font-size: 18px;
    font-weight: 600;
    margin-bottom: 16px;
  }
  
  .status-card {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  
  .tier-badge {
    display: inline-flex;
    padding: 4px 12px;
    border-radius: 4px;
    font-weight: 600;
    font-size: 14px;
    text-transform: capitalize;
    width: fit-content;
  }
  
  .tier-badge.free {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
  }
  
  .tier-badge.basic {
    background: var(--primary);
    color: white;
  }
  
  .tier-badge.premium {
    background: linear-gradient(135deg, #FFD700, #FFA500);
    color: black;
  }
  
  .status-details {
    display: flex;
    gap: 24px;
    flex-wrap: wrap;
  }
  
  .detail {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .detail .label {
    font-size: 12px;
    color: var(--text-secondary);
  }
  
  .detail .value {
    font-weight: 500;
  }
  
  .btn-primary {
    background: var(--primary);
    color: white;
    border: none;
    padding: 10px 20px;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 500;
  }
  
  .btn-primary:hover {
    opacity: 0.9;
  }
  
  .btn-danger {
    background: transparent;
    color: var(--danger);
    border: 1px solid var(--danger);
    padding: 10px 20px;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 500;
    width: fit-content;
  }
  
  .btn-danger:hover {
    background: var(--danger);
    color: white;
  }
  
  .invoice-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  
  .invoice-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px;
    background: var(--bg-tertiary);
    border-radius: 4px;
  }
  
  .invoice-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .invoice-desc {
    font-weight: 500;
  }
  
  .invoice-date {
    font-size: 12px;
    color: var(--text-secondary);
  }
  
  .invoice-amount {
    font-weight: 600;
  }
</style>
