<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$stores';

	let email = '';
	let password = '';
	let loading = false;

	async function handleSubmit() {
		loading = true;
		auth.clearError();
		
		const success = await auth.login({ email, password });
		
		if (success) {
			goto('/');
		}
		
		loading = false;
	}
</script>

<svelte:head>
	<title>Login - Hearth</title>
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
				<h1 class="text-2xl font-bold text-white">Welcome back!</h1>
				<p class="text-gray-400 mt-1">We're so excited to see you again!</p>
			</div>

			<!-- Error message -->
			{#if $auth.error}
				<div class="bg-red-500/10 border border-red-500/50 text-red-400 px-4 py-3 rounded-md mb-6">
					{$auth.error}
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
					/>
					<a href="/forgot-password" class="text-sm text-hearth-400 hover:underline mt-2 inline-block">
						Forgot your password?
					</a>
				</div>

				<button
					type="submit"
					class="btn btn-primary w-full py-3"
					disabled={loading}
				>
					{#if loading}
						<span class="flex items-center justify-center gap-2">
							<span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
							Logging in...
						</span>
					{:else}
						Log In
					{/if}
				</button>
			</form>

			<p class="text-gray-400 text-sm mt-6">
				Need an account?
				<a href="/register" class="text-hearth-400 hover:underline">Register</a>
			</p>
		</div>
	</div>
</div>
