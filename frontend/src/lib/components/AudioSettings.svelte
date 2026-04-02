<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { audioSettings, currentServerAudioSettings } from '$lib/stores/audioSettings';

  export let serverId: string;

  let audioInputDevices: MediaDeviceInfo[] = [];
  let audioOutputDevices: MediaDeviceInfo[] = [];

  let selectedInputDevice = '';
  let selectedOutputDevice = '';
  let inputVolume = 100;
  let outputVolume = 100;
  let pushToTalkEnabled = false;
  let pushToTalkKey = '';
  let recordingKey = false;
  let saving = false;
  let loaded = false;

  // Mic test
  let micTestActive = false;
  let micLevel = 0;
  let micTestCleanup: (() => void) | null = null;

  async function loadDevices() {
    try {
      await navigator.mediaDevices.getUserMedia({ audio: true }).then(stream => {
        stream.getTracks().forEach(track => track.stop());
      }).catch(err => {
        // User denied permission - that's fine, devices will be empty
        console.debug('Audio permission not granted:', err);
      });

      const devices = await navigator.mediaDevices.enumerateDevices();
      audioInputDevices = devices.filter(d => d.kind === 'audioinput');
      audioOutputDevices = devices.filter(d => d.kind === 'audiooutput');
    } catch (err) {
      console.error('Failed to enumerate devices:', err);
    }
  }

  async function loadSettings() {
    const data = await audioSettings.loadForServer(serverId);
    if (data) {
      selectedInputDevice = data.input_device_id;
      selectedOutputDevice = data.output_device_id;
      inputVolume = data.input_volume;
      outputVolume = data.output_volume;
      pushToTalkEnabled = data.push_to_talk_enabled;
      pushToTalkKey = data.push_to_talk_key;
    }
    loaded = true;
  }

  async function saveSettings() {
    saving = true;
    await audioSettings.updateForServer(serverId, {
      input_device_id: selectedInputDevice,
      output_device_id: selectedOutputDevice,
      input_volume: inputVolume,
      output_volume: outputVolume,
      push_to_talk_enabled: pushToTalkEnabled,
      push_to_talk_key: pushToTalkKey,
    });
    saving = false;
  }

  // Debounced save on any change
  let saveTimeout: ReturnType<typeof setTimeout> | null = null;
  function scheduleSave() {
    if (!loaded) return;
    if (saveTimeout) clearTimeout(saveTimeout);
    saveTimeout = setTimeout(saveSettings, 500);
  }

  $: if (loaded) {
    // Trigger save on any change
    void selectedInputDevice;
    void selectedOutputDevice;
    void inputVolume;
    void outputVolume;
    void pushToTalkEnabled;
    void pushToTalkKey;
    scheduleSave();
  }

  function startRecordingKey() {
    recordingKey = true;
  }

  function handleKeyCapture(e: KeyboardEvent) {
    if (!recordingKey) return;
    e.preventDefault();
    e.stopPropagation();
    pushToTalkKey = e.code;
    recordingKey = false;
  }

  function formatKeyCode(code: string): string {
    if (!code) return 'Not set';
    return code
      .replace('Key', '')
      .replace('Digit', '')
      .replace('Arrow', '')
      .replace('Left', 'Left ')
      .replace('Right', 'Right ')
      .replace('Numpad', 'Numpad ');
  }

  async function startMicTest() {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: selectedInputDevice ? { deviceId: selectedInputDevice } : true
      });
      const audioContext = new AudioContext();
      const source = audioContext.createMediaStreamSource(stream);
      const analyser = audioContext.createAnalyser();
      analyser.fftSize = 256;
      source.connect(analyser);

      const dataArray = new Uint8Array(analyser.frequencyBinCount);
      micTestActive = true;

      const interval = setInterval(() => {
        analyser.getByteFrequencyData(dataArray);
        const avg = dataArray.reduce((a, b) => a + b, 0) / dataArray.length;
        micLevel = Math.min(100, Math.round((avg / 128) * 100));
      }, 50);

      micTestCleanup = () => {
        clearInterval(interval);
        stream.getTracks().forEach(t => t.stop());
        audioContext.close();
        micTestActive = false;
        micLevel = 0;
      };
    } catch (err) {
      console.error('Failed to start mic test:', err);
    }
  }

  function stopMicTest() {
    if (micTestCleanup) {
      micTestCleanup();
      micTestCleanup = null;
    }
  }

  onMount(async () => {
    await loadDevices();
    await loadSettings();
  });

  onDestroy(() => {
    stopMicTest();
    if (saveTimeout) clearTimeout(saveTimeout);
  });
</script>

<svelte:window on:keydown={handleKeyCapture} />

