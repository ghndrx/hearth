<script lang="ts">
	import { sendGameMove } from '$lib/stores/voiceActivity';
	import { user as authUser } from '$lib/stores/auth';
	import type { VoiceActivityParticipant, PokerState } from '$lib/stores/voiceActivity';

	export let activityId: string;
	export let state: unknown;
	export let participants: VoiceActivityParticipant[];

	$: pokerState = state as PokerState | null;
	$: myPlayer = pokerState?.players?.find(p => p.user_id === $authUser?.id);
	$: isMyTurn = pokerState?.current_turn === $authUser?.id;
	$: canStart = pokerState?.phase === 'waiting' && (pokerState?.players?.length ?? 0) >= 2;

	let betAmount = 0;

	async function handleAction(action: string, data?: Record<string, unknown>) {
		await sendGameMove(activityId, action, data);
	}

	function getPlayerName(userId: string): string {
		const p = participants.find(p => p.user_id === userId);
		return p?.display_name || p?.username || 'Unknown';
	}

	const suitSymbols: Record<string, string> = { 'H': '♥', 'D': '♦', 'C': '♣', 'S': '♠' };
	function formatCard(card: string): { value: string; suit: string; color: string } {
		if (!card || card.length < 2) return { value: '?', suit: '?', color: 'white' };
		const value = card.slice(0, -1);
		const suitKey = card.slice(-1);
		const suit = suitSymbols[suitKey] || suitKey;
		const color = (suitKey === 'H' || suitKey === 'D') ? '#ed4245' : 'white';
		return { value, suit, color };
	}
</script>

<div class="poker-game">
	{#if pokerState}
		<div class="poker-table">
			<div class="table-info">
				<span class="phase-badge">{pokerState.phase.toUpperCase()}</span>
				<span class="pot-display">Pot: {pokerState.pot}</span>
				<span class="blinds">Blinds: {pokerState.small_blind}/{pokerState.big_blind}</span>
			</div>

			<!-- Community Cards -->
			<div class="community-cards">
				{#each pokerState.community_cards as card}
					{@const c = formatCard(card)}
					<div class="card" style="color: {c.color}">
						<span class="card-value">{c.value}</span>
						<span class="card-suit">{c.suit}</span>
					</div>
				{:else}
					<div class="cards-placeholder">
						{#if pokerState.phase === 'waiting'}Waiting for players...
						{:else}Cards will appear here{/if}
					</div>
				{/each}
			</div>

			<!-- Players around the table -->
			<div class="players-ring">
				{#each pokerState.players as player, i}
					<div class="player-seat" class:active={pokerState.current_turn === player.user_id} class:folded={player.folded}>
						<div class="player-name">{getPlayerName(player.user_id)}</div>
						<div class="player-chips">{player.chips} chips</div>
						{#if player.bet > 0}
							<div class="player-bet">Bet: {player.bet}</div>
						{/if}
						{#if player.folded}
							<div class="player-status folded-label">Folded</div>
						{:else if player.all_in}
							<div class="player-status allin-label">ALL IN</div>
						{/if}
						{#if player.is_dealer}
							<div class="dealer-chip">D</div>
						{/if}
					</div>
				{/each}
			</div>
		</div>

		<!-- My Hand -->
		{#if myPlayer}
			<div class="my-hand">
				<h4>Your Hand</h4>
				<div class="hand-cards">
					{#each myPlayer.hand as card}
						{@const c = formatCard(card)}
						<div class="card my-card" style="color: {c.color}">
							<span class="card-value">{c.value}</span>
							<span class="card-suit">{c.suit}</span>
						</div>
					{:else}
						<span class="no-cards">No cards dealt yet</span>
					{/each}
				</div>
				<div class="my-info">
					<span>Chips: {myPlayer.chips}</span>
				</div>
			</div>
		{/if}

		<!-- Actions -->
		<div class="poker-actions">
			{#if canStart}
				<button class="btn-action btn-start" on:click={() => handleAction('start')}>Start Game</button>
			{:else if isMyTurn && !myPlayer?.folded}
				<button class="btn-action" on:click={() => handleAction('check')}>Check</button>
				<button class="btn-action" on:click={() => handleAction('call')}>Call</button>
				<div class="bet-group">
					<input type="number" bind:value={betAmount} min={pokerState.big_blind} max={myPlayer?.chips ?? 0} class="bet-input" />
					<button class="btn-action btn-bet" on:click={() => handleAction('bet', { amount: betAmount })}>Bet</button>
				</div>
				<button class="btn-action btn-fold" on:click={() => handleAction('fold')}>Fold</button>
			{:else}
				<div class="waiting-text">
					{#if myPlayer?.folded}You folded this round
					{:else}Waiting for other players...{/if}
				</div>
			{/if}
		</div>
	{:else}
		<div class="loading-state">Loading poker game...</div>
	{/if}
</div>

<style>
	.poker-game {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.poker-table {
		background: #1a472a;
		border: 3px solid #2d6b3f;
		border-radius: 120px;
		padding: 32px 24px;
		min-height: 280px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
	}

	.table-info {
		display: flex;
		gap: 16px;
		align-items: center;
	}

	.phase-badge {
		background: rgba(255, 255, 255, 0.15);
		color: #ffd700;
		padding: 2px 10px;
		border-radius: 10px;
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.05em;
	}

	.pot-display {
		color: #ffd700;
		font-size: 18px;
		font-weight: 700;
	}

	.blinds {
		color: rgba(255, 255, 255, 0.6);
		font-size: 12px;
	}

	.community-cards {
		display: flex;
		gap: 8px;
		min-height: 80px;
		align-items: center;
	}

	.cards-placeholder {
		color: rgba(255, 255, 255, 0.4);
		font-size: 13px;
	}

	.card {
		width: 50px;
		height: 72px;
		background: white;
		border-radius: 6px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		font-weight: 700;
		box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
	}

	.card-value { font-size: 16px; }
	.card-suit { font-size: 20px; }

	.my-card {
		width: 60px;
		height: 84px;
		border: 2px solid #5865f2;
	}

	.players-ring {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
		justify-content: center;
	}

	.player-seat {
		background: rgba(0, 0, 0, 0.3);
		border-radius: 8px;
		padding: 8px 12px;
		min-width: 90px;
		text-align: center;
		border: 2px solid transparent;
		transition: border-color 0.2s ease;
	}

	.player-seat.active {
		border-color: #ffd700;
	}

	.player-seat.folded {
		opacity: 0.5;
	}

	.player-name { color: white; font-size: 12px; font-weight: 600; }
	.player-chips { color: #ffd700; font-size: 11px; }
	.player-bet { color: #43b581; font-size: 11px; }
	.player-status { font-size: 10px; font-weight: 700; }
	.folded-label { color: #ed4245; }
	.allin-label { color: #ffd700; }

	.dealer-chip {
		display: inline-block;
		width: 18px;
		height: 18px;
		background: white;
		color: #1a472a;
		border-radius: 50%;
		font-size: 10px;
		font-weight: 800;
		line-height: 18px;
		text-align: center;
		margin-top: 2px;
	}

	.my-hand {
		background: var(--bg-secondary, #2b2d31);
		border-radius: 8px;
		padding: 12px 16px;
	}

	.my-hand h4 {
		font-size: 12px;
		text-transform: uppercase;
		color: var(--text-muted, #6d6f78);
		margin-bottom: 8px;
	}

	.hand-cards {
		display: flex;
		gap: 8px;
		margin-bottom: 8px;
	}

	.no-cards { color: var(--text-muted, #6d6f78); font-size: 12px; }
	.my-info { font-size: 12px; color: var(--text-secondary, #b5bac1); }

	.poker-actions {
		display: flex;
		gap: 8px;
		align-items: center;
		flex-wrap: wrap;
	}

	.btn-action {
		padding: 8px 16px;
		border: none;
		border-radius: 4px;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
		background: var(--brand-primary, #5865f2);
		color: white;
		transition: background 0.15s ease;
	}

	.btn-action:hover { background: #4752c4; }
	.btn-start { background: #43b581; }
	.btn-start:hover { background: #3ba374; }
	.btn-fold { background: #ed4245; }
	.btn-fold:hover { background: #c93b3e; }
	.btn-bet { background: #faa61a; color: #1e1f22; }

	.bet-group {
		display: flex;
		gap: 4px;
		align-items: center;
	}

	.bet-input {
		width: 80px;
		padding: 6px 8px;
		border-radius: 4px;
		border: 1px solid var(--border-subtle, #1e1f22);
		background: var(--bg-tertiary, #1e1f22);
		color: var(--text-primary, #f2f3f5);
		font-size: 13px;
	}

	.waiting-text {
		color: var(--text-muted, #6d6f78);
		font-size: 13px;
		font-style: italic;
	}

	.loading-state {
		text-align: center;
		color: var(--text-muted, #6d6f78);
		padding: 40px;
	}
</style>
