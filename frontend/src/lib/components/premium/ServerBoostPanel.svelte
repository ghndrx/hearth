<script lang="ts">
  import { onMount } from 'svelte';
  import type { ServerBoost, ServerPerks } from '$lib/types/premium';
  
  export let serverId: string;
  
  let boosts: ServerBoost[] = [];
  let perks: ServerPerks | null = null;
  let userBoosted = false;
  let loading = true;
  let error = '';
  
  $: boostCount = boosts.length;
  $: nextLevelBoosts = perks ? perks.boosts_required - boostCount : 0;
  
  onMount(async () => {
    await Promise.all([loadServerBoosts(), loadServerPerks(), checkUserBoost()]);
    loading = false;
  });
  
  async function loadServerBoosts() {
    try {
      const res = await fetch(`/api/v1/servers/${serverId}/boosts`);
      if (res.ok) {
        boosts = await res.json();
      }
    } catch (e) {
      error = 'Failed to load boosts';
    }
  }
  
  async function loadServerPerks() {
    try {
      const res = await fetch(`/api/v1/servers/${serverId}/perks`);
      if (res.ok) {
        perks = await res.json();
      }
    } catch (e) {
      // Non-premium servers may not have perks
    }
  }
  
  async function checkUserBoost() {
    try {
      const res = await fetch(`/api/v1/premium/boosts`);
      if (res.ok) {
        const userBoosts: ServerBoost[] = await res.json();
        userBoosted = userBoosts.some((b: ServerBoost) => b.server_id === serverId && b.active);
      }
    } catch (e) {
      // User might not be premium
    }
  }
  
  async function boostServer() {
    try {
      const res = await fetch(`/api/v1/servers/${serverId}/boost`, { method: 'POST' });
      if (res.ok) {
        userBoosted = true;
        await loadServerBoosts();
        await loadServerPerks();
      } else {
        const data = await res.json();
        error = data.message || 'Failed to boost server';
      }
    } catch (e) {
      error = 'Failed to boost server';
    }
  }
  
  async function unboostServer() {
    try {
      const res = await fetch(`/api/v1/servers/${serverId}/boost`, { method: 'DELETE' });
      if (res.ok) {
        userBoosted = false;
        await loadServerBoosts();
        await loadServerPerks();
      }
    } catch (e) {
      error = 'Failed to unboost server';
    }
  }
  
  function formatBytes(bytes: number): string {
    if (bytes < 1024 * 1024) {
      return `${(bytes / 1024).toFixed(0)}KB`;
    }
    return `${(bytes / (1024 * 1024)).toFixed(0)}MB`;
  }
  
  function formatBitrate(bps: number): string {
    return `${(bps / 1000).toFixed(0)}kbps`;
  }
</script>

