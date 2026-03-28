import { writable } from 'svelte/store';
import { api } from '$lib/api';

export interface Webhook {
	id: string;
	name: string;
	channel_id: string;
	guild_id: string;
	token?: string;
	avatar?: string | null;
	type: number;
}

export const webhooks = writable<Webhook[]>([]);
export const webhooksLoading = writable(false);
export const webhooksError = writable<string | null>(null);

export async function loadChannelWebhooks(channelId: string): Promise<Webhook[]> {
	webhooksLoading.set(true);
	webhooksError.set(null);
	try {
		const data = await api.get<Webhook[]>(`/channels/${channelId}/webhooks`);
		webhooks.set(data);
		return data;
	} catch (err) {
		const message = err instanceof Error ? err.message : 'Failed to load webhooks';
		webhooksError.set(message);
		return [];
	} finally {
		webhooksLoading.set(false);
	}
}

export async function createWebhook(channelId: string, name: string, avatar?: string): Promise<Webhook | null> {
	try {
		const webhook = await api.post<Webhook>(`/channels/${channelId}/webhooks`, { name, avatar });
		webhooks.update(w => [...w, webhook]);
		return webhook;
	} catch (err) {
		const message = err instanceof Error ? err.message : 'Failed to create webhook';
		webhooksError.set(message);
		return null;
	}
}

export async function updateWebhook(webhookId: string, updates: { name?: string; avatar?: string; channel_id?: string }): Promise<Webhook | null> {
	try {
		const webhook = await api.patch<Webhook>(`/webhooks/${webhookId}`, updates);
		webhooks.update(w => w.map(wh => wh.id === webhookId ? webhook : wh));
		return webhook;
	} catch (err) {
		const message = err instanceof Error ? err.message : 'Failed to update webhook';
		webhooksError.set(message);
		return null;
	}
}

export async function deleteWebhook(webhookId: string): Promise<boolean> {
	try {
		await api.delete(`/webhooks/${webhookId}`);
		webhooks.update(w => w.filter(wh => wh.id !== webhookId));
		return true;
	} catch (err) {
		const message = err instanceof Error ? err.message : 'Failed to delete webhook';
		webhooksError.set(message);
		return false;
	}
}
