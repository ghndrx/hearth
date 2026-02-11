<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$stores';

	let username = '';
	let email = '';
	let password = '';
	let confirmPassword = '';
	let loading = false;
	let validationError = '';

	function validateForm(): boolean {
		if (password !== confirmPassword) {
			validationError = 'Passwords do not match';
			return false;
		}
		if (password.length < 8) {
			validationError = 'Password must be at least 8 characters';
			return false;
		}
		if (username.length < 2 || username.length > 32) {
			validationError = 'Username must be between 2 and 32 characters';
			return false;
		}
		if (!/^[a-zA-Z0-9_]+$/.test(username)) {
			validationError = 'Username can only contain letters, numbers, and underscores';
			return false;
		}
		validationError = '';
		return true;
	}

	async function handleSubmit() {
		if (!validateForm()) return;

		loading = true;
		auth.clearError();

		const success = await auth.register({ username, email, password });

		if (success) {
			goto('/');
		}

		loading = false;
	}
</script>

<svelte:head>
	<title>Create Account - Hearth</title>
</svelte:head>

<div class="min-h-screen bg-dark-950 flex items-center justify-center p-4">
	<div class="w-full max-w-md">
		<div class="bg-dark-800 rounded-lg shadow-xl p-8">
			<!-- Logo/Header -->
			<div class="text-center mb-8">
				<div class="w-16 h-16 bg-hearth-500 rounded-2xl flex items-center justify-center mx-auto mb-4">
					<svg class="w-10 h-10 text-white" fill="currentColor" viewBox="0 0 24 24">
						<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
					</svg>
				</div>
				<h1 class="text-2xl font-bold text-white">Create an account</h1>
			</div>

			<!-- Error message -->
			{#if $auth.error || validationError}
				<div class="bg-red-500/10 border border-red-500/50 text-red-400 px-4 py-3 rounded-md mb-6">
					{validationError || $auth.error}
				</div>
			{/if}

			<!-- Form -->
			<form on:submit|preventDefault={handleSubmit} class="space-y-5">
				<div>
					<label for="email" class="block text-xs font-semibold uppercase text-gray-400 mb-2">
						Email <span class="text-red-400">*</span>
					</label>
					<input
						id="email"
						type="email"
						class="input"
						bind:value={email}
						required
						disabled={loading}
					/>
				</div>

				<div>
					<label for="username" class="block text-xs font-semibold uppercase text-gray-400 mb-2">
						Username <span class="text-red-400">*</span>
					</label>
					<input
						id="username"
						type="text"
						class="input"
						bind:value={username}
						required
						disabled={loading}
						minlength="2"
						maxlength="32"
					/>
				</div>

				<div>
					<label for="password" class="block text-xs font-semibold uppercase text-gray-400 mb-2">
						Password <span class="text-red-400">*</span>
					</label>
					<input
						id="password"
						type="password"
						class="input"
						bind:value={password}
						required
						disabled={loading}
						minlength="8"
					/>
				</div>

				<div>
					<label for="confirmPassword" class="block text-xs font-semibold uppercase text-gray-400 mb-2">
						Confirm Password <span class="text-red-400">*</span>
					</label>
					<input
						id="confirmPassword"
						type="password"
						class="input"
						bind:value={confirmPassword}
						required
						disabled={loading}
					/>
				</div>

				<button
					type="submit"
					class="btn btn-primary w-full py-3"
					disabled={loading}
				>
					{#if loading}
						<span class="flex items-center justify-center gap-2">
							<span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
							Creating account...
						</span>
					{:else}
						Continue
					{/if}
				</button>
			</form>

			<p class="text-sm text-gray-500 mt-4">
				By registering, you agree to Hearth's Terms of Service and Privacy Policy.
			</p>

			<p class="text-gray-400 text-sm mt-6">
				<a href="/login" class="text-hearth-400 hover:underline">Already have an account?</a>
			</p>
		</div>
	</div>
</div>
