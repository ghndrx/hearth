<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import Button from './Button.svelte';
	import LoadingSpinner from './LoadingSpinner.svelte';

	export let email: string;
	export let password: string;

	const dispatch = createEventDispatcher<{
		back: void;
		success: void;
	}>();

	let mfaCode = '';
	let loading = false;
	let error = '';

	async function handleMFALogin() {
		if (!mfaCode.trim() || mfaCode.length !== 6) {
			error = 'Please enter a valid 6-digit code';
			return;
		}

		loading = true;
		error = '';

		try {
			await auth.loginWithMFA(email, password, mfaCode);
			dispatch('success');
		} catch (err: any) {
			if (err.status === 400 && err.data?.error === 'invalid_mfa_code') {
				error = 'Invalid authentication code. Please try again.';
			} else {
				error = err.message || 'Authentication failed';
			}
		} finally {
			loading = false;
		}
	}

	function handleBack() {
		dispatch('back');
	}

	function handleKeyPress(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			handleMFALogin();
		}
	}

	// Auto-format MFA code input
	function formatMFACode(value: string) {
		// Only allow digits and limit to 6 characters
		return value.replace(/\D/g, '').slice(0, 6);
	}

	$: mfaCode = formatMFACode(mfaCode);
</script>

<div class="mfa-login-form">
	<div class="form-header">
		<h2>Two-Factor Authentication</h2>
		<p>Enter the 6-digit code from your authenticator app</p>
	</div>

	<form on:submit|preventDefault={handleMFALogin}>
		<div class="input-group">
			<label for="mfa-code">Authentication Code</label>
			<input
				id="mfa-code"
				type="text"
				placeholder="123456"
				maxlength="6"
				pattern="[0-9]{6}"
				bind:value={mfaCode}
				on:keypress={handleKeyPress}
				disabled={loading}
				autocomplete="one-time-code"
				class:error={error}
				autofocus
			/>
			{#if error}
				<p class="error-text">{error}</p>
			{/if}
		</div>

		<div class="button-group">
			<Button type="button" variant="secondary" on:click={handleBack} disabled={loading}>
				Back
			</Button>
			<Button type="submit" disabled={loading || mfaCode.length !== 6}>
				{#if loading}
					<LoadingSpinner size="sm" />
					Verifying...
				{:else}
					Continue
				{/if}
			</Button>
		</div>
	</form>

	<div class="help-text">
		<p>Can't access your authenticator app? Use one of your backup codes instead.</p>
	</div>
</div>

<style>
	.mfa-login-form {
		max-width: 400px;
		margin: 0 auto;
		padding: 32px 24px;
	}

	.form-header {
		text-align: center;
		margin-bottom: 32px;
	}

	.form-header h2 {
		margin: 0 0 8px 0;
		color: var(--text-primary);
		font-size: 28px;
		font-weight: 600;
	}

	.form-header p {
		margin: 0;
		color: var(--text-secondary);
		font-size: 16px;
	}

	.input-group {
		margin-bottom: 24px;
	}

	.input-group label {
		display: block;
		margin-bottom: 8px;
		color: var(--text-primary);
		font-weight: 500;
		font-size: 14px;
	}

	.input-group input {
		width: 100%;
		padding: 16px;
		border: 2px solid var(--border);
		border-radius: 8px;
		background: var(--input-bg);
		color: var(--text-primary);
		font-size: 24px;
		text-align: center;
		letter-spacing: 4px;
		font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
		transition: border-color 0.2s ease;
	}

	.input-group input:focus {
		outline: none;
		border-color: var(--brand-primary);
		box-shadow: 0 0 0 3px rgba(88, 101, 242, 0.1);
	}

	.input-group input.error {
		border-color: var(--error);
	}

	.input-group input:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.error-text {
		margin: 8px 0 0 0;
		color: var(--error);
		font-size: 14px;
	}

	.button-group {
		display: flex;
		gap: 12px;
		margin-bottom: 24px;
	}

	.button-group :global(button) {
		flex: 1;
		padding: 12px 24px;
		font-size: 16px;
	}

	.help-text {
		text-align: center;
		padding-top: 24px;
		border-top: 1px solid var(--border);
	}

	.help-text p {
		margin: 0;
		color: var(--text-secondary);
		font-size: 14px;
		line-height: 1.4;
	}
</style>