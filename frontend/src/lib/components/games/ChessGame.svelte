<script lang="ts">
	import { sendGameMove } from '$lib/stores/voiceActivity';
	import { user as authUser } from '$lib/stores/auth';
	import type { VoiceActivityParticipant, ChessState } from '$lib/stores/voiceActivity';

	export let activityId: string;
	export let state: unknown;
	export let participants: VoiceActivityParticipant[];

	$: chessState = state as ChessState | null;
	$: board = chessState ? parseFEN(chessState.board) : [];
	$: isWhite = chessState?.white_player === $authUser?.id;
	$: isBlack = chessState?.black_player === $authUser?.id;
	$: isMyTurn = (chessState?.current_turn === 'white' && isWhite) || (chessState?.current_turn === 'black' && isBlack);
	$: canJoin = !isWhite && !isBlack && chessState?.status === 'waiting';
	$: gameOver = chessState?.status === 'checkmate' || chessState?.status === 'stalemate' || chessState?.status === 'draw' || chessState?.status === 'resigned';

	let selectedSquare: string | null = null;

	interface Square {
		piece: string;
		color: 'light' | 'dark';
		position: string;
	}

	const pieceUnicode: Record<string, string> = {
		'K': '♔', 'Q': '♕', 'R': '♖', 'B': '♗', 'N': '♘', 'P': '♙',
		'k': '♚', 'q': '♛', 'r': '♜', 'b': '♝', 'n': '♞', 'p': '♟'
	};

	function parseFEN(fen: string): Square[][] {
		const rows = fen.split(' ')[0]?.split('/') ?? [];
		const board: Square[][] = [];
		for (let r = 0; r < rows.length; r++) {
			const row: Square[] = [];
			let col = 0;
			for (const ch of rows[r]) {
				if (/\d/.test(ch)) {
					const n = parseInt(ch);
					for (let i = 0; i < n; i++) {
						const pos = String.fromCharCode(97 + col) + (8 - r);
						row.push({ piece: '', color: (r + col) % 2 === 0 ? 'light' : 'dark', position: pos });
						col++;
					}
				} else {
					const pos = String.fromCharCode(97 + col) + (8 - r);
					row.push({ piece: ch, color: (r + col) % 2 === 0 ? 'light' : 'dark', position: pos });
					col++;
				}
			}
			board.push(row);
		}
		return board;
	}

	function handleSquareClick(position: string, piece: string) {
		if (!isMyTurn || gameOver) return;

		if (selectedSquare) {
			if (selectedSquare === position) {
				selectedSquare = null;
				return;
			}
			// Attempt move
			sendGameMove(activityId, 'move', {
				from: selectedSquare,
				to: position,
				fen: '' // Server validates; client sends empty FEN, server-side would compute
			});
			selectedSquare = null;
		} else if (piece) {
			// Select a piece (only own pieces)
			const isUpperCase = piece === piece.toUpperCase();
			if ((isWhite && isUpperCase) || (isBlack && !isUpperCase)) {
				selectedSquare = position;
			}
		}
	}

	async function handleJoin() {
		await sendGameMove(activityId, 'join', {});
	}

	async function handleResign() {
		await sendGameMove(activityId, 'resign', {});
	}

	function getPlayerName(userId: string | undefined): string {
		if (!userId) return 'Waiting...';
		const p = participants.find(p => p.user_id === userId);
		return p?.display_name || p?.username || 'Unknown';
	}
</script>

