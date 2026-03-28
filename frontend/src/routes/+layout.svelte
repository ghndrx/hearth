<script lang="ts">
	import { onMount } from 'svelte';
	import { auth, isAuthenticated, isLoading } from '$lib/stores/auth';
	import { gateway } from '$lib/gateway';
	import { initE2EE, isE2EEReady } from '$lib/e2ee';
	import '$lib/styles/theme.css';
	
	let e2eeReady = false;
	
	onMount(async () => {
		// Initialize E2EE first (before auth to ensure keys are ready)
		const e2eeResult = await initE2EE({ autoRegister: false });
		e2eeReady = e2eeResult.success;
		console.info('[Layout] E2EE initialization:', e2eeResult);
		
		// Then initialize auth
		auth.init();
	});
	
	// Connect gateway when authenticated AND E2EE is ready
	$: if ($isAuthenticated && e2eeReady) {
		const token = localStorage.getItem('hearth_token');
		if (token) {
			gateway.connect(token);
		}
	}
</script>

<svelte:head>
	<title>Hearth</title>
	<meta name="description" content="Self-hosted chat with E2EE" />
</svelte:head>

{#if $isLoading}
	<div class="loading">
		<div class="spinner"></div>
		<p>Loading...</p>
	</div>
{:else}
	<slot />
{/if}

<style>
	:global(body) {
		margin: 0;
		padding: 0;
	}
	
	.loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100vh;
		background: var(--bg-tertiary);
		color: var(--text-muted);
	}
	
	.spinner {
		width: 48px;
		height: 48px;
		border: 3px solid var(--bg-modifier-accent);
		border-top-color: var(--brand-primary);
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}
	
	@keyframes spin {
		to { transform: rotate(360deg); }
	}
	
	.loading p {
		margin-top: 16px;
	}
</style>
