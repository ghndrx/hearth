<script>
  export let title = 'Status';
  export let value = '-';
  export let status = 'neutral'; // 'success', 'warning', 'error', 'neutral'
  export let icon = null;
  export let loading = false;

  $: statusClass = {
    success: 'status-success',
    warning: 'status-warning',
    error: 'status-error',
    neutral: 'status-neutral'
  }[status] || 'status-neutral';
</script>

<div class="status-card {statusClass}" class:loading>
  <div class="card-header">
    {#if icon}
      <span class="icon">{icon}</span>
    {/if}
    <h3 class="title">{title}</h3>
  </div>
  
  <div class="card-body">
    {#if loading}
      <div class="skeleton"></div>
    {:else}
      <span class="value">{value}</span>
    {/if}
  </div>
  
  <div class="status-indicator">
    <span class="dot"></span>
    <span class="status-label">{status}</span>
  </div>
</div>

<style>
  .status-card {
    background: var(--card-bg, #ffffff);
    border-radius: 12px;
    padding: 1.25rem;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
    border: 1px solid var(--card-border, #e5e7eb);
    transition: all 0.2s ease;
  }

  .status-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  }

  .card-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }

  .icon {
    font-size: 1.25rem;
  }

  .title {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-secondary, #6b7280);
    text-transform: uppercase;
    letter-spacing: 0.025em;
    margin: 0;
  }

  .card-body {
    margin-bottom: 1rem;
  }

  .value {
    font-size: 2rem;
    font-weight: 700;
    color: var(--text-primary, #111827);
    line-height: 1.2;
  }

  .skeleton {
    height: 2rem;
    width: 60%;
    background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
    background-size: 200% 100%;
    animation: shimmer 1.5s infinite;
    border-radius: 4px;
  }

  @keyframes shimmer {
    0% { background-position: 200% 0; }
    100% { background-position: -200% 0; }
  }

  .status-indicator {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: currentColor;
  }

  .status-label {
    font-size: 0.75rem;
    font-weight: 500;
    text-transform: capitalize;
    color: var(--text-secondary, #6b7280);
  }

  /* Status variants */
  .status-success {
    --dot-color: #10b981;
    border-left: 3px solid #10b981;
  }

  .status-success .dot {
    background: #10b981;
  }

  .status-warning {
    --dot-color: #f59e0b;
    border-left: 3px solid #f59e0b;
  }

  .status-warning .dot {
    background: #f59e0b;
  }

  .status-error {
    --dot-color: #ef4444;
    border-left: 3px solid #ef4444;
  }

  .status-error .dot {
    background: #ef4444;
  }

  .status-neutral {
    --dot-color: #6b7280;
    border-left: 3px solid #6b7280;
  }

  .status-neutral .dot {
    background: #6b7280;
  }

  .loading {
    opacity: 0.7;
    pointer-events: none;
  }
</style>