<div class="chess-game">
	{#if chessState}
		<!-- Player info -->
		<div class="player-bar top">
			<span class="piece-icon">♚</span>
			<span class="player-label">Black: {getPlayerName(chessState.black_player)}</span>
			{#if chessState.current_turn === 'black' && !gameOver}
				<span class="turn-indicator">thinking...</span>
			{/if}
		</div>

		<!-- Chess Board -->
		<div class="chess-board">
			{#each board as row, r}
				{#each row as square}
					<button
						class="square {square.color}"
						class:selected={selectedSquare === square.position}
						class:valid-target={selectedSquare && !square.piece}
						on:click={() => handleSquareClick(square.position, square.piece)}
						disabled={!isMyTurn || gameOver}
					>
						{#if square.piece}
							<span class="piece" class:white-piece={square.piece === square.piece.toUpperCase()}>
								{pieceUnicode[square.piece] || square.piece}
							</span>
						{/if}
					</button>
				{/each}
			{/each}
		</div>

		<div class="player-bar bottom">
			<span class="piece-icon">♔</span>
			<span class="player-label">White: {getPlayerName(chessState.white_player)}</span>
			{#if chessState.current_turn === 'white' && !gameOver}
				<span class="turn-indicator">thinking...</span>
			{/if}
		</div>

		<!-- Move history -->
		{#if chessState.move_history.length > 0}
			<div class="move-history">
				<span class="history-label">Moves:</span>
				{#each chessState.move_history as move, i}
					<span class="move">{i + 1}. {move}</span>
				{/each}
			</div>
		{/if}

		<!-- Status / Actions -->
		<div class="chess-status">
			{#if gameOver}
				<div class="game-over">
					{#if chessState.status === 'checkmate'}Checkmate! {getPlayerName(chessState.winner)} wins!
					{:else if chessState.status === 'stalemate'}Stalemate! It's a draw.
					{:else if chessState.status === 'draw'}Game drawn.
					{:else if chessState.status === 'resigned'}
						{getPlayerName(chessState.winner)} wins by resignation!
					{/if}
				</div>
			{:else if canJoin}
				<button class="btn-join" on:click={handleJoin}>Join as Black</button>
			{:else if isMyTurn}
				<div class="your-turn">Your turn!</div>
				<button class="btn-resign" on:click={handleResign}>Resign</button>
			{:else if chessState.status === 'waiting'}
				<div class="waiting">Waiting for opponent...</div>
			{:else}
				<div class="waiting">Opponent's turn</div>
				<button class="btn-resign" on:click={handleResign}>Resign</button>
			{/if}
		</div>
	{:else}
		<div class="loading-state">Loading chess game...</div>
	{/if}
</div>

<style>
	.chess-game {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
	}

	.chess-board {
		display: grid;
		grid-template-columns: repeat(8, 48px);
		grid-template-rows: repeat(8, 48px);
		border: 2px solid var(--border-subtle, #1e1f22);
		border-radius: 4px;
		overflow: hidden;
	}

	.square {
		width: 48px;
		height: 48px;
		display: flex;
		align-items: center;
		justify-content: center;
		border: none;
		cursor: pointer;
		padding: 0;
		transition: background 0.1s ease;
	}

	.square.light { background: #f0d9b5; }
	.square.dark { background: #b58863; }
	.square.selected { background: #7b61ff !important; }
	.square:hover:not(:disabled) { filter: brightness(1.1); }
	.square:disabled { cursor: default; }

	.piece {
		font-size: 32px;
		line-height: 1;
		user-select: none;
	}

	.white-piece { filter: drop-shadow(0 1px 1px rgba(0,0,0,0.3)); }

	.player-bar {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 12px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 4px;
		width: 384px;
	}

	.piece-icon { font-size: 20px; }
	.player-label { font-size: 13px; color: var(--text-primary, #f2f3f5); font-weight: 600; }
	.turn-indicator { font-size: 11px; color: #faa61a; margin-left: auto; }

	.move-history {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		max-width: 384px;
		padding: 8px;
		background: var(--bg-secondary, #2b2d31);
		border-radius: 4px;
		font-size: 11px;
	}

	.history-label { color: var(--text-muted, #6d6f78); margin-right: 4px; }
	.move { color: var(--text-secondary, #b5bac1); }

	.chess-status {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.game-over {
		font-size: 16px;
		font-weight: 700;
		color: #ffd700;
		text-align: center;
	}

	.your-turn {
		color: #43b581;
		font-weight: 600;
		font-size: 14px;
	}

	.waiting {
		color: var(--text-muted, #6d6f78);
		font-size: 13px;
	}

	.btn-join {
		padding: 8px 24px;
		background: #43b581;
		color: white;
		border: none;
		border-radius: 4px;
		font-weight: 600;
		cursor: pointer;
	}

	.btn-join:hover { background: #3ba374; }

	.btn-resign {
		padding: 6px 16px;
		background: #ed4245;
		color: white;
		border: none;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 600;
		cursor: pointer;
	}

	.btn-resign:hover { background: #c93b3e; }

	.loading-state {
		text-align: center;
		color: var(--text-muted, #6d6f78);
		padding: 40px;
	}
</style>
