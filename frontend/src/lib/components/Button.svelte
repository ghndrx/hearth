<script lang="ts">
  export let variant: 'primary' | 'secondary' | 'danger' | 'ghost' = 'primary';
  export let size: 'sm' | 'md' | 'lg' = 'md';
  export let disabled = false;
  export let type: 'button' | 'submit' | 'reset' = 'button';
  export let fullWidth = false;
  export let ariaLabel: string | undefined = undefined;

  function handleClick(event: MouseEvent) {
    if (disabled) {
      event.preventDefault();
      event.stopPropagation();
      return;
    }
  }
</script>

<button
  {type}
  {disabled}
  class="button"
  class:primary={variant === 'primary'}
  class:secondary={variant === 'secondary'}
  class:danger={variant === 'danger'}
  class:ghost={variant === 'ghost'}
  class:sm={size === 'sm'}
  class:md={size === 'md'}
  class:lg={size === 'lg'}
  class:full-width={fullWidth}
  aria-label={ariaLabel}
  on:click={handleClick}
  on:click
>
  <slot />
</button>

<style>
  .button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    border: none;
    border-radius: 6px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s ease;
    font-family: inherit;
  }

  .button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .sm {
    padding: 6px 12px;
    font-size: 13px;
  }

  .md {
    padding: 10px 16px;
    font-size: 14px;
  }

  .lg {
    padding: 14px 24px;
    font-size: 16px;
  }

  .full-width {
    width: 100%;
  }

  .primary {
    background-color: #5865f2;
    color: white;
  }

  .primary:hover:not(:disabled) {
    background-color: #4752c4;
  }

  .secondary {
    background-color: #4f545c;
    color: white;
  }

  .secondary:hover:not(:disabled) {
    background-color: #686d73;
  }

  .danger {
    background-color: #ed4245;
    color: white;
  }

  .danger:hover:not(:disabled) {
    background-color: #c03537;
  }

  .ghost {
    background-color: transparent;
    color: #b9bbbe;
  }

  .ghost:hover:not(:disabled) {
    background-color: rgba(255, 255, 255, 0.1);
    color: white;
  }
</style>