<div class="server-boost-panel">
  <div class="boost-header">
    <h3>Server Boosts</h3>
    <div class="boost-level">
      Level {perks?.level || 0}
    </div>
  </div>
  
  {#if loading}
    <div class="loading">Loading...</div>
  {:else}
    <div class="boost-stats">
      <div class="stat">
        <span class="stat-value">{boostCount}</span>
        <span class="stat-label">Total Boosts</span>
      </div>
      {#if perks && perks.level < 3}
        <div class="stat">
          <span class="stat-value">{nextLevelBoosts}</span>
          <span class="stat-label">to Level {perks.level + 1}</span>
        </div>
      {/if}
    </div>
    
    <div class="perks-list">
      <h4>Current Perks</h4>
      <ul>
        <li>
          <span class="perk-icon">🎨</span>
          <span class="perk-label">Emoji Limit</span>
          <span class="perk-value">{perks?.emoji_limit || 50}</span>
        </li>
        <li>
          <span class="perk-icon">📎</span>
          <span class="perk-label">File Upload</span>
          <span class="perk-value">{formatBytes(perks?.file_upload_limit || 8 * 1024 * 1024)}</span>
        </li>
        <li>
          <span class="perk-icon">🎧</span>
          <span class="perk-label">Voice Bitrate</span>
          <span class="perk-value">{formatBitrate(perks?.voice_bitrate || 96000)}</span>
        </li>
        <li>
          <span class="perk-icon">🔗</span>
          <span class="perk-label">Vanity URL</span>
          <span class="perk-value">{perks?.has_vanity_url ? '✓' : '✗'}</span>
        </li>
        <li>
          <span class="perk-icon">🎬</span>
          <span class="perk-label">Animated Icon</span>
          <span class="perk-value">{perks?.has_animated_icon ? '✓' : '✗'}</span>
        </li>
        <li>
          <span class="perk-icon">🖼️</span>
          <span class="perk-label">Server Banner</span>
          <span class="perk-value">{perks?.has_banner ? '✓' : '✗'}</span>
        </li>
        <li>
          <span class="perk-icon">✨</span>
          <span class="perk-label">Splash Screen</span>
          <span class="perk-value">{perks?.has_splash_screen ? '✓' : '✗'}</span>
        </li>
      </ul>
    </div>
    
    <div class="boost-actions">
      {#if userBoosted}
        <button class="btn-unboost" on:click={unboostServer}>
          Remove Boost
        </button>
        <p class="boosted-text">You are boosting this server!</p>
      {:else}
        <button class="btn-boost" on:click={boostServer}>
          Boost this Server
        </button>
      {/if}
    </div>
    
    <div class="boosters-list">
      <h4>Top Boosters</h4>
      {#if boosts.length > 0}
        <ul>
          {#each boosts.slice(0, 5) as boost}
            <li>
              <span class="booster-avatar">👤</span>
              <span class="booster-name">User {boost.user_id.slice(0, 8)}</span>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="no-boosters">No boosts yet. Be the first to boost!</p>
      {/if}
    </div>
  {/if}
  
  {#if error}
    <div class="error">{error}</div>
  {/if}
</div>

<style>
  .server-boost-panel {
    background: var(--bg-secondary);
    border-radius: 8px;
    padding: 16px;
  }
  
  .boost-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }
  
  .boost-header h3 {
    font-size: 16px;
    font-weight: 600;
  }
  
  .boost-level {
    background: linear-gradient(135deg, #FFD700, #FFA500);
    color: black;
    padding: 4px 12px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 600;
  }
  
  .loading {
    text-align: center;
    padding: 20px;
    color: var(--text-secondary);
  }
  
  .boost-stats {
    display: flex;
    gap: 24px;
    margin-bottom: 16px;
  }
  
  .stat {
    display: flex;
    flex-direction: column;
  }
  
  .stat-value {
    font-size: 20px;
    font-weight: 700;
  }
  
  .stat-label {
    font-size: 12px;
    color: var(--text-secondary);
  }
  
  .perks-list {
    margin-bottom: 16px;
  }
  
  .perks-list h4 {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }
  
  .perks-list ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  
  .perks-list li {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
    font-size: 13px;
  }
  
  .perk-icon {
    width: 20px;
    text-align: center;
  }
  
  .perk-label {
    flex: 1;
  }
  
  .perk-value {
    font-weight: 500;
  }
  
  .boost-actions {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
  }
  
  .btn-boost, .btn-unboost {
    width: 100%;
    padding: 10px;
    border-radius: 4px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .btn-boost {
    background: linear-gradient(135deg, #FFD700, #FFA500);
    border: none;
    color: black;
  }
  
  .btn-boost:hover {
    opacity: 0.9;
  }
  
  .btn-unboost {
    background: transparent;
    border: 1px solid var(--danger);
    color: var(--danger);
  }
  
  .btn-unboost:hover {
    background: var(--danger);
    color: white;
  }
  
  .boosted-text {
    font-size: 12px;
    color: var(--success);
  }
  
  .boosters-list h4 {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }
  
  .boosters-list ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  
  .boosters-list li {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 0;
    font-size: 13px;
  }
  
  .booster-avatar {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: var(--bg-tertiary);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  
  .no-boosters {
    font-size: 13px;
    color: var(--text-secondary);
  }
  
  .error {
    margin-top: 12px;
    padding: 8px;
    background: var(--danger-bg);
    color: var(--danger);
    border-radius: 4px;
    font-size: 12px;
  }
</style>
