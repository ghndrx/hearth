import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	// The guild discovery data is loaded client-side in the component
	// This is because it requires authentication and the page is dynamic
	// Return empty props - component handles data fetching
	return {
		title: 'Discover Servers'
	};
};
