<script lang="ts">
  import { onMount } from 'svelte';
  import PremiumDashboard from '$lib/components/premium/PremiumDashboard.svelte';
  import type { PremiumStatus, BillingInvoice, PaymentMethod } from '$lib/types/premium';
  
  let premiumStatus: PremiumStatus | null = null;
  let invoices: BillingInvoice[] = [];
  let paymentMethods: PaymentMethod[] = [];
  let loading = true;
  let error = '';
  
  onMount(async () => {
    try {
      const [statusRes, invoicesRes, methodsRes] = await Promise.all([
        fetch('/api/v1/premium/subscription'),
        fetch('/api/v1/premium/invoices'),
        fetch('/api/v1/premium/payment-methods')
      ]);
      
      if (statusRes.ok) {
        premiumStatus = await statusRes.json();
      }
      if (invoicesRes.ok) {
        invoices = await invoicesRes.json();
      }
      if (methodsRes.ok) {
        paymentMethods = await methodsRes.json();
      }
    } catch (e) {
      error = 'Failed to load premium status';
    } finally {
      loading = false;
    }
  });
  
  async function handleSubscribe(tier: 'basic' | 'premium') {
    try {
      const res = await fetch('/api/v1/premium/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tier })
      });
      
      if (res.ok) {
        premiumStatus = await res.json();
      } else {
        const data = await res.json();
        error = data.message || 'Failed to subscribe';
      }
    } catch (e) {
      error = 'Failed to subscribe';
    }
  }
  
  async function handleCancel() {
    if (!confirm('Are you sure you want to cancel your subscription?')) return;
    
    try {
      const res = await fetch('/api/v1/premium/subscription', { method: 'DELETE' });
      if (res.ok) {
        premiumStatus = await res.json();
      }
    } catch (e) {
      error = 'Failed to cancel subscription';
    }
  }
  
  async function handleReactivate() {
    try {
      const res = await fetch('/api/v1/premium/subscription/reactivate', { method: 'POST' });
      if (res.ok) {
        premiumStatus = await res.json();
      }
    } catch (e) {
      error = 'Failed to reactivate subscription';
    }
  }
</script>

<div class="premium-page">
  <h1>Premium</h1>
  
  {#if loading}
    <div class="loading">Loading...</div>
  {:else if error}
    <div class="error">{error}</div>
  {:else if premiumStatus}
    <PremiumDashboard
      {premiumStatus}
      {invoices}
      {paymentMethods}
      on:subscribe={(e) => handleSubscribe(e.detail)}
      on:cancel={handleCancel}
      on:reactivate={handleReactivate}
    />
  {/if}
</div>

<style>
  .premium-page {
    padding: 20px;
    max-width: 800px;
    margin: 0 auto;
  }
  
  h1 {
    margin-bottom: 20px;
    font-size: 24px;
    font-weight: 600;
  }
  
  .loading, .error {
    padding: 20px;
    text-align: center;
    color: var(--text-secondary);
  }
  
  .error {
    color: var(--danger);
    background: var(--bg-tertiary);
    border-radius: 8px;
  }
</style>
