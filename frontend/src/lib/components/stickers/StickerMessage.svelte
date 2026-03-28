<script lang="ts">
	export let stickerId: string;
	export let stickerName: string;
	export let stickerUrl: string;
	export let stickerFormat: string = 'PNG';
</script>

<div class="sticker-message" role="img" aria-label="Sticker: {stickerName}">
	<img 
		src={stickerUrl} 
		alt={stickerName}
		class="sticker-image"
		class:animated={stickerFormat === 'GIF' || stickerFormat === 'APNG'}
	/>
</div>

<style>
	.sticker-message {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 8px 0;
	}

	.sticker-image {
		max-width: 120px;
		max-height: 120px;
		object-fit: contain;
		image-rendering: -webkit-optimize-contrast;
		border-radius: 4px;
	}

	.sticker-image.animated {
		image-rendering: auto;
	}
</style>
