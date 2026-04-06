<script lang="ts">
	import { createEventDispatcher, onDestroy } from 'svelte';

	export let maxDurationMs: number = 5 * 60 * 1000; // 5 minutes default

	const dispatch = createEventDispatcher<{
		send: { blob: Blob; duration: number; waveform: number[] };
		cancel: void;
	}>();

	let isRecording = false;
	let isPaused = false;
	let duration = 0;
	let waveformData: number[] = new Array(50).fill(0.2);
	let mediaRecorder: MediaRecorder | null = null;
	let audioChunks: Blob[] = [];
	let startTime = 0;
	let timerInterval: ReturnType<typeof setInterval> | null = null;
	let audioContext: AudioContext | null = null;
	let analyser: AnalyserNode | null = null;
	let animationFrame: number | null = null;
	let recordedBlob: Blob | null = null;
	let recordedDuration = 0;
	let recordedWaveform: number[] = [];

	// Live waveform data during recording
	let liveWaveform: number[] = new Array(50).fill(0.1);

	async function startRecording() {
		try {
			const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
			
			// Set up audio context for waveform visualization
			audioContext = new AudioContext();
			analyser = audioContext.createAnalyser();
			const source = audioContext.createMediaStreamSource(stream);
			source.connect(analyser);
			analyser.fftSize = 256;
			
			// Create media recorder
			mediaRecorder = new MediaRecorder(stream, {
				mimeType: 'audio/webm;codecs=opus'
			});
			
			audioChunks = [];
			
			mediaRecorder.ondataavailable = (event) => {
				if (event.data.size > 0) {
					audioChunks.push(event.data);
				}
			};
			
			mediaRecorder.onstop = () => {
				const audioBlob = new Blob(audioChunks, { type: 'audio/webm' });
				recordedBlob = audioBlob;
				recordedDuration = duration;
				recordedWaveform = [...waveformData];
				
				// Cleanup
				stream.getTracks().forEach(track => track.stop());
				if (audioContext) {
					audioContext.close();
					audioContext = null;
				}
				if (animationFrame) {
					cancelAnimationFrame(animationFrame);
					animationFrame = null;
				}
			};
			
			mediaRecorder.start(100);
			isRecording = true;
			isPaused = false;
			startTime = Date.now();
			
			// Start timer
			timerInterval = setInterval(() => {
				duration = Date.now() - startTime;
				if (duration >= maxDurationMs) {
					stopRecording();
				}
			}, 100);
			
			// Start waveform visualization
			visualizeWaveform();
			
		} catch (error) {
			console.error('Failed to start recording:', error);
			alert('Could not access microphone. Please check permissions.');
		}
	}

	function visualizeWaveform() {
		if (!analyser || !isRecording) return;
		
		const dataArray = new Uint8Array(analyser.frequencyBinCount);
		analyser.getByteFrequencyData(dataArray);
		
		// Sample the frequency data to get waveform visualization
		const samples = 50;
		const step = Math.floor(dataArray.length / samples);
		liveWaveform = [];
		
		for (let i = 0; i < samples; i++) {
			const value = dataArray[i * step] / 255;
			liveWaveform.push(Math.max(0.1, value));
		}
		
		animationFrame = requestAnimationFrame(visualizeWaveform);
	}

	function pauseRecording() {
		if (mediaRecorder && mediaRecorder.state === 'recording') {
			mediaRecorder.pause();
			isPaused = true;
			if (timerInterval) {
				clearInterval(timerInterval);
				timerInterval = null;
			}
		}
	}

	function resumeRecording() {
		if (mediaRecorder && mediaRecorder.state === 'paused') {
			mediaRecorder.resume();
			isPaused = false;
			startTime = Date.now() - duration;
			timerInterval = setInterval(() => {
				duration = Date.now() - startTime;
				if (duration >= maxDurationMs) {
					stopRecording();
				}
			}, 100);
		}
	}

	function stopRecording() {
		if (mediaRecorder && (mediaRecorder.state === 'recording' || mediaRecorder.state === 'paused')) {
			mediaRecorder.stop();
			isRecording = false;
			isPaused = false;
			if (timerInterval) {
				clearInterval(timerInterval);
				timerInterval = null;
			}
			// Finalize waveform data
			waveformData = [...liveWaveform];
		}
	}

	function cancelRecording() {
		if (mediaRecorder && (mediaRecorder.state === 'recording' || mediaRecorder.state === 'paused')) {
			mediaRecorder.stop();
			audioChunks = [];
		}
		isRecording = false;
		isPaused = false;
		duration = 0;
		recordedBlob = null;
		if (timerInterval) {
			clearInterval(timerInterval);
			timerInterval = null;
		}
		if (audioContext) {
			audioContext.close();
			audioContext = null;
		}
		if (animationFrame) {
			cancelAnimationFrame(animationFrame);
			animationFrame = null;
		}
		dispatch('cancel');
	}

	function sendRecording() {
		if (recordedBlob) {
			dispatch('send', {
				blob: recordedBlob,
				duration: recordedDuration,
				waveform: recordedWaveform
			});
		}
		cancelRecording();
	}

	function formatTime(ms: number): string {
		const seconds = Math.floor(ms / 1000);
		const mins = Math.floor(seconds / 60);
		const secs = seconds % 60;
		return `${mins}:${secs.toString().padStart(2, '0')}`;
	}

	// Auto-start recording when component mounts
	startRecording();

	onDestroy(() => {
		if (isRecording) {
			cancelRecording();
		}
	});
</script>

<div class="voice-recorder-panel">
	{#if isRecording || recordedBlob}
		<div class="recording-area">
			<div class="recording-header">
				<span class="recording-status">
					{#if isRecording}
						<span class="recording-dot"></span>
						Recording
					{:else if recordedBlob}
						Recorded
					{/if}
				</span>
				<span class="recording-time">{formatTime(isRecording ? duration : recordedDuration)}</span>
			</div>
			
			<div class="waveform-area">
				{#if isRecording}
					<div class="live-waveform">
						{#each liveWaveform as amplitude, i}
							<div 
								class="waveform-bar" 
								style="height: {Math.max(4, amplitude * 32)}px"
							></div>
						{/each}
					</div>
				{:else if recordedWaveform.length > 0}
					<div class="recorded-waveform">
						{#each recordedWaveform as amplitude, i}
							<div 
								class="waveform-bar" 
								style="height: {Math.max(4, amplitude * 32)}px"
							></div>
						{/each}
					</div>
				{/if}
			</div>
			
			<div class="recording-actions">
				<button 
					class="action-button cancel"
					on:click={cancelRecording}
					aria-label="Cancel recording"
				>
					<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
						<path fill="currentColor" d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
					</svg>
					<span>Cancel</span>
				</button>
				
				{#if isRecording}
					{#if isPaused}
						<button 
							class="action-button resume"
							on:click={resumeRecording}
							aria-label="Resume recording"
						>
							<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
								<path fill="currentColor" d="M8 5v14l11-7z"/>
							</svg>
							<span>Resume</span>
						</button>
					{:else}
						<button 
							class="action-button pause"
							on:click={pauseRecording}
							aria-label="Pause recording"
						>
							<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
								<rect x="6" y="4" width="4" height="16" fill="currentColor"/>
								<rect x="14" y="4" width="4" height="16" fill="currentColor"/>
							</svg>
							<span>Pause</span>
						</button>
					{/if}
					
					<button 
						class="action-button stop"
						on:click={stopRecording}
						aria-label="Stop recording"
					>
						<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
							<rect x="6" y="6" width="12" height="12" fill="currentColor"/>
						</svg>
						<span>Done</span>
					</button>
				{:else}
					<button 
						class="action-button send"
						on:click={sendRecording}
						aria-label="Send voice message"
					>
						<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
							<path fill="currentColor" d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/>
						</svg>
						<span>Send</span>
					</button>
				{/if}
			</div>
			
			<div class="duration-info">
				Max: {formatTime(maxDurationMs)}
			</div>
		</div>
	{:else}
		<div class="idle-state">
			<div class="idle-icon">
				<svg viewBox="0 0 24 24" width="32" height="32" fill="currentColor" aria-hidden="true">
					<path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"/>
					<path d="M17 11c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z"/>
				</svg>
			</div>
			<p>Initializing microphone...</p>
		</div>
	{/if}
</div>

<style>
	.voice-recorder-panel {
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		padding: 16px;
		margin: 8px 0;
	}

	.recording-area {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.recording-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.recording-status {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 14px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
	}

	.recording-dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background: var(--status-danger, #f23f43);
		animation: pulse 1s ease-in-out infinite;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}

	.recording-time {
		font-size: 16px;
		font-weight: 600;
		color: var(--text-primary, #f2f3f5);
		font-variant-numeric: tabular-nums;
	}

	.waveform-area {
		height: 48px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--bg-tertiary, #1e1f22);
		border-radius: 4px;
		padding: 0 8px;
	}

	.live-waveform,
	.recorded-waveform {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 2px;
		height: 100%;
		width: 100%;
	}

	.waveform-bar {
		width: 3px;
		background: var(--brand-primary, #5865f2);
		border-radius: 1px;
		transition: height 0.05s;
	}

	.recorded-waveform .waveform-bar {
		background: var(--brand-primary, #5865f2);
	}

	.recording-actions {
		display: flex;
		justify-content: center;
		gap: 12px;
	}

	.action-button {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 16px;
		border-radius: 20px;
		border: none;
		cursor: pointer;
		font-size: 14px;
		font-weight: 500;
		transition: transform 0.1s, background-color 0.15s;
	}

	.action-button:active {
		transform: scale(0.95);
	}

	.action-button.cancel {
		background: var(--bg-tertiary, #1e1f22);
		color: var(--text-muted, #949ba4);
	}

	.action-button.cancel:hover {
		background: var(--status-danger, #f23f43);
		color: white;
	}

	.action-button.pause {
		background: var(--bg-tertiary, #1e1f22);
		color: var(--text-muted, #949ba4);
	}

	.action-button.pause:hover {
		background: var(--bg-modifier-hover, rgba(79, 84, 92, 0.32));
		color: var(--text-primary, #f2f3f5);
	}

	.action-button.resume {
		background: var(--brand-primary, #5865f2);
		color: white;
	}

	.action-button.resume:hover {
		background: var(--brand-hover, #4752c4);
	}

	.action-button.stop {
		background: var(--brand-primary, #5865f2);
		color: white;
	}

	.action-button.stop:hover {
		background: var(--brand-hover, #4752c4);
	}

	.action-button.send {
		background: var(--status-success, #23a559);
		color: white;
	}

	.action-button.send:hover {
		background: #2d8a4e;
	}

	.duration-info {
		text-align: center;
		font-size: 12px;
		color: var(--text-muted, #949ba4);
	}

	.idle-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 16px;
	}

	.idle-icon {
		color: var(--text-muted, #949ba4);
	}

	.idle-state p {
		font-size: 14px;
		color: var(--text-muted, #949ba4);
		margin: 0;
	}
</style>
