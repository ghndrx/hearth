import { dev } from '$app/environment';

export class Logger {
	private constructor(private readonly prefix: string) {}

	static create(prefix: string): Logger {
		return new Logger(prefix);
	}

	debug(message: string, ...args: unknown[]): void {
		if (dev) {
			console.debug(`[${this.prefix}] ${message}`, ...args);
		}
	}

	info(message: string, ...args: unknown[]): void {
		if (dev) {
			console.log(`[${this.prefix}] ${message}`, ...args);
		}
	}

	warn(message: string, ...args: unknown[]): void {
		if (dev) {
			console.warn(`[${this.prefix}] ${message}`, ...args);
		}
	}

	error(message: string, ...args: unknown[]): void {
		if (dev) {
			console.error(`[${this.prefix}] ${message}`, ...args);
		}
	}
}