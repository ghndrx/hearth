<script lang="ts">
  import type { PaymentMethod } from '$lib/types/premium';
  
  export let methods: PaymentMethod[] = [];
  
  let error = '';
  
  async function removeMethod(id: string) {
    if (!confirm('Are you sure you want to remove this payment method?')) return;
    
    try {
      const res = await fetch(`/api/v1/premium/payment-methods/${id}`, { method: 'DELETE' });
      if (res.ok) {
        methods = methods.filter(m => m.id !== id);
      } else {
        error = 'Failed to remove payment method';
      }
    } catch (e) {
      error = 'Failed to remove payment method';
    }
  }
  
  function getCardIcon(brand?: string): string {
    switch (brand?.toLowerCase()) {
      case 'visa':
        return '💳';
      case 'mastercard':
        return '💳';
      case 'amex':
        return '💳';
      default:
        return '💳';
    }
  }
</script>

<div class="payment-methods">
  {#if error}
    <div class="error">{error}</div>
  {/if}
  
  {#if methods.length === 0}
    <p class="empty">No payment methods on file.</p>
  {:else}
    <div class="method-list">
      {#each methods as method}
        <div class="method-item">
          <div class="method-icon">{getCardIcon(method.brand)}</div>
          <div class="method-info">
            <span class="method-type">
              {method.brand || 'Card'} ending in {method.last4}
            </span>
            {#if method.expires_at}
              <span class="method-expires">Expires {method.expires_at}</span>
            {/if}
          </div>
          <div class="method-actions">
            {#if method.is_default}
              <span class="default-badge">Default</span>
            {/if}
            <button 
              class="btn-remove" 
              on:click={() => removeMethod(method.id)}
              disabled={methods.length === 1 && method.is_default}
            >
              Remove
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .payment-methods {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  
  .error {
    padding: 12px;
    background: var(--danger-bg);
    color: var(--danger);
    border-radius: 4px;
    font-size: 14px;
  }
  
  .empty {
    color: var(--text-secondary);
    text-align: center;
    padding: 20px;
  }
  
  .method-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  .method-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: var(--bg-tertiary);
    border-radius: 4px;
  }
  
  .method-icon {
    font-size: 24px;
  }
  
  .method-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  
  .method-type {
    font-weight: 500;
    text-transform: capitalize;
  }
  
  .method-expires {
    font-size: 12px;
    color: var(--text-secondary);
  }
  
  .method-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  
  .default-badge {
    font-size: 11px;
    padding: 2px 8px;
    background: var(--success-bg);
    color: var(--success);
    border-radius: 4px;
  }
  
  .btn-remove {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-secondary);
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
  }
  
  .btn-remove:hover:not(:disabled) {
    border-color: var(--danger);
    color: var(--danger);
  }
  
  .btn-remove:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