<div class="space-y-6">
  <div class="bg-[var(--bg-secondary)] rounded-lg p-4">
    <h2 class="text-xs font-bold uppercase text-[var(--text-muted)] mb-4">Input Settings</h2>

    <div class="mb-4">
      <label class="block text-sm text-[var(--text-secondary)] mb-2">Input Device</label>
      <select
        bind:value={selectedInputDevice}
        class="w-full p-2.5 bg-[var(--bg-tertiary)] border border-[var(--bg-modifier-accent)] rounded text-[var(--text-primary)] text-sm"
      >
        <option value="">Default</option>
        {#each audioInputDevices as device}
          <option value={device.deviceId}>{device.label || 'Microphone'}</option>
        {/each}
      </select>
    </div>

    <div class="mb-4">
      <label class="block text-sm text-[var(--text-secondary)] mb-2">Input Volume</label>
      <input
        type="range"
        min="0"
        max="100"
        bind:value={inputVolume}
        class="w-full accent-[var(--brand-primary)]"
      />
      <span class="text-xs text-[var(--text-muted)]">{inputVolume}%</span>
    </div>

    <div class="mb-4">
      <button
        on:click={() => micTestActive ? stopMicTest() : startMicTest()}
        class="px-4 py-2 bg-[var(--brand-primary)] text-white rounded text-sm hover:bg-[var(--brand-primary-dark)] transition-colors"
      >
        {micTestActive ? 'Stop Mic Test' : 'Test Microphone'}
      </button>
      {#if micTestActive}
        <div class="mt-2 h-2 bg-[var(--bg-tertiary)] rounded overflow-hidden">
          <div
            class="h-full bg-[var(--text-positive)] transition-all duration-75"
            style="width: {micLevel}%"
          ></div>
        </div>
      {/if}
    </div>
  </div>

  <div class="bg-[var(--bg-secondary)] rounded-lg p-4">
    <h2 class="text-xs font-bold uppercase text-[var(--text-muted)] mb-4">Output Settings</h2>

    <div class="mb-4">
      <label class="block text-sm text-[var(--text-secondary)] mb-2">Output Device</label>
      <select
        bind:value={selectedOutputDevice}
        class="w-full p-2.5 bg-[var(--bg-tertiary)] border border-[var(--bg-modifier-accent)] rounded text-[var(--text-primary)] text-sm"
      >
        <option value="">Default</option>
        {#each audioOutputDevices as device}
          <option value={device.deviceId}>{device.label || 'Speaker'}</option>
        {/each}
      </select>
    </div>

    <div class="mb-4">
      <label class="block text-sm text-[var(--text-secondary)] mb-2">Output Volume</label>
      <input
        type="range"
        min="0"
        max="100"
        bind:value={outputVolume}
        class="w-full accent-[var(--brand-primary)]"
      />
      <span class="text-xs text-[var(--text-muted)]">{outputVolume}%</span>
    </div>
  </div>

  <div class="bg-[var(--bg-secondary)] rounded-lg p-4">
    <h2 class="text-xs font-bold uppercase text-[var(--text-muted)] mb-4">Push to Talk</h2>

    <div class="flex items-center justify-between mb-4">
      <div>
        <label class="block text-sm text-[var(--text-primary)]">Enable Push to Talk</label>
        <p class="text-xs text-[var(--text-muted)]">Hold a key to transmit audio instead of always-on</p>
      </div>
      <label class="relative inline-flex cursor-pointer w-10 h-6">
        <input
          type="checkbox"
          bind:checked={pushToTalkEnabled}
          class="opacity-0 w-0 h-0"
        />
        <span class="absolute cursor-pointer inset-0 bg-[var(--bg-modifier-accent)] rounded-full transition-colors before:content-[''] before:absolute before:h-[18px] before:w-[18px] before:left-[3px] before:bottom-[3px] before:bg-white before:rounded-full before:transition-transform [&:has(input:checked)]:bg-[var(--brand-primary)] [&:has(input:checked)]:before:translate-x-4"></span>
      </label>
    </div>

    {#if pushToTalkEnabled}
      <div class="mb-4">
        <label class="block text-sm text-[var(--text-secondary)] mb-2">Keybind</label>
        <div class="flex items-center gap-3">
          <div class="flex-1 p-2.5 bg-[var(--bg-tertiary)] border border-[var(--bg-modifier-accent)] rounded text-[var(--text-primary)] text-sm">
            {#if recordingKey}
              <span class="text-[var(--brand-primary)] animate-pulse">Press any key...</span>
            {:else}
              {formatKeyCode(pushToTalkKey)}
            {/if}
          </div>
          <button
            on:click={startRecordingKey}
            class="px-4 py-2 bg-[var(--bg-modifier-accent)] text-[var(--text-primary)] rounded text-sm hover:bg-[var(--bg-modifier-selected)] transition-colors"
          >
            {recordingKey ? 'Listening...' : 'Record Keybind'}
          </button>
        </div>
      </div>
    {/if}
  </div>

  {#if saving}
    <p class="text-xs text-[var(--text-muted)] text-right">Saving...</p>
  {/if}
</div>
