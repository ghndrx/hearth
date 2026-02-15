import { writable, derived } from 'svelte/store';

export interface PopoutUser {
  id: string;
  username: string;
  display_name: string | null;
  avatar: string | null;
  banner: string | null;
  bio: string | null;
  pronouns: string | null;
  bot: boolean;
  created_at: string;
}

export interface PopoutMember {
  nickname: string | null;
  joined_at: string;
  roles: {
    id: string;
    name: string;
    color: string;
  }[];
}

export interface PopoutState {
  isOpen: boolean;
  user: PopoutUser | null;
  member: PopoutMember | null;
  position: { x: number; y: number } | null;
  anchor: 'left' | 'right';
  mutualServers: { id: string; name: string; icon: string | null }[];
  mutualFriends: { id: string; username: string; avatar: string | null }[];
}

const initialState: PopoutState = {
  isOpen: false,
  user: null,
  member: null,
  position: null,
  anchor: 'right',
  mutualServers: [],
  mutualFriends: [],
};

function createPopoutStore() {
  const { subscribe, set, update } = writable<PopoutState>(initialState);

  return {
    subscribe,

    /**
     * Open the user popout at a specific position
     */
    open(options: {
      user: PopoutUser;
      member?: PopoutMember | null;
      position?: { x: number; y: number };
      anchor?: 'left' | 'right';
      mutualServers?: { id: string; name: string; icon: string | null }[];
      mutualFriends?: { id: string; username: string; avatar: string | null }[];
    }) {
      set({
        isOpen: true,
        user: options.user,
        member: options.member ?? null,
        position: options.position ?? null,
        anchor: options.anchor ?? 'right',
        mutualServers: options.mutualServers ?? [],
        mutualFriends: options.mutualFriends ?? [],
      });
    },

    /**
     * Close the user popout
     */
    close() {
      update((state) => ({
        ...state,
        isOpen: false,
      }));
    },

    /**
     * Update popout data (e.g., after fetching more info)
     */
    updateData(data: Partial<PopoutState>) {
      update((state) => ({
        ...state,
        ...data,
      }));
    },

    /**
     * Reset to initial state
     */
    reset() {
      set(initialState);
    },
  };
}

export const popoutStore = createPopoutStore();

// Derived store for checking if popout is open
export const isPopoutOpen = derived(popoutStore, ($popout) => $popout.isOpen);
