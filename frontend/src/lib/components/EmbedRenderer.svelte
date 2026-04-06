<script context="module" lang="ts">
	export interface EmbedData {
		id?: string;
		title?: string;
		description?: string;
		url?: string;
		color?: number;
		author?: {
			name?: string;
			url?: string;
			icon?: string;
		};
		footer?: {
			text?: string;
			icon?: string;
		};
		image_url?: string;
		thumbnail_url?: string;
		timestamp?: string;
		type?: string;
		fields?: Array<{
			name: string;
			value: string;
			inline?: boolean;
		}>;
	}
</script>

<script lang="ts">
	export let embed: EmbedData;
	export let showBorder = true;

	// Convert color number to CSS rgb value
	function getColorStyle(color: number | undefined): string {
		if (!color) return '';
		const r = (color >> 16) & 0xff;
		const g = (color >> 8) & 0xff;
		const b = color & 0xff;
		return `rgb(${r}, ${g}, ${b})`;
	}

	function getColorHex(color: number | undefined): string {
		if (!color) return '#000000';
		const r = (color >> 16) & 0xff;
		const g = (color >> 8) & 0xff;
		const b = color & 0xff;
		return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`;
	}

	// Format timestamp
	function formatTimestamp(timestamp: string | undefined): string {
		if (!timestamp) return '';
		try {
			const date = new Date(timestamp);
			return date.toLocaleDateString('en-US', {
				year: 'numeric',
				month: 'short',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch {
			return timestamp;
		}
	}

	$: borderColor = embed.color ? getColorHex(embed.color) : '#4f545c';
	$: hasAuthor = embed.author?.name || embed.author?.icon;
	$: hasFooter = embed.footer?.text || embed.footer?.icon;
	$: hasFields = embed.fields && embed.fields.length > 0;
	$: hasImage = embed.image_url || embed.thumbnail_url;
</script>

<div class="embed" style={showBorder ? `border-left: 4px solid ${borderColor}` : ''}>
	<div class="embed-content">
		<!-- Author Section -->
		{#if hasAuthor}
			<div class="embed-author">
				{#if embed.author?.icon}
					<img class="embed-author-icon" src={embed.author.icon} alt="" />
				{/if}
				{#if embed.author?.name}
					{#if embed.author?.url}
						<a class="embed-author-name" href={embed.author.url} target="_blank" rel="noopener noreferrer">
							{embed.author.name}
						</a>
					{:else}
						<span class="embed-author-name">{embed.author.name}</span>
					{/if}
				{/if}
			</div>
		{/if}

		<!-- Title -->
		{#if embed.title}
			<div class="embed-title">
				{#if embed.url}
					<a href={embed.url} target="_blank" rel="noopener noreferrer">{embed.title}</a>
				{:else}
					{embed.title}
				{/if}
			</div>
		{/if}

		<!-- Description -->
		{#if embed.description}
			<div class="embed-description">{embed.description}</div>
		{/if}

		<!-- Fields -->
		{#if hasFields}
			<div class="embed-fields">
				{#each embed.fields as field}
					<div class="embed-field" class:inline={field.inline}>
						<div class="embed-field-name">{field.name}</div>
						<div class="embed-field-value">{field.value}</div>
					</div>
				{/each}
			</div>
		{/if}

		<!-- Image/Thumbnail -->
		{#if embed.image_url}
			<div class="embed-image-container">
				<img class="embed-image" src={embed.image_url} alt="" />
			</div>
		{/if}
	</div>

	<!-- Thumbnail (side image) -->
	{#if embed.thumbnail_url && !embed.image_url}
		<div class="embed-thumbnail-container">
			<img class="embed-thumbnail" src={embed.thumbnail_url} alt="" />
		</div>
	{/if}

	<!-- Footer -->
	{#if hasFooter}
		<div class="embed-footer">
			{#if embed.footer?.icon}
				<img class="embed-footer-icon" src={embed.footer.icon} alt="" />
			{/if}
			<span class="embed-footer-text">
				{embed.footer?.text || ''}
				{#if embed.timestamp && embed.footer?.text}
					<span class="embed-timestamp-separator">•</span>
				{/if}
				{#if embed.timestamp}
					<span class="embed-timestamp">{formatTimestamp(embed.timestamp)}</span>
				{/if}
			</span>
		</div>
	{/if}
</div>

<style>
	.embed {
		display: flex;
		flex-direction: column;
		background: rgba(79, 84, 92, 0.1);
		border-radius: 4px;
		padding: 8px 16px;
		margin: 4px 0;
		max-width: 520px;
	}

	.embed-content {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	/* Author */
	.embed-author {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.embed-author-icon {
		width: 24px;
		height: 24px;
		border-radius: 50%;
		object-fit: cover;
	}

	.embed-author-name {
		font-size: 13px;
		font-weight: 600;
		color: #fff;
		text-decoration: none;
	}

	.embed-author-name:hover {
		text-decoration: underline;
	}

	/* Title */
	.embed-title {
		font-size: 14px;
		font-weight: 600;
		color: #fff;
	}

	.embed-title a {
		color: #fff;
		text-decoration: none;
	}

	.embed-title a:hover {
		text-decoration: underline;
	}

	/* Description */
	.embed-description {
		font-size: 14px;
		color: #b9bbbe;
		line-height: 1.375;
		white-space: pre-wrap;
		word-break: break-word;
	}

	/* Fields */
	.embed-fields {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-top: 4px;
	}

	.embed-field {
		flex: 0 0 100%;
		min-width: 0;
	}

	.embed-field.inline {
		flex: 1 1 150px;
		min-width: 100px;
		max-width: 250px;
	}

	.embed-field-name {
		font-size: 12px;
		font-weight: 600;
		color: #fff;
		margin-bottom: 2px;
	}

	.embed-field-value {
		font-size: 13px;
		color: #b9bbbe;
		line-height: 1.375;
	}

	/* Images */
	.embed-image-container {
		margin-top: 8px;
		border-radius: 4px;
		overflow: hidden;
	}

	.embed-image {
		max-width: 100%;
		max-height: 300px;
		object-fit: contain;
		display: block;
	}

	.embed-thumbnail-container {
		flex-shrink: 0;
		margin-left: 16px;
	}

	.embed-thumbnail {
		width: 20px;
		height: 20px;
		object-fit: contain;
		border-radius: 4px;
	}

	/* Footer */
	.embed-footer {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 8px;
		padding-top: 8px;
		border-top: 1px solid rgba(79, 84, 92, 0.3);
	}

	.embed-footer-icon {
		width: 20px;
		height: 20px;
		border-radius: 50%;
		object-fit: cover;
	}

	.embed-footer-text {
		font-size: 12px;
		color: #72767d;
	}

	.embed-timestamp-separator {
		margin: 0 4px;
	}

	.embed-timestamp {
		color: #72767d;
	}
</style>
