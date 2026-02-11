<script lang="ts">
  import { presenceStore, getStatusColor, type PresenceStatus } from '$lib/stores/presence';
  
  export let userId: string;
  export let size: 'sm' | 'md' | 'lg' = 'md';
  export let showTooltip = true;
  
  $: presence = presenceStore.getPresence(userId);
  $: status = presence?.status ?? 'offline';
  $: color = getStatusColor(status);
  
  const sizes = {
    sm: 8,
    md: 12,
    lg: 16,
  };
  
  $: sizeValue = sizes[size];
  $: borderWidth = size === 'sm' ? 2 : 3;
</script>

<div
  class="presence-indicator"
  class:has-tooltip={showTooltip}
  style="--size: {sizeValue}px; --color: {color}; --border: {borderWidth}px;"
  title={showTooltip ? status.charAt(0).toUpperCase() + status.slice(1) : undefined}
>
  {#if status === 'idle'}
    <div class="idle-moon"></div>
  {:else if status === 'dnd'}
    <div class="dnd-dash"></div>
  {:else if status === 'offline' || status === 'invisible'}
    <div class="offline-ring"></div>
  {/if}
</div>

<style>
  .presence-indicator {
    width: var(--size);
    height: var(--size);
    border-radius: 50%;
    background-color: var(--color);
    border: var(--border) solid var(--bg-primary, #36393f);
    position: relative;
    flex-shrink: 0;
  }
  
  .has-tooltip {
    cursor: help;
  }
  
  .idle-moon {
    position: absolute;
    top: -1px;
    right: -1px;
    width: 50%;
    height: 50%;
    border-radius: 50%;
    background-color: var(--bg-primary, #36393f);
  }
  
  .dnd-dash {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 60%;
    height: 20%;
    background-color: var(--bg-primary, #36393f);
    border-radius: 1px;
  }
  
  .offline-ring {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 40%;
    height: 40%;
    border-radius: 50%;
    background-color: var(--bg-primary, #36393f);
  }
</style>
