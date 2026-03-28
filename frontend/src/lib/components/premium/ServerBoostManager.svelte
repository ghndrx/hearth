<script lang="ts">
  import { onMount } from 'svelte';
  import type { ServerBoost, ServerPerks } from '$lib/types/premium';
  
  export let boostsUsed: number;
  export let boostsTotal: number;
  
  let userBoosts: ServerBoost[] = [];
  let serverBoosts: Map<string, { boosts: ServerBoost[]; perks: ServerPerks }> = new Map();
  let loading = true;
  let error = '';
  
  $: boostsAvailable = boostsTotal - boostsUsed;
  
  onMount(async () => {
    await loadBoosts();
  });
  
  async function loadBoosts() {
    try {
      // Get user's boosts
      const res = await fetch('/api/v1/premium/boosts');
      if (res.ok) {
        userBoosts = await res.json();
      }
    } catch (e) {
      error = 'Failed to load boosts';
    } finally {
      loading = false;
    }
  }
  
  async function unboostServer(serverId: string) {
    try {
      const res = await fetch(`/api/v1/servers/${serverId}/boost`, { method: 'DELETE' });
      if (res.ok) {
        userBoosts = userBoosts.filter(b => b.server_id !== serverId);
        await loadBoosts();
      }
    } catch (e) {
      error = 'Failed to unboost server';
    }
  }
</script>

<div class="boost-manager">
  <div class="boost-status">
    <div class="boost-count">
      <span class="count">{boostsUsed}</span>
      <span class="separator">/</span>
      <span class="total">{boostsTotal}</span>
      <span class="label">boosts used</span>
    </div>
    <div class="boost-count available">
      <span class="count">{boostsAvailable}</span>
      <span class="label">available</span>
    </div>
  </div>
  
  {#if loading}
    <div class="loading">Loading your boosts...</div>
  {:else if userBoosts.length > 0}
    <div class="boosted-servers">
      <h3>Servers You're Boosting</h3>
      <div class="server-list">
        {#each userBoosts as boost}
          <div class="server-item">
            <div class="server-icon">
              <img src="/assets/server-default.png" alt="" />
            </div>
            <div class="server-info">
              <span class="server-name">Server {boost.server_id.slice(0, 8)}</span>
              <span class="boost-date">Boosted since {new Date(boost.created_at).toLocaleDateString()}</span>
            </div>
            <button class="btn-unboost" on:click={() => unboostServer(boost.server_id)}>
              Remove Boost
            </button>
          </div>
        {/each}
      </div>
    </div>
  {:else}
    <div class="empty-state">
      <p>You haven't boosted any servers yet. Use your available boosts to help your favorite servers unlock perks!</p>
    </div>
  {/if}
  
  {#if error}
    <div class="error">{error}</div>
  {/if}
</div>

<style>
  .boost-manager {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  
  .boost-status {
    display: flex;
    gap: 24px;
  }
  
  .boost-count {
    display: flex;
    align-items: baseline;
    gap: 4px;
  }
  
  .boost-count .count {
    font-size: 24px;
    font-weight: 700;
  }
  
  .boost-count .separator {
    color: var(--text-secondary);
  }
  
  .boost-count .total {
    font-size: 18px;
    color: var(--text-secondary);
  }
  
  .boost-count .label {
    font-size: 14px;
    color: var(--text-secondary);
    margin-left: 8px;
  }
  
  .boost-count.available {
    color: var(--success);
  }
  
  .boosted-servers h3 {
    font-size: 14px;
    font-weight: 600;
    margin-bottom: 12px;
  }
  
  .server-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  .server-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: var(--bg-tertiary);
    border-radius: 4px;
  }
  
  .server-icon {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    overflow: hidden;
    background: var(--bg-primary);
  }
  
  .server-icon img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  
  .server-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  
  .server-name {
    font-weight: 500;
  }
  
  .boost-date {
    font-size: 12px;
    color: var(--text-secondary);
  }
  
  .btn-unboost {
    background: transparent;
    border: 1px solid var(--danger);
    color: var(--danger);
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
  }
  
  .btn-unboost:hover {
    background: var(--danger);
    color: white;
  }
  
  .empty-state {
    padding: 20px;
    background: var(--bg-tertiary);
    border-radius: 4px;
    text-align: center;
    color: var(--text-secondary);
  }
  
  .loading {
    padding: 20px;
    text-align: center;
    color: var(--text-secondary);
  }
  
  .error {
    padding: 12px;
    background: var(--danger-bg);
    color: var(--danger);
    border-radius: 4px;
    font-size: 14px;
  }
</style>
