<script lang="ts">
  import { onMount } from 'svelte';
  import { channelNotificationOverrides, type ChannelNotificationLevel } from '$lib/stores/channelNotificationOverrides';

  export let channelId: string;
  export let compact = false;

  let isOpen = false;
  let containerRef: HTMLDivElement;

  $: override = $channelNotificationOverrides.overrides.get(channelId);
  $: currentLevel = override ?? 'all_messages';
  $: hasOverride = $channelNotificationOverrides.overrides.has(channelId);

  onMount(() => {
    // Load overrides on mount if not already loaded
    if ($channelNotificationOverrides.overrides.size === 0) {
      channelNotificationOverrides.loadOverrides();
    }
  });

  function handleClickOutside(event: MouseEvent) {
    if (containerRef && !containerRef.contains(event.target as Node)) {
      isOpen = false;
    }
  }

  function setLevel(level: ChannelNotificationLevel) {
    channelNotificationOverrides.setOverride(channelId, level);
    isOpen = false;
  }

  function clearOverride() {
    channelNotificationOverrides.clearOverride(channelId);
    isOpen = false;
  }

  const options: { level: ChannelNotificationLevel; label: string; description: string; icon: string }[] = [
    { 
      level: 'all_messages', 
      label: 'All Messages', 
      description: 'Receive notifications for all messages',
      icon: 'M12 22c1.1 0 2-.9 2-2h-4c0 1.1.9 2 2 2zm6-6v-5c0-3.07-1.63-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68C7.64 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2zm-2 1H8v-6c0-2.48 1.51-4.5 4-4.5s4 2.02 4 4.5v6z'
    },
    { 
      level: 'mentions_only', 
      label: 'Mentions Only', 
      description: 'Only receive notifications when mentioned',
      icon: 'M12 22c1.1 0 2-.9 2-2h-4c0 1.1.9 2 2 2zm6-6v-5c0-3.07-1.63-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68C7.64 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2z'
    },
    { 
      level: 'nothing', 
      label: 'Nothing', 
      description: 'No notifications from this channel',
      icon: 'M20 18.69L7.84 6.14 5.27 3.49 4 4.76l2.8 2.8v.01c-.52.99-.8 2.16-.8 3.42v5l-2 2v1h13.73l2 2L21 19.72l-1-1.03zM12 22c1.1 0 2-.9 2-2h-4c0 1.1.89 2 2 2zm6-7.32V11c0-3.08-1.64-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68c-.15.03-.29.08-.43.13L18 14.68z'
    }
  ];
</script>

<svelte:window on:click={handleClickOutside} />

<div bind:this={containerRef} class="relative inline-block">
  <!-- Trigger Button -->
  <button
    class="flex items-center gap-2 px-3 py-1.5 rounded-md transition-colors
      {hasOverride 
        ? 'bg-[var(--status-warning)]/10 text-[var(--status-warning)] hover:bg-[var(--status-warning)]/20' 
        : 'bg-[var(--bg-modifier-accent)] text-[var(--text-secondary)] hover:bg-[var(--bg-modifier-hover)] hover:text-[var(--text-primary)]'}
      {compact ? 'text-sm' : ''}"
    on:click={() => isOpen = !isOpen}
    title={hasOverride 
      ? `Notifications: ${options.find(o => o.level === currentLevel)?.label}` 
      : 'Notification settings (using default)'}
  >
    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
      {#if currentLevel === 'all_messages'}
        <path d="M12 22c1.1 0 2-.9 2-2h-4c0 1.1.9 2 2 2zm6-6v-5c0-3.07-1.64-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68C7.63 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2zm-2 1H8v-6c0-2.48 1.51-4.5 4-4.5s4 2.02 4 4.5v6z"/>
      {:else if currentLevel === 'mentions_only'}
        <path d="M12 22c1.1 0 2-.9 2-2h-4c0 1.1.9 2 2 2zm6-6v-5c0-3.07-1.63-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68C7.64 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2z"/>
      {:else}
        <path d="M20 18.69L7.84 6.14 5.27 3.49 4 4.76l2.8 2.8v.01c-.52.99-.8 2.16-.8 3.42v5l-2 2v1h13.73l2 2L21 19.72l-1-1.03zM12 22c1.1 0 2-.9 2-2h-4c0 1.1.89 2 2 2zm6-7.32V11c0-3.08-1.64-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68c-.15.03-.29.08-.43.13L18 14.68z"/>
      {/if}
    </svg>
    
    {#if !compact}
      <span class="font-medium">
        {options.find(o => o.level === currentLevel)?.label}
      </span>
    {/if}
    
    {#if hasOverride}
      <span class="w-2 h-2 rounded-full bg-[var(--status-warning)]"></span>
    {/if}
  </button>

  <!-- Dropdown Menu -->
  {#if isOpen}
    <div class="absolute z-50 mt-2 w-72 rounded-lg bg-[var(--bg-secondary)] border border-[var(--bg-modifier-accent)] shadow-lg
      right-0 {compact ? 'top-full' : 'top-full'}"
    >
      <div class="p-3 border-b border-[var(--bg-modifier-accent)]">
        <h3 class="text-sm font-semibold text-[var(--text-primary)]">Notification Settings</h3>
        <p class="text-xs text-[var(--text-muted)] mt-0.5">Choose what you want to be notified about</p>
      </div>
      
      <div class="p-2">
        {#each options as option}
          <button
            class="w-full flex items-start gap-3 p-3 rounded-lg transition-colors text-left
              {currentLevel === option.level 
                ? 'bg-[var(--brand-primary)]/10 border border-[var(--brand-primary)]/30' 
                : 'hover:bg-[var(--bg-modifier-hover)] border border-transparent'}"
            on:click={() => setLevel(option.level)}
          >
            <div class="flex-shrink-0 mt-0.5">
              <svg class="w-5 h-5 {currentLevel === option.level ? 'text-[var(--brand-primary)]' : 'text-[var(--text-secondary)]'}" viewBox="0 0 24 24" fill="currentColor">
                <path d={option.icon}/>
              </svg>
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium text-[var(--text-primary)]">{option.label}</div>
              <div class="text-xs text-[var(--text-muted)] mt-0.5">{option.description}</div>
            </div>
            {#if currentLevel === option.level}
              <div class="flex-shrink-0">
                <div class="w-2 h-2 rounded-full bg-[var(--brand-primary)]"></div>
              </div>
            {/if}
          </button>
        {/each}
      </div>

      {#if hasOverride}
        <div class="p-2 border-t border-[var(--bg-modifier-accent)]">
          <button
            class="w-full flex items-center justify-center gap-2 p-2 rounded-lg text-sm
              text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-modifier-hover)]
              transition-colors"
            on:click={clearOverride}
          >
            <span>Reset to Default</span>
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>
