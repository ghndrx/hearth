/**
 * Custom Svelte transitions and animation utilities for Hearth.
 * All transitions respect prefers-reduced-motion automatically.
 */
import { cubicOut, cubicIn, quintOut } from 'svelte/easing';
import type { TransitionConfig } from 'svelte/transition';

/**
 * Check if the user prefers reduced motion.
 */
export function prefersReducedMotion(): boolean {
	if (typeof window === 'undefined') return false;
	return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/**
 * Scale duration based on reduced-motion preference.
 * Returns 0 for reduced motion, otherwise the original duration.
 */
function motionDuration(duration: number): number {
	return prefersReducedMotion() ? 0 : duration;
}

/**
 * Message appear animation: slides up and fades in.
 */
export function messageIn(
	node: Element,
	params: { duration?: number; delay?: number } = {}
): TransitionConfig {
	const duration = motionDuration(params.duration ?? 250);
	const delay = params.delay ?? 0;

	return {
		duration,
		delay,
		easing: cubicOut,
		css: (t: number) => {
			const y = (1 - t) * 8;
			return `opacity: ${t}; transform: translateY(${y}px);`;
		}
	};
}

/**
 * Slide + fade for sidebars and panels.
 */
export function panelSlide(
	node: Element,
	params: { duration?: number; direction?: 'left' | 'right' } = {}
): TransitionConfig {
	const duration = motionDuration(params.duration ?? 200);
	const direction = params.direction ?? 'right';
	const sign = direction === 'right' ? 1 : -1;

	return {
		duration,
		easing: cubicOut,
		css: (t: number) => {
			const x = (1 - t) * 20 * sign;
			return `opacity: ${t}; transform: translateX(${x}px);`;
		}
	};
}

/**
 * Enhanced modal backdrop transition.
 */
export function backdropFade(
	node: Element,
	params: { duration?: number } = {}
): TransitionConfig {
	const duration = motionDuration(params.duration ?? 150);

	return {
		duration,
		easing: cubicOut,
		css: (t: number) => `opacity: ${t};`
	};
}

/**
 * Enhanced modal content transition: scales up with slight bounce.
 */
export function modalPop(
	node: Element,
	params: { duration?: number } = {}
): TransitionConfig {
	const duration = motionDuration(params.duration ?? 200);

	return {
		duration,
		easing: quintOut,
		css: (t: number) => {
			const scale = 0.95 + t * 0.05;
			return `opacity: ${t}; transform: scale(${scale});`;
		}
	};
}

/**
 * Category collapse/expand with height animation.
 */
export function collapseVertical(
	node: Element,
	params: { duration?: number } = {}
): TransitionConfig {
	const duration = motionDuration(params.duration ?? 200);
	const height = node.scrollHeight;

	return {
		duration,
		easing: cubicOut,
		css: (t: number) => {
			return `height: ${t * height}px; opacity: ${t}; overflow: hidden;`;
		}
	};
}

/**
 * Tooltip/popover appear animation.
 */
export function popIn(
	node: Element,
	params: { duration?: number } = {}
): TransitionConfig {
	const duration = motionDuration(params.duration ?? 100);

	return {
		duration,
		easing: cubicOut,
		css: (t: number) => {
			const scale = 0.92 + t * 0.08;
			return `opacity: ${t}; transform: scale(${scale});`;
		}
	};
}

/**
 * Page crossfade transition for route changes.
 */
export function pageFade(
	node: Element,
	params: { duration?: number; delay?: number } = {}
): TransitionConfig {
	const duration = motionDuration(params.duration ?? 150);
	const delay = params.delay ?? 0;

	return {
		duration,
		delay,
		easing: cubicOut,
		css: (t: number) => `opacity: ${t};`
	};
}

/**
 * Staggered list item animation helper.
 * Returns delay based on item index for cascading effect.
 */
export function staggerDelay(index: number, baseDelay = 30, maxDelay = 300): number {
	if (prefersReducedMotion()) return 0;
	return Math.min(index * baseDelay, maxDelay);
}
