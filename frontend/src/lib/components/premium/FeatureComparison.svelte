<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { PremiumTier } from '$lib/types/premium';
  import { TIER_COMPARISONS } from '$lib/types/premium';
  
  export let currentTier: PremiumTier;
  
  const dispatch = createEventDispatcher();
  
  function handleSubscribe(tier: PremiumTier) {
    if (tier === 'basic' || tier === 'premium') {
      dispatch('subscribe', tier);
    }
  }
  
  $: tierOrder = ['free', 'basic', 'premium'];
  $: currentIndex = tierOrder.indexOf(currentTier);
</script>

<div class="feature-comparison">
  <div class="tiers-grid">
    {#each TIER_COMPARISONS as tier}
      <div class="tier-card {tier.tier}" class:current={tier.tier === currentTier}>
        {#if tier.recommended}
          <div class="recommended-badge">Recommended</div>
        {/if}
        
        <div class="tier-header">
          <h3>{tier.name}</h3>
          <div class="price">
            {#if tier.price === 0}
              <span class="amount">Free</span>
            {:else}
              <span class="currency">$</span>
              <span class="amount">{tier.price.toFixed(2)}</span>
              <span class="period">/month</span>
            {/if}
          </div>
        </div>
        
        <ul class="features-list">
          {#each tier.features as feature}
            <li>
              <span class="check">✓</span>
              {feature}
            </li>
          {/each}
        </ul>
        
        <div class="tier-action">
          {#if tier.tier === currentTier}
            <button class="btn-current" disabled>Current Plan</button>
          {:else if tier.tier === 'free'}
            <button class="btn-free" disabled>Free Plan</button>
          {:else}
            <button 
              class="btn-subscribe" 
              on:click={() => handleSubscribe(tier.tier)}
            >
              Subscribe to {tier.name}
            </button>
          {/if}
        </div>
      </div>
    {/each}
  </div>
</div>

<style>
  .feature-comparison {
    overflow-x: auto;
  }
  
  .tiers-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
    min-width: 600px;
  }
  
  .tier-card {
    position: relative;
    background: var(--bg-tertiary);
    border-radius: 8px;
    padding: 20px;
    display: flex;
    flex-direction: column;
  }
  
  .tier-card.current {
    border: 2px solid var(--primary);
  }
  
  .tier-card.premium {
    background: linear-gradient(180deg, rgba(255, 215, 0, 0.1) 0%, var(--bg-tertiary) 100%);
  }
  
  .recommended-badge {
    position: absolute;
    top: -10px;
    left: 50%;
    transform: translateX(-50%);
    background: linear-gradient(135deg, #FFD700, #FFA500);
    color: black;
    padding: 4px 12px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 600;
    white-space: nowrap;
  }
  
  .tier-header {
    text-align: center;
    margin-bottom: 20px;
  }
  
  .tier-header h3 {
    font-size: 18px;
    font-weight: 600;
    margin-bottom: 8px;
  }
  
  .price {
    display: flex;
    align-items: baseline;
    justify-content: center;
    gap: 2px;
  }
  
  .currency {
    font-size: 16px;
    color: var(--text-secondary);
  }
  
  .amount {
    font-size: 28px;
    font-weight: 700;
  }
  
  .period {
    font-size: 14px;
    color: var(--text-secondary);
  }
  
  .features-list {
    list-style: none;
    padding: 0;
    margin: 0 0 20px 0;
    flex: 1;
  }
  
  .features-list li {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 6px 0;
    font-size: 13px;
  }
  
  .check {
    color: var(--success);
    font-weight: 700;
  }
  
  .tier-action {
    margin-top: auto;
  }
  
  .tier-action button {
    width: 100%;
    padding: 10px;
    border-radius: 4px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .btn-current {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    cursor: not-allowed;
  }
  
  .btn-free {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text-secondary);
  }
  
  .btn-subscribe {
    background: var(--primary);
    border: none;
    color: white;
  }
  
  .btn-subscribe:hover {
    opacity: 0.9;
  }
  
  .tier-card.basic .btn-subscribe {
    background: var(--primary);
  }
  
  .tier-card.premium .btn-subscribe {
    background: linear-gradient(135deg, #FFD700, #FFA500);
    color: black;
  }
</style>
