<script lang="ts">
  import '../app.css';
  import { Sidebar, SidebarGroup, SidebarItem, SidebarButton, uiHelpers } from "flowbite-svelte";
  import { ChartOutline, UsersGroupOutline, TheatreOutline,  RectangleListOutline} from "flowbite-svelte-icons";
  import { page } from "$app/state";
  let activeUrl = $state(page.url.pathname);
  const spanClass = "flex-1 ms-3 whitespace-nowrap";
  const SidebarUi = uiHelpers();
  let isSidebarOpen = $state(false);
  const closeSidebar = SidebarUi.close;
  $effect(() => {
    isSidebarOpen = SidebarUi.isOpen;
    activeUrl = page.url.pathname;
  });
</script>

<SidebarButton onclick={SidebarUi.toggle} class="mb-2" />
<div class="relative">
  <Sidebar {activeUrl} backdrop={false} isOpen={isSidebarOpen} closeSidebar={closeSidebar} params={{ x: -50, duration: 50 }} class="z-50 h-full" position="absolute" activeClass="p-2" nonActiveClass="p-2">
    <SidebarGroup>
		<SidebarItem label="Dashboard" href="/">
        {#snippet icon()}
          <ChartOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
        {/snippet}
      </SidebarItem>
      <SidebarItem label="Contacts" href="/">
        {#snippet icon()}
          <UsersGroupOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
        {/snippet}
      </SidebarItem>
      <SidebarItem label="Activities" {spanClass} href="/">
        {#snippet icon()}
          <TheatreOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
        {/snippet}
      </SidebarItem>
      <SidebarItem label="Notes" {spanClass} href="/">
        {#snippet icon()}
          <RectangleListOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
        {/snippet}
      </SidebarItem>
    </SidebarGroup>
  </Sidebar>
  <div class="h-96 overflow-auto px-4 md:ml-64">
    <div class="rounded-lg border-2 border-dashed border-gray-200 p-4 dark:border-gray-700">

    </div>
  </div>
</div>