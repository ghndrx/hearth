import { writable } from 'svelte/store';
import { gateway } from './gateway';

export type PresenceStatus = 'online' | 'idle' | 'dnd' | 'invisible' | 'offline';

export interface CustomStatusInfo {
  custom_text?: string | null;
  emoji?: string | null;
  emoji_id?: string | null;
  emoji_name?: string | null;
  clear_after?: string | null;
}

export interface Presence {
  userId: string;
  status: PresenceStatus;
  customStatus?: string;
  customStatusInfo?: CustomStatusInfo;
  activities: Activity[];
  clientStatus?: {
    desktop?: PresenceStatus;
    mobile?: PresenceStatus;
    web?: PresenceStatus;
  };
}

export interface Activity {
  name: string;
  type: number; // 0=Playing, 1=Streaming, 2=Listening, 3=Watching, 4=Custom, 5=Competing
  url?: string;
  details?: string;
  state?: string;
  emoji?: string; // Custom status emoji
  timestamps?: {
    start?: number;
    end?: number;
  };
  assets?: {
    large_image?: string;
    large_text?: string;
    small_image?: string;
    small_text?: string;
  };
}

function createPresenceStore() {
  const presences = writable<Map<string, Presence>>(new Map());

  // Subscribe to gateway presence updates
  gateway.on('PRESENCE_UPDATE', (data) => {
    const update = data as {
      user: { id: string };
      status: PresenceStatus;
      activities: Activity[];
      client_status?: {
        desktop?: PresenceStatus;
        mobile?: PresenceStatus;
        web?: PresenceStatus;
      };
      custom_status?: CustomStatusInfo;
    };

    presences.update(map => {
      const existing = map.get(update.user.id);
      map.set(update.user.id, {
        userId: update.user.id,
        status: update.status,
        activities: update.activities || [],
        clientStatus: update.client_status,
        customStatusInfo: update.custom_status || existing?.customStatusInfo,
      });
      return new Map(map);
    });
  });

  // Subscribe to user status updates (custom status changes)
  gateway.on('USER_STATUS_UPDATE', (data) => {
    const update = data as {
      user_id: string;
      custom_text?: string | null;
      emoji?: string | null;
      emoji_id?: string | null;
      emoji_name?: string | null;
      clear_after?: string | null;
    };

    presences.update(map => {
      const existing = map.get(update.user_id);
      if (existing) {
        existing.customStatusInfo = {
          custom_text: update.custom_text,
          emoji: update.emoji,
          emoji_id: update.emoji_id,
          emoji_name: update.emoji_name,
          clear_after: update.clear_after,
        };
        map.set(update.user_id, { ...existing });
      }
      return new Map(map);
    });
  });

  // Handle READY event with initial presences
  gateway.on('READY', (data) => {
    const ready = data as {
      presences?: Array<{
        user: { id: string };
        status: PresenceStatus;
        activities: Activity[];
        client_status?: object;
      }>;
    };

    if (ready.presences) {
      presences.update(map => {
        for (const p of ready.presences!) {
          map.set(p.user.id, {
            userId: p.user.id,
            status: p.status,
            activities: p.activities || [],
            clientStatus: p.client_status as Presence['clientStatus'],
          });
        }
        return new Map(map);
      });
    }
  });

  function getPresence(userId: string): Presence {
    let result: Presence = {
      userId,
      status: 'offline',
      activities: [],
    };

    presences.subscribe(map => {
      result = map.get(userId) || result;
    })();

    return result;
  }

  function setStatus(status: PresenceStatus) {
    gateway.updatePresence(status, status === 'idle');
  }

  return {
    subscribe: presences.subscribe,
    getPresence,
    setStatus,
  };
}

export const presenceStore = createPresenceStore();

// Helper to get status color
export function getStatusColor(status: PresenceStatus): string {
  switch (status) {
    case 'online':
      return '#3ba55c';
    case 'idle':
      return '#faa61a';
    case 'dnd':
      return '#ed4245';
    case 'invisible':
    case 'offline':
    default:
      return '#747f8d';
  }
}

// Helper to get status label
export function getStatusLabel(status: PresenceStatus): string {
  switch (status) {
    case 'online':
      return 'Online';
    case 'idle':
      return 'Idle';
    case 'dnd':
      return 'Do Not Disturb';
    case 'invisible':
      return 'Invisible';
    case 'offline':
    default:
      return 'Offline';
  }
}

// Activity type labels
export function getActivityLabel(type: number): string {
  switch (type) {
    case 0:
      return 'Playing';
    case 1:
      return 'Streaming';
    case 2:
      return 'Listening to';
    case 3:
      return 'Watching';
    case 4:
      return '';
    case 5:
      return 'Competing in';
    default:
      return '';
  }
}
