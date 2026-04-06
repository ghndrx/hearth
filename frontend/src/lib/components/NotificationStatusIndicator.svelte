<script lang="ts">
  import { channelNotificationOverrides, type ChannelNotificationLevel } from '$lib/stores/channelNotificationOverrides';
  import { onMount } from 'svelte';

  export let channelId: string;
  export let size: 'sm' | 'md' = 'md';

  $: override = $channelNotificationOverrides.overrides.get(channelId);
  $: hasOverride = $channelNotificationOverrides.overrides.has(channelId);

  onMount(() => {
    // Ensure overrides are loaded
    if ($channelNotificationOverrides.overrides.size === 0) {
      channelNotificationOverrides.loadOverrides();
    }
  });

  const sizeClasses = {
    sm: 'w-3.5 h-3.5',
    md: 'w-4 h-4'
  };

  function getIconPath(level: ChannelNotificationLevel | undefined) {
    switch (level) {
      case 'all_messages':
        return 'M12 22c1.1 0 2-.9 2-2h-4c0 1.1.9 2 2 2zm6-6v-5c0-3.07-1.64-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68C7.63 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2zm-2 1H8v-6c0-2.48 1.51-4.5 4-4.5s4 2.02 4 4.5v6z';
      case 'mentions_only':
        return 'M12 22c1.1 0 2-.9 2-2h-4c0 1.1.9 2 2 2zm6-6v-5c0-3.07-1.63-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68C7.64 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2z';
      case 'nothing':
        return 'M20 18.69L7.84 6.14 5.27 3.49 4 4.76l2.8 2.8v.01c-.52.99-.8 2.16-.8 3.42v5l-2 2v1h13.73l2 2L21 19.72l-1-1.03zM12 22c1.1 0 2-.9 2-2h-4c0 1.1.89 2 2 2zm6-7.32V11c0-3.08-1.64-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68c-.15.03-.29.08-.43.13L18 14.68z';
      default:
        return 'M12 22c1.1 0 2-.9 2-2h-4c0 1.1.9 2 2 2zm6-6v-5c0-3.07-1.64-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68C7.63 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2z';
    }
  }

  function getTooltip(level: ChannelNotificationLevel | undefined, hasOverride: boolean) {
    if (!hasOverride) return 'Default notification settings';
    
    switch (level) {
      case 'all_messages':
        return 'All messages: You will receive notifications for all messages';
      case 'mentions_only':
        return 'Mentions only: You will only receive notifications for mentions';
      case 'nothing':
        return 'Muted: You will not receive any notifications';
      default:
        return 'Notification settings';
    }
  }

  $: iconPath = getIconPath(override);
  $: tooltipText = getTooltip(override, hasOverride);
  $: statusColor = override === 'nothing' 
    ? 'text-[var(--status-danger)]' 
    : override === 'mentions_only'
      ? 'text-[var(--status-warning)]'
      : hasOverride
        ? 'text-[var(--status-success)]'
        : 'text-[var(--text-muted)]';
</script>

<div 
  class="inline-flex items-center justify-center {statusColor}"
  title={tooltipText}
>
  <svg class={sizeClasses[size]} viewBox="0 0 24 24" fill="currentColor">
    <path d={iconPath}/>
  </svg>
</div>
