<script lang="ts">
	import { fade, fly } from 'svelte/transition';
	import { createEventDispatcher } from 'svelte';
	import { user, auth } from '$lib/stores/auth';
	import { settings, type Theme, type MessageDisplay } from '$lib/stores/settings';
	import Avatar from './Avatar.svelte';
	
	export let open = false;
	
	const dispatch = createEventDispatcher();
	
	let activeSection = 'account';
	let editingProfile = false;
	let saving = false;
	let profileForm = {
		display_name: '',
		bio: '',
		pronouns: ''
	};
	
	const sections = [
		{ id: 'divider-user', label: 'User Settings', divider: true },
		{ id: 'account', label: 'My Account', icon: 'user' },
		{ id: 'profile', label: 'User Profile', icon: 'profile' },
		{ id: 'privacy', label: 'Privacy & Safety', icon: 'shield' },
		{ id: 'divider-app', label: 'App Settings', divider: true },
		{ id: 'appearance', label: 'Appearance', icon: 'palette' },
		{ id: 'notifications', label: 'Notifications', icon: 'bell' },
		{ id: 'keybinds', label: 'Keybinds', icon: 'keyboard' },
		{ id: 'divider-other', label: '', divider: true },
		{ id: 'about', label: 'About Hearth', icon: 'info' },
		{ id: 'logout', label: 'Log Out', icon: 'logout', danger: true }
	];
	
	$: if (open && $user) {
		profileForm = {
			display_name: $user.display_name || '',
			bio: $user.bio || '',
			pronouns: $user.pronouns || ''
		};
	}
	
	$: appSettings = $settings.app;
	
	function close() {
		open = false;
		editingProfile = false;
		dispatch('close');
	}
	
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') close();
	}
	
	async function handleSectionClick(id: string) {
		if (id === 'logout') {
			if (confirm('Are you sure you want to log out?')) {
				await auth.logout();
				close();
			}
			return;
		}
		
		if (!id.startsWith('divider')) {
			activeSection = id;
		}
	}
	
	async function saveProfile() {
		saving = true;
		try {
			await auth.updateProfile({
				display_name: profileForm.display_name || null,
				bio: profileForm.bio || null,
				pronouns: profileForm.pronouns || null
			});
			editingProfile = false;
		} catch (error) {
			console.error('Failed to save profile:', error);
		} finally {
			saving = false;
		}
	}
	
	function setTheme(theme: Theme) {
		settings.updateApp({ theme });
	}
	
	function setMessageDisplay(display: MessageDisplay) {
		settings.updateApp({ messageDisplay: display });
	}
	
	function toggleSetting(key: keyof typeof appSettings) {
		settings.updateApp({ [key]: !appSettings[key] } as any);
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
	<div 
		class="settings-overlay"
		transition:fade={{ duration: 150 }}
		on:click|self={close}
		role="dialog"
		aria-modal="true"
		aria-labelledby="settings-title"
	>
		<div 
			class="settings-container"
			transition:fly={{ y: 20, duration: 200 }}
		>
			<!-- Sidebar -->
			<nav class="settings-sidebar" aria-label="Settings navigation">
				<div class="sidebar-content">
					{#each sections as section}
						{#if section.divider}
							<div class="sidebar-divider">
								{#if section.label}
									<span>{section.label}</span>
								{/if}
							</div>
						{:else}
							<button
								class="sidebar-item"
								class:active={activeSection === section.id}
								class:danger={section.danger}
								on:click={() => handleSectionClick(section.id)}
							>
								{section.label}
							</button>
						{/if}
					{/each}
				</div>
			</nav>
			
			<!-- Content -->
			<main class="settings-content">
				<div class="content-scroll">
					{#if activeSection === 'account'}
						<section>
							<h1 id="settings-title">My Account</h1>
							
							<div class="account-card">
								<div class="account-banner" style="background: linear-gradient(135deg, #5865f2 0%, #eb459e 100%);">
								</div>
								<div class="account-info">
									<div class="account-avatar">
										<Avatar 
											src={$user?.avatar} 
											alt={$user?.username || ''} 
											size={80}
										/>
									</div>
									<div class="account-details">
										<span class="account-name">{$user?.display_name || $user?.username}</span>
										<span class="account-tag">@{$user?.username}</span>
									</div>
									<button class="btn btn-primary" on:click={() => editingProfile = true}>
										Edit User Profile
									</button>
								</div>
							</div>
							
							<div class="settings-section">
								<h2>Account Details</h2>
								
								<div class="field-group">
									<div class="field">
										<label>Username</label>
										<div class="field-value">
											<span>{$user?.username}</span>
											<button class="btn-edit">Edit</button>
										</div>
									</div>
									
									<div class="field">
										<label>Email</label>
										<div class="field-value">
											<span>{$user?.email}</span>
											<button class="btn-edit">Edit</button>
										</div>
									</div>
									
									<div class="field">
										<label>Phone Number</label>
										<div class="field-value">
											<span class="muted">Not set</span>
											<button class="btn-edit">Add</button>
										</div>
									</div>
								</div>
							</div>
							
							<div class="settings-section">
								<h2>Password and Authentication</h2>
								<button class="btn btn-secondary">Change Password</button>
								
								<div class="mfa-section">
									<div>
										<h3>Two-Factor Authentication</h3>
										<p class="description">
											Protect your account with an extra layer of security.
										</p>
									</div>
									<button class="btn btn-primary">Enable 2FA</button>
								</div>
							</div>
							
							<div class="settings-section danger-zone">
								<h2>Account Removal</h2>
								<p class="description">
									Deleting your account is permanent and cannot be undone.
								</p>
								<button class="btn btn-danger">Delete Account</button>
							</div>
						</section>
					
					{:else if activeSection === 'profile'}
						<section>
							<h1>User Profile</h1>
							
							<div class="profile-editor">
								<div class="profile-preview">
									<div class="preview-card">
										<div class="preview-banner" style="background: linear-gradient(135deg, #5865f2 0%, #eb459e 100%);">
										</div>
										<div class="preview-body">
											<div class="preview-avatar">
												<Avatar 
													src={$user?.avatar} 
													alt={$user?.username || ''} 
													size={92}
												/>
											</div>
											<div class="preview-info">
												<span class="preview-name">
													{profileForm.display_name || $user?.username}
												</span>
												<span class="preview-tag">@{$user?.username}</span>
												{#if profileForm.pronouns}
													<span class="preview-pronouns">{profileForm.pronouns}</span>
												{/if}
											</div>
											{#if profileForm.bio}
												<div class="preview-bio">
													<h4>About Me</h4>
													<p>{profileForm.bio}</p>
												</div>
											{/if}
										</div>
									</div>
								</div>
								
								<div class="profile-form">
									<div class="form-field">
										<label for="display-name">Display Name</label>
										<input 
											type="text" 
											id="display-name"
											bind:value={profileForm.display_name}
											placeholder={$user?.username}
											maxlength={32}
										/>
									</div>
									
									<div class="form-field">
										<label for="pronouns">Pronouns</label>
										<input 
											type="text" 
											id="pronouns"
											bind:value={profileForm.pronouns}
											placeholder="Add your pronouns"
											maxlength={40}
										/>
									</div>
									
									<div class="form-field">
										<label for="bio">About Me</label>
										<textarea 
											id="bio"
											bind:value={profileForm.bio}
											placeholder="Tell us about yourself"
											maxlength={190}
											rows={4}
										></textarea>
										<span class="char-count">{profileForm.bio.length}/190</span>
									</div>
									
									<div class="form-actions">
										<button 
											class="btn btn-primary" 
											on:click={saveProfile}
											disabled={saving}
										>
											{saving ? 'Saving...' : 'Save Changes'}
										</button>
									</div>
								</div>
							</div>
						</section>
					
					{:else if activeSection === 'privacy'}
						<section>
							<h1>Privacy & Safety</h1>
							
							<div class="settings-section">
								<h2>Direct Messages</h2>
								
								<div class="toggle-item">
									<div>
										<span class="toggle-label">Allow DMs from server members</span>
										<span class="toggle-description">
											Allow members from servers you're in to send you direct messages.
										</span>
									</div>
									<label class="toggle">
										<input type="checkbox" checked />
										<span class="toggle-slider"></span>
									</label>
								</div>
								
								<div class="toggle-item">
									<div>
										<span class="toggle-label">Allow message requests</span>
										<span class="toggle-description">
											People you haven't messaged before can send you requests.
										</span>
									</div>
									<label class="toggle">
										<input type="checkbox" checked />
										<span class="toggle-slider"></span>
									</label>
								</div>
							</div>
							
							<div class="settings-section">
								<h2>Data & Privacy</h2>
								<button class="btn btn-secondary">Request My Data</button>
							</div>
						</section>
					
					{:else if activeSection === 'appearance'}
						<section>
							<h1>Appearance</h1>
							
							<div class="settings-section">
								<h2>Theme</h2>
								
								<div class="theme-options">
									<button 
										class="theme-option"
										class:selected={appSettings.theme === 'dark'}
										on:click={() => setTheme('dark')}
									>
										<div class="theme-preview dark-preview">
											<div class="preview-sidebar"></div>
											<div class="preview-content">
												<div class="preview-message"></div>
												<div class="preview-message short"></div>
											</div>
										</div>
										<span>Dark</span>
									</button>
									
									<button 
										class="theme-option"
										class:selected={appSettings.theme === 'light'}
										on:click={() => setTheme('light')}
									>
										<div class="theme-preview light-preview">
											<div class="preview-sidebar"></div>
											<div class="preview-content">
												<div class="preview-message"></div>
												<div class="preview-message short"></div>
											</div>
										</div>
										<span>Light</span>
									</button>
									
									<button 
										class="theme-option"
										class:selected={appSettings.theme === 'midnight'}
										on:click={() => setTheme('midnight')}
									>
										<div class="theme-preview midnight-preview">
											<div class="preview-sidebar"></div>
											<div class="preview-content">
												<div class="preview-message"></div>
												<div class="preview-message short"></div>
											</div>
										</div>
										<span>Midnight</span>
									</button>
								</div>
							</div>
							
							<div class="settings-section">
								<h2>Message Display</h2>
								
								<div class="display-options">
									<button 
										class="display-option"
										class:selected={appSettings.messageDisplay === 'cozy'}
										on:click={() => setMessageDisplay('cozy')}
									>
										<div class="display-preview cozy">
											<div class="msg-avatar"></div>
											<div class="msg-content">
												<div class="msg-header"></div>
												<div class="msg-text"></div>
											</div>
										</div>
										<span>Cozy</span>
									</button>
									
									<button 
										class="display-option"
										class:selected={appSettings.messageDisplay === 'compact'}
										on:click={() => setMessageDisplay('compact')}
									>
										<div class="display-preview compact">
											<div class="msg-timestamp"></div>
											<div class="msg-content">
												<div class="msg-text"></div>
											</div>
										</div>
										<span>Compact</span>
									</button>
								</div>
							</div>
							
							<div class="settings-section">
								<h2>Chat Font Size</h2>
								<div class="slider-field">
									<input 
										type="range" 
										min="12" 
										max="24" 
										step="1"
										value={appSettings.fontSize}
										on:input={(e) => settings.updateApp({ fontSize: parseInt(e.currentTarget.value) })}
									/>
									<span class="slider-value">{appSettings.fontSize}px</span>
								</div>
							</div>
							
							<div class="settings-section">
								<h2>Accessibility</h2>
								
								<div class="toggle-item">
									<div>
										<span class="toggle-label">Enable Animations</span>
										<span class="toggle-description">
											Show animated avatars, emoji, and stickers.
										</span>
									</div>
									<label class="toggle">
										<input 
											type="checkbox" 
											checked={appSettings.enableAnimations}
											on:change={() => toggleSetting('enableAnimations')}
										/>
										<span class="toggle-slider"></span>
									</label>
								</div>
							</div>
						</section>
					
					{:else if activeSection === 'notifications'}
						<section>
							<h1>Notifications</h1>
							
							<div class="settings-section">
								<h2>Desktop Notifications</h2>
								
								<div class="toggle-item">
									<div>
										<span class="toggle-label">Enable Desktop Notifications</span>
										<span class="toggle-description">
											Receive notifications even when Hearth is minimized.
										</span>
									</div>
									<label class="toggle">
										<input 
											type="checkbox" 
											checked={appSettings.notificationsEnabled}
											on:change={() => toggleSetting('notificationsEnabled')}
										/>
										<span class="toggle-slider"></span>
									</label>
								</div>
								
								<div class="toggle-item">
									<div>
										<span class="toggle-label">Enable Sounds</span>
										<span class="toggle-description">
											Play notification sounds for messages.
										</span>
									</div>
									<label class="toggle">
										<input 
											type="checkbox" 
											checked={appSettings.enableSounds}
											on:change={() => toggleSetting('enableSounds')}
										/>
										<span class="toggle-slider"></span>
									</label>
								</div>
							</div>
						</section>
					
					{:else if activeSection === 'keybinds'}
						<section>
							<h1>Keybinds</h1>
							
							<div class="keybind-list">
								<div class="keybind-item">
									<span>Navigate Up</span>
									<kbd>↑</kbd>
								</div>
								<div class="keybind-item">
									<span>Navigate Down</span>
									<kbd>↓</kbd>
								</div>
								<div class="keybind-item">
									<span>Focus Message Input</span>
									<kbd>Tab</kbd>
								</div>
								<div class="keybind-item">
									<span>Quick Switcher</span>
									<div class="keybind-combo">
										<kbd>Ctrl</kbd>
										<span>+</span>
										<kbd>K</kbd>
									</div>
								</div>
								<div class="keybind-item">
									<span>Search</span>
									<div class="keybind-combo">
										<kbd>Ctrl</kbd>
										<span>+</span>
										<kbd>F</kbd>
									</div>
								</div>
								<div class="keybind-item">
									<span>Mark as Read</span>
									<kbd>Escape</kbd>
								</div>
								<div class="keybind-item">
									<span>Upload File</span>
									<div class="keybind-combo">
										<kbd>Ctrl</kbd>
										<span>+</span>
										<kbd>U</kbd>
									</div>
								</div>
							</div>
						</section>
					
					{:else if activeSection === 'about'}
						<section>
							<h1>About Hearth</h1>
							
							<div class="about-content">
								<div class="about-logo">
									<span class="hearth-icon">🔥</span>
									<h2>Hearth</h2>
								</div>
								
								<p class="about-description">
									A self-hosted, end-to-end encrypted chat platform. 
									Your conversations, your data, your servers.
								</p>
								
								<div class="about-info">
									<div class="info-row">
										<span>Version</span>
										<span>0.1.0-alpha</span>
									</div>
									<div class="info-row">
										<span>Build</span>
										<span>Development</span>
									</div>
								</div>
								
								<div class="about-links">
									<a href="https://github.com/yourusername/hearth" target="_blank" rel="noopener">
										GitHub Repository
									</a>
									<a href="/docs" target="_blank" rel="noopener">
										Documentation
									</a>
								</div>
								
								<div class="toggle-item">
									<div>
										<span class="toggle-label">Developer Mode</span>
										<span class="toggle-description">
											Show developer options and debug information.
										</span>
									</div>
									<label class="toggle">
										<input 
											type="checkbox" 
											checked={appSettings.developerMode}
											on:change={() => toggleSetting('developerMode')}
										/>
										<span class="toggle-slider"></span>
									</label>
								</div>
							</div>
						</section>
					{/if}
				</div>
				
				<!-- Close button -->
				<button class="close-btn" on:click={close} aria-label="Close settings">
					<div class="close-icon">
						<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
							<path d="M18.3 5.71a.996.996 0 0 0-1.41 0L12 10.59 7.11 5.7A.996.996 0 1 0 5.7 7.11L10.59 12 5.7 16.89a.996.996 0 1 0 1.41 1.41L12 13.41l4.89 4.89a.996.996 0 1 0 1.41-1.41L13.41 12l4.89-4.89c.38-.38.38-1.02 0-1.4z"/>
						</svg>
					</div>
					<span class="close-keybind">ESC</span>
				</button>
			</main>
		</div>
	</div>
{/if}

<style>
	.settings-overlay {
		position: fixed;
		inset: 0;
		background: var(--bg-tertiary);
		z-index: 1000;
		display: flex;
	}
	
	.settings-container {
		display: flex;
		width: 100%;
		height: 100%;
	}
	
	/* Sidebar */
	.settings-sidebar {
		width: 218px;
		background: var(--bg-secondary);
		display: flex;
		justify-content: flex-end;
		padding: 60px 6px 60px 20px;
		flex-shrink: 0;
	}
	
	.sidebar-content {
		width: 192px;
	}
	
	.sidebar-divider {
		padding: 6px 10px;
		margin-top: 8px;
	}
	
	.sidebar-divider span {
		font-size: 12px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.02em;
		color: var(--text-muted);
	}
	
	.sidebar-item {
		width: 100%;
		padding: 6px 10px;
		margin-bottom: 2px;
		background: none;
		border: none;
		border-radius: 4px;
		font-size: 16px;
		color: var(--text-secondary);
		cursor: pointer;
		text-align: left;
	}
	
	.sidebar-item:hover {
		background: var(--bg-modifier-hover);
		color: var(--text-primary);
	}
	
	.sidebar-item.active {
		background: var(--bg-modifier-selected);
		color: var(--text-primary);
	}
	
	.sidebar-item.danger {
		color: var(--status-danger);
	}
	
	.sidebar-item.danger:hover {
		background: rgba(242, 63, 67, 0.1);
	}
	
	/* Content */
	.settings-content {
		flex: 1;
		position: relative;
		display: flex;
		justify-content: flex-start;
		padding: 60px 40px 80px;
	}
	
	.content-scroll {
		width: 100%;
		max-width: 740px;
		overflow-y: auto;
	}
	
	section h1 {
		font-size: 20px;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 20px;
	}
	
	section h2 {
		font-size: 12px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.02em;
		color: var(--text-muted);
		margin-bottom: 8px;
	}
	
	section h3 {
		font-size: 16px;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 4px;
	}
	
	.settings-section {
		margin-bottom: 40px;
		padding-bottom: 40px;
		border-bottom: 1px solid var(--bg-modifier-accent);
	}
	
	.description {
		color: var(--text-secondary);
		font-size: 14px;
		margin-bottom: 16px;
	}
	
	/* Account Card */
	.account-card {
		background: var(--bg-secondary);
		border-radius: 8px;
		overflow: hidden;
		margin-bottom: 40px;
	}
	
	.account-banner {
		height: 100px;
	}
	
	.account-info {
		display: flex;
		align-items: center;
		padding: 16px;
		gap: 16px;
	}
	
	.account-avatar {
		margin-top: -40px;
		border: 6px solid var(--bg-secondary);
		border-radius: 50%;
	}
	
	.account-details {
		flex: 1;
	}
	
	.account-name {
		display: block;
		font-size: 20px;
		font-weight: 600;
		color: var(--text-primary);
	}
	
	.account-tag {
		color: var(--text-secondary);
	}
	
	/* Fields */
	.field-group {
		background: var(--bg-secondary);
		border-radius: 8px;
		padding: 16px;
	}
	
	.field {
		padding: 12px 0;
		border-bottom: 1px solid var(--bg-modifier-accent);
	}
	
	.field:last-child {
		border-bottom: none;
	}
	
	.field label {
		display: block;
		font-size: 12px;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--text-muted);
		margin-bottom: 4px;
	}
	
	.field-value {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}
	
	.field-value span {
		color: var(--text-primary);
	}
	
	.field-value .muted {
		color: var(--text-muted);
	}
	
	.btn-edit {
		background: none;
		border: none;
		color: var(--text-link);
		font-size: 14px;
		cursor: pointer;
	}
	
	.btn-edit:hover {
		text-decoration: underline;
	}
	
	/* MFA Section */
	.mfa-section {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-top: 16px;
		padding: 16px;
		background: var(--bg-secondary);
		border-radius: 8px;
	}
	
	/* Buttons */
	.btn {
		padding: 8px 16px;
		border: none;
		border-radius: 4px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s ease;
	}
	
	.btn-primary {
		background: var(--brand-primary);
		color: white;
	}
	
	.btn-primary:hover {
		background: var(--brand-hover);
	}
	
	.btn-primary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
	
	.btn-secondary {
		background: var(--bg-modifier-accent);
		color: var(--text-primary);
	}
	
	.btn-secondary:hover {
		background: var(--bg-modifier-selected);
	}
	
	.btn-danger {
		background: var(--status-danger);
		color: white;
	}
	
	.btn-danger:hover {
		background: #d62f33;
	}
	
	/* Danger Zone */
	.danger-zone {
		border-color: rgba(242, 63, 67, 0.2);
	}
	
	/* Profile Editor */
	.profile-editor {
		display: grid;
		grid-template-columns: 300px 1fr;
		gap: 40px;
	}
	
	.preview-card {
		background: var(--bg-secondary);
		border-radius: 8px;
		overflow: hidden;
	}
	
	.preview-banner {
		height: 60px;
	}
	
	.preview-body {
		padding: 16px;
	}
	
	.preview-avatar {
		margin-top: -50px;
		margin-bottom: 8px;
		border: 4px solid var(--bg-secondary);
		border-radius: 50%;
		width: fit-content;
	}
	
	.preview-info {
		margin-bottom: 12px;
	}
	
	.preview-name {
		display: block;
		font-size: 20px;
		font-weight: 600;
		color: var(--text-primary);
	}
	
	.preview-tag {
		color: var(--text-secondary);
		font-size: 14px;
	}
	
	.preview-pronouns {
		display: inline-block;
		margin-left: 8px;
		padding: 2px 8px;
		background: var(--bg-tertiary);
		border-radius: 4px;
		font-size: 12px;
		color: var(--text-muted);
	}
	
	.preview-bio {
		padding-top: 12px;
		border-top: 1px solid var(--bg-modifier-accent);
	}
	
	.preview-bio h4 {
		font-size: 12px;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--text-primary);
		margin-bottom: 8px;
	}
	
	.preview-bio p {
		color: var(--text-secondary);
		font-size: 14px;
		white-space: pre-wrap;
	}
	
	/* Form */
	.form-field {
		margin-bottom: 20px;
	}
	
	.form-field label {
		display: block;
		font-size: 12px;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--text-muted);
		margin-bottom: 8px;
	}
	
	.form-field input,
	.form-field textarea {
		width: 100%;
		padding: 10px;
		background: var(--bg-tertiary);
		border: none;
		border-radius: 4px;
		color: var(--text-primary);
		font-size: 16px;
		font-family: inherit;
	}
	
	.form-field input:focus,
	.form-field textarea:focus {
		outline: 2px solid var(--brand-primary);
	}
	
	.form-field textarea {
		resize: vertical;
		min-height: 100px;
	}
	
	.char-count {
		display: block;
		text-align: right;
		font-size: 12px;
		color: var(--text-muted);
		margin-top: 4px;
	}
	
	.form-actions {
		display: flex;
		justify-content: flex-end;
	}
	
	/* Toggle */
	.toggle-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 16px 0;
		border-bottom: 1px solid var(--bg-modifier-accent);
	}
	
	.toggle-item:last-child {
		border-bottom: none;
	}
	
	.toggle-label {
		display: block;
		font-size: 16px;
		color: var(--text-primary);
		margin-bottom: 4px;
	}
	
	.toggle-description {
		font-size: 14px;
		color: var(--text-muted);
	}
	
	.toggle {
		position: relative;
		display: inline-block;
		width: 40px;
		height: 24px;
		flex-shrink: 0;
	}
	
	.toggle input {
		opacity: 0;
		width: 0;
		height: 0;
	}
	
	.toggle-slider {
		position: absolute;
		cursor: pointer;
		inset: 0;
		background: var(--bg-modifier-accent);
		border-radius: 24px;
		transition: background 0.2s;
	}
	
	.toggle-slider::before {
		content: '';
		position: absolute;
		height: 18px;
		width: 18px;
		left: 3px;
		bottom: 3px;
		background: white;
		border-radius: 50%;
		transition: transform 0.2s;
	}
	
	.toggle input:checked + .toggle-slider {
		background: var(--brand-primary);
	}
	
	.toggle input:checked + .toggle-slider::before {
		transform: translateX(16px);
	}
	
	/* Theme Options */
	.theme-options {
		display: flex;
		gap: 16px;
	}
	
	.theme-option {
		background: none;
		border: 2px solid transparent;
		border-radius: 8px;
		padding: 8px;
		cursor: pointer;
		text-align: center;
	}
	
	.theme-option.selected {
		border-color: var(--brand-primary);
	}
	
	.theme-option span {
		display: block;
		margin-top: 8px;
		color: var(--text-primary);
		font-size: 14px;
	}
	
	.theme-preview {
		width: 120px;
		height: 80px;
		border-radius: 4px;
		display: flex;
		overflow: hidden;
	}
	
	.dark-preview {
		background: #313338;
	}
	
	.dark-preview .preview-sidebar {
		width: 30px;
		background: #1e1f22;
	}
	
	.dark-preview .preview-content {
		flex: 1;
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		justify-content: center;
	}
	
	.dark-preview .preview-message {
		height: 12px;
		background: #2b2d31;
		border-radius: 2px;
	}
	
	.dark-preview .preview-message.short {
		width: 60%;
	}
	
	.light-preview {
		background: #ffffff;
	}
	
	.light-preview .preview-sidebar {
		width: 30px;
		background: #f2f3f5;
	}
	
	.light-preview .preview-content {
		flex: 1;
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		justify-content: center;
	}
	
	.light-preview .preview-message {
		height: 12px;
		background: #e3e5e8;
		border-radius: 2px;
	}
	
	.light-preview .preview-message.short {
		width: 60%;
	}
	
	.midnight-preview {
		background: #000000;
	}
	
	.midnight-preview .preview-sidebar {
		width: 30px;
		background: #0a0a0a;
	}
	
	.midnight-preview .preview-content {
		flex: 1;
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		justify-content: center;
	}
	
	.midnight-preview .preview-message {
		height: 12px;
		background: #1a1a1a;
		border-radius: 2px;
	}
	
	.midnight-preview .preview-message.short {
		width: 60%;
	}
	
	/* Display Options */
	.display-options {
		display: flex;
		gap: 16px;
	}
	
	.display-option {
		background: none;
		border: 2px solid transparent;
		border-radius: 8px;
		padding: 12px;
		cursor: pointer;
		text-align: center;
	}
	
	.display-option.selected {
		border-color: var(--brand-primary);
	}
	
	.display-option span {
		display: block;
		margin-top: 8px;
		color: var(--text-primary);
		font-size: 14px;
	}
	
	.display-preview {
		width: 200px;
		height: 60px;
		background: var(--bg-secondary);
		border-radius: 4px;
		padding: 8px;
		display: flex;
		gap: 8px;
		align-items: flex-start;
	}
	
	.display-preview.cozy .msg-avatar {
		width: 40px;
		height: 40px;
		border-radius: 50%;
		background: var(--brand-primary);
		flex-shrink: 0;
	}
	
	.display-preview.cozy .msg-content {
		flex: 1;
	}
	
	.display-preview.cozy .msg-header {
		height: 14px;
		width: 80px;
		background: var(--bg-modifier-accent);
		border-radius: 2px;
		margin-bottom: 4px;
	}
	
	.display-preview.cozy .msg-text {
		height: 12px;
		background: var(--bg-modifier-accent);
		border-radius: 2px;
	}
	
	.display-preview.compact {
		align-items: center;
	}
	
	.display-preview.compact .msg-timestamp {
		width: 40px;
		height: 12px;
		background: var(--bg-modifier-accent);
		border-radius: 2px;
	}
	
	.display-preview.compact .msg-text {
		height: 12px;
		width: 100px;
		background: var(--bg-modifier-accent);
		border-radius: 2px;
	}
	
	/* Slider */
	.slider-field {
		display: flex;
		align-items: center;
		gap: 16px;
	}
	
	.slider-field input[type="range"] {
		flex: 1;
		height: 8px;
		-webkit-appearance: none;
		background: var(--bg-modifier-accent);
		border-radius: 4px;
		outline: none;
	}
	
	.slider-field input[type="range"]::-webkit-slider-thumb {
		-webkit-appearance: none;
		width: 16px;
		height: 16px;
		border-radius: 50%;
		background: var(--brand-primary);
		cursor: pointer;
	}
	
	.slider-value {
		min-width: 40px;
		color: var(--text-secondary);
	}
	
	/* Keybinds */
	.keybind-list {
		background: var(--bg-secondary);
		border-radius: 8px;
		padding: 8px 16px;
	}
	
	.keybind-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 12px 0;
		border-bottom: 1px solid var(--bg-modifier-accent);
	}
	
	.keybind-item:last-child {
		border-bottom: none;
	}
	
	.keybind-item span {
		color: var(--text-primary);
	}
	
	.keybind-combo {
		display: flex;
		align-items: center;
		gap: 4px;
		color: var(--text-muted);
	}
	
	kbd {
		display: inline-block;
		padding: 4px 8px;
		background: var(--bg-tertiary);
		border-radius: 4px;
		font-size: 12px;
		font-family: inherit;
		color: var(--text-primary);
	}
	
	/* About */
	.about-content {
		text-align: center;
		max-width: 400px;
		margin: 0 auto;
	}
	
	.about-logo {
		margin-bottom: 16px;
	}
	
	.hearth-icon {
		font-size: 48px;
	}
	
	.about-logo h2 {
		font-size: 32px;
		color: var(--text-primary);
		text-transform: none;
		letter-spacing: normal;
	}
	
	.about-description {
		color: var(--text-secondary);
		margin-bottom: 24px;
	}
	
	.about-info {
		background: var(--bg-secondary);
		border-radius: 8px;
		padding: 16px;
		margin-bottom: 24px;
	}
	
	.info-row {
		display: flex;
		justify-content: space-between;
		padding: 8px 0;
	}
	
	.info-row:first-child {
		border-bottom: 1px solid var(--bg-modifier-accent);
	}
	
	.info-row span:first-child {
		color: var(--text-muted);
	}
	
	.info-row span:last-child {
		color: var(--text-primary);
	}
	
	.about-links {
		display: flex;
		justify-content: center;
		gap: 24px;
		margin-bottom: 32px;
	}
	
	.about-links a {
		color: var(--text-link);
	}
	
	.about-content .toggle-item {
		text-align: left;
	}
	
	/* Close Button */
	.close-btn {
		position: absolute;
		top: 60px;
		right: 40px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		background: none;
		border: none;
		cursor: pointer;
		color: var(--text-muted);
	}
	
	.close-btn:hover {
		color: var(--text-primary);
	}
	
	.close-icon {
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		border: 2px solid currentColor;
		border-radius: 50%;
	}
	
	.close-keybind {
		font-size: 12px;
		font-weight: 600;
	}
	
	/* Responsive */
	@media (max-width: 1024px) {
		.settings-sidebar {
			width: 180px;
			padding: 40px 6px 40px 12px;
		}
		
		.sidebar-content {
			width: 100%;
		}
		
		.settings-content {
			padding: 40px 20px 60px;
		}
		
		.profile-editor {
			grid-template-columns: 1fr;
		}
		
		.close-btn {
			top: 20px;
			right: 20px;
		}
	}
	
	@media (max-width: 768px) {
		.settings-container {
			flex-direction: column;
		}
		
		.settings-sidebar {
			width: 100%;
			padding: 16px;
			justify-content: flex-start;
			overflow-x: auto;
		}
		
		.sidebar-content {
			display: flex;
			gap: 8px;
		}
		
		.sidebar-divider {
			display: none;
		}
		
		.sidebar-item {
			white-space: nowrap;
			margin-bottom: 0;
		}
		
		.theme-options {
			flex-wrap: wrap;
		}
		
		.theme-preview {
			width: 100px;
			height: 60px;
		}
	}
</style>
