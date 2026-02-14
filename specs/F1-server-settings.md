# F1: ServerSettings Modal

## Goal
Server owners can edit server name, icon, and delete server.

## Input Files (READ THESE FIRST)
- `frontend/src/lib/components/Modal.svelte` - Wrapper pattern
- `frontend/src/lib/components/Button.svelte` - Button component
- `frontend/src/lib/components/ConfirmDialog.svelte` - Confirm pattern
- `frontend/src/lib/stores/servers.ts` - Server store methods

## Output Files
- `frontend/src/lib/components/ServerSettings.svelte` - NEW FILE

## Component Specification

### Props
```typescript
export let server: { id: string; name: string; icon?: string };
export let open = false;
```

### Events
```typescript
dispatch('close')
```

### Features
1. Server name input (text field)
2. Server icon upload with preview
3. Delete server button → opens ConfirmDialog
4. Save button → calls updateServer()
5. Cancel button → dispatches close

### UI Layout
```
┌─────────────────────────────┐
│ Server Settings          X │
├─────────────────────────────┤
│ [Icon Preview] [Upload]     │
│                             │
│ Server Name                 │
│ [___________________]       │
│                             │
│ [Delete Server]    [Cancel] │
│                     [Save]  │
└─────────────────────────────┘
```

### Store Methods to Use
```typescript
import { updateServer, deleteServer } from '$lib/stores/servers';
```

## Complete Code Template
```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import Modal from './Modal.svelte';
  import Button from './Button.svelte';
  import ConfirmDialog from './ConfirmDialog.svelte';
  import { updateServer, deleteServer } from '$lib/stores/servers';

  export let server: { id: string; name: string; icon?: string };
  export let open = false;

  const dispatch = createEventDispatcher<{ close: void }>();

  let name = server.name;
  let iconPreview = server.icon || '';
  let showDeleteConfirm = false;
  let saving = false;

  function handleIconChange(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files?.[0]) {
      iconPreview = URL.createObjectURL(input.files[0]);
    }
  }

  async function handleSave() {
    saving = true;
    await updateServer(server.id, { name, icon: iconPreview });
    saving = false;
    dispatch('close');
  }

  async function handleDelete() {
    await deleteServer(server.id);
    dispatch('close');
  }
</script>

<Modal {open} title="Server Settings" on:close>
  <div class="space-y-4">
    <div class="flex items-center gap-4">
      <img src={iconPreview || '/default-server.png'} alt="Server icon" class="w-16 h-16 rounded-full" />
      <input type="file" accept="image/*" on:change={handleIconChange} />
    </div>

    <label class="block">
      <span class="text-sm text-gray-400">Server Name</span>
      <input type="text" bind:value={name} class="w-full mt-1 px-3 py-2 bg-[#1e1f22] rounded" />
    </label>

    <div class="flex justify-between pt-4">
      <Button variant="danger" on:click={() => (showDeleteConfirm = true)}>Delete Server</Button>
      <div class="flex gap-2">
        <Button variant="secondary" on:click={() => dispatch('close')}>Cancel</Button>
        <Button on:click={handleSave} disabled={saving}>{saving ? 'Saving...' : 'Save'}</Button>
      </div>
    </div>
  </div>
</Modal>

<ConfirmDialog
  open={showDeleteConfirm}
  title="Delete Server"
  message="This cannot be undone."
  on:confirm={handleDelete}
  on:cancel={() => (showDeleteConfirm = false)}
/>
```

## Acceptance Criteria
- [ ] `npm run check` passes
- [ ] Uses existing Modal, Button, ConfirmDialog components
- [ ] Dispatches 'close' event properly
- [ ] Calls store methods correctly

## Verification Command
```bash
cd frontend && npm run check && echo "✅ Svelte check passes"
```

## Commit Message
```
feat: add ServerSettings modal component
```
