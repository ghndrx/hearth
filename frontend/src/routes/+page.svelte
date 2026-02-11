<script lang="ts">
	import { Sidebar, ChannelList, MessageArea, MemberList } from '$components';
</script>

<svelte:head>
	<title>Hearth</title>
</svelte:head>

<div class="h-screen flex overflow-hidden">
	<Sidebar />
	<ChannelList />
	<MessageArea />
	<MemberList />
</div>
