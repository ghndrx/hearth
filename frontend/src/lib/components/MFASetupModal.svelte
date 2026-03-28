<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import Modal from './Modal.svelte';
	import Button from './Button.svelte';
	import LoadingSpinner from './LoadingSpinner.svelte';

	export let open = false;

	const dispatch = createEventDispatcher<{
		close: void;
		complete: void;
	}>();

	let step: 'setup' | 'verify' = 'setup';
	let loading = false;
	let error = '';
	let setupData: { secret: string; qr_code_url: string; backup_codes: string[] } | null = null;
	let verificationCode = '';
	let backupCodesShown = false;

	async function handleSetupStart() {
		loading = true;
		error = '';

		try {
			setupData = await auth.enableMFA();
			step = 'verify';
		} catch (err: any) {
			error = err.message || 'Failed to start MFA setup';
		} finally {
			loading = false;
		}
	}

	async function handleVerify() {
		if (!verificationCode.trim()) {
			error = 'Please enter the verification code';
			return;
		}

		loading = true;
		error = '';

		try {
			await auth.verifyMFASetup(verificationCode);
			backupCodesShown = true;
		} catch (err: any) {
			error = err.message || 'Invalid verification code';
		} finally {
			loading = false;
		}
	}

	function handleComplete() {
		dispatch('complete');
		handleClose();
	}

	function handleClose() {
		step = 'setup';
		setupData = null;
		verificationCode = '';
		backupCodesShown = false;
		loading = false;
		error = '';
		dispatch('close');
	}

	function downloadBackupCodes() {
		if (!setupData) return;

		const content = setupData.backup_codes.join('\n');
		const blob = new Blob([content], { type: 'text/plain' });
		const url = URL.createObjectURL(blob);

		const a = document.createElement('a');
		a.href = url;
		a.download = 'hearth-backup-codes.txt';
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);

		URL.revokeObjectURL(url);
	}

	// Start setup automatically when modal opens
	$: if (open && step === 'setup' && !setupData && !loading) {
		handleSetupStart();
	}
</script>

<Modal bind:open on:close={handleClose}>
	<div class="mfa-setup-modal">
		{#if step === 'setup' || (step === 'verify' && !setupData)}
			<div class="setup-step">
				<h2>Setting up Two-Factor Authentication</h2>
				{#if loading}
					<div class="loading-container">
						<LoadingSpinner />
						<p>Generating your MFA setup...</p>
					</div>
				{:else if error}
					<div class="error-message">
						<p>{error}</p>
						<Button on:click={handleSetupStart}>Try Again</Button>
					</div>
				{/if}
			</div>
		{:else if step === 'verify' && !backupCodesShown && setupData}
			<div class="verify-step">
				<h2>Verify Two-Factor Authentication</h2>
				<p>Scan the QR code with your authenticator app, then enter the 6-digit code below:</p>

				<div class="qr-code-container">
					<img src={setupData.qr_code_url} alt="QR Code for MFA setup" />
				</div>

				<div class="secret-container">
					<p><strong>Manual entry key:</strong></p>
					<code class="secret-code">{setupData.secret}</code>
				</div>

				<div class="verification-form">
					<label for="verification-code">Verification Code:</label>
					<input
						id="verification-code"
						type="text"
						placeholder="123456"
						maxlength="6"
						pattern="[0-9]{6}"
						bind:value={verificationCode}
						on:keypress={(e) => e.key === 'Enter' && handleVerify()}
						disabled={loading}
					/>

					{#if error}
						<p class="error">{error}</p>
					{/if}

					<div class="button-group">
						<Button variant="secondary" on:click={handleClose} disabled={loading}>
							Cancel
						</Button>
						<Button on:click={handleVerify} disabled={loading || !verificationCode.trim()}>
							{#if loading}
								<LoadingSpinner size="sm" />
								Verifying...
							{:else}
								Verify & Enable MFA
							{/if}
						</Button>
					</div>
				</div>
			</div>
		{:else if backupCodesShown && setupData}
			<div class="backup-codes-step">
				<h2>Backup Codes</h2>
				<p class="backup-warning">
					<strong>Important:</strong> Save these backup codes in a safe place.
					You can use them to access your account if you lose your authenticator device.
				</p>

				<div class="backup-codes-list">
					{#each setupData.backup_codes as code}
						<code class="backup-code">{code}</code>
					{/each}
				</div>

				<div class="button-group">
					<Button variant="secondary" on:click={downloadBackupCodes}>
						Download Codes
					</Button>
					<Button on:click={handleComplete}>
						I've Saved My Codes
					</Button>
				</div>
			</div>
		{/if}
	</div>
</Modal>

<style>
	.mfa-setup-modal {
		padding: 24px;
		min-width: 400px;
		max-width: 500px;
	}

	.setup-step h2,
	.verify-step h2,
	.backup-codes-step h2 {
		margin: 0 0 16px 0;
		color: var(--text-primary);
		font-size: 24px;
		font-weight: 600;
	}

	.loading-container {
		text-align: center;
		padding: 32px;
	}

	.loading-container p {
		margin-top: 16px;
		color: var(--text-secondary);
	}

	.error-message {
		text-align: center;
		padding: 24px;
	}

	.error-message p {
		color: var(--error);
		margin-bottom: 16px;
	}

	.qr-code-container {
		text-align: center;
		margin: 24px 0;
	}

	.qr-code-container img {
		border: 1px solid var(--border);
		border-radius: 8px;
		background: white;
		padding: 16px;
	}

	.secret-container {
		margin: 24px 0;
		padding: 16px;
		background: var(--bg-secondary);
		border-radius: 8px;
		text-align: center;
	}

	.secret-container p {
		margin: 0 0 8px 0;
		color: var(--text-secondary);
		font-size: 14px;
	}

	.secret-code {
		display: inline-block;
		padding: 8px 12px;
		background: var(--bg-primary);
		border: 1px solid var(--border);
		border-radius: 4px;
		font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
		font-size: 16px;
		letter-spacing: 1px;
		color: var(--text-primary);
		word-break: break-all;
	}

	.verification-form {
		margin-top: 24px;
	}

	.verification-form label {
		display: block;
		margin-bottom: 8px;
		color: var(--text-primary);
		font-weight: 500;
	}

	.verification-form input {
		width: 100%;
		padding: 12px;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--input-bg);
		color: var(--text-primary);
		font-size: 18px;
		text-align: center;
		letter-spacing: 2px;
		font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
	}

	.verification-form input:focus {
		outline: none;
		border-color: var(--brand-primary);
		box-shadow: 0 0 0 2px rgba(88, 101, 242, 0.2);
	}

	.error {
		color: var(--error);
		margin: 8px 0;
		font-size: 14px;
	}

	.backup-warning {
		padding: 16px;
		background: var(--warning-bg, #fef3cd);
		border: 1px solid var(--warning-border, #f0ad4e);
		border-radius: 6px;
		color: var(--warning-text, #8a6d3b);
		margin-bottom: 24px;
	}

	.backup-codes-list {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 8px;
		margin-bottom: 24px;
		padding: 16px;
		background: var(--bg-secondary);
		border-radius: 8px;
	}

	.backup-code {
		padding: 8px 12px;
		background: var(--bg-primary);
		border: 1px solid var(--border);
		border-radius: 4px;
		font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
		font-size: 14px;
		text-align: center;
		color: var(--text-primary);
	}

	.button-group {
		display: flex;
		gap: 12px;
		justify-content: flex-end;
		margin-top: 24px;
	}
</style>