<script lang="ts">
	import { createEventDispatcher, onDestroy } from 'svelte';

	export let maxDurationMs: number = 5 * 60 * 1000; // 5 minutes default

	const dispatch = createEventDispatcher<{
		recorded: { blob: Blob; duration: number; waveform: number[] };
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

	// Live waveform data during recording
	let liveWaveform: number[] = new Array(50).fill(0);

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
				dispatch('recorded', {
					blob: audioBlob,
					duration: duration,
					waveform: waveformData
				});
				
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
			
			mediaRecorder.start(100); // Collect data every 100ms
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
			// Discard the recorded audio
			audioChunks = [];
		}
		isRecording = false;
		isPaused = false;
		duration = 0;
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

	function formatTime(ms: number): string {
		const seconds = Math.floor(ms / 1000);
		const mins = Math.floor(seconds / 60);
		const secs = seconds % 60;
		return `${mins}:${secs.toString().padStart(2, '0')}`;
	}

	onDestroy(() => {
		if (isRecording) {
			cancelRecording();
		}
	});
</script>

<div class="voice-recorder" class:recording={isRecording}>
	{#if !isRecording}
		<button 
			class="start-button"
			on:click={startRecording}
			aria-label="Start recording"
		>
			<svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor" aria-hidden="true">
				<path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"/>
				<path d="M17 11c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z"/>
			</svg>
			<span>Hold to record</span>
		</button>
	{:else}
		<div class="recording-ui">
			<div class="recording-indicator">
				<span class="recording-dot"></span>
				<span class="recording-time">{formatTime(duration)}</span>
			</div>
			
			<div class="live-waveform">
				{#each liveWaveform as amplitude, i}
					<div 
						class="waveform-bar" 
						style="height: {Math.max(4, amplitude * 32)}px"
					></div>
				{/each}
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
				</button>
				
				{#if isPaused}
					<button 
						class="action-button resume"
						on:click={resumeRecording}
						aria-label="Resume recording"
					>
						<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
							<path fill="currentColor" d="M8 5v14l11-7z"/>
						</svg>
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
					</button>
				{/if}
				
				<button 
					class="action-button stop"
					on:click={stopRecording}
					aria-label="Stop and send recording"
				>
					<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
						<path fill="currentColor" d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"/>
					</svg>
				</button>
			</div>
			
			<div class="duration-limit">
				Max: {formatTime(maxDurationMs)}
			</div>
		</div>
	{/if}
</div>

<style>
	.voice-recorder {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 8px;
	}

	.start-button {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		background: var(--brand-primary, #5865f2);
		color: white;
		border: none;
		border-radius: 20px;
		cursor: pointer;
		font-size: 14px;
		font-weight: 500;
		transition: background-color 0.15s, transform 0.1s;
	}

	.start-button:hover {
		background: var(--brand-hover, #4752c4);
	}

	.start-button:active {
		transform: scale(0.95);
	}

	.recording-ui {
		display: flex;
		align-items: center;
		gap: 16px;
		width: 100%;
	}

	.recording-indicator {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.recording-dot {
		width: 12px;
		height: 12px;
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
		min-width: 45px;
	}

	.live-waveform {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 2px;
		height: 32px;
	}

	.waveform-bar {
		width: 3px;
		background: var(--brand-primary, #5865f2);
		border-radius: 1px;
		transition: height 0.05s;
	}

	.recording-actions {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.action-button {
		width: 36px;
		height: 36px;
		border-radius: 50%;
		border: none;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: transform 0.1s, background-color 0.15s;
	}

	.action-button:active {
		transform: scale(0.95);
	}

	.action-button.cancel {
		background: var(--bg-secondary, #2b2d31);
		color: var(--text-muted, #949ba4);
	}

	.action-button.cancel:hover {
		background: var(--status-danger, #f23f43);
		color: white;
	}

	.action-button.pause {
		background: var(--bg-secondary, #2b2d31);
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
		background: var(--status-success, #23a559);
		color: white;
	}

	.action-button.stop:hover {
		background: #2d8a4e;
	}

	.duration-limit {
		font-size: 12px;
		color: var(--text-muted, #949ba4);
	}
</style>
