<script lang="ts">
  import '../app.css';
  import { Sidebar, SidebarGroup, SidebarItem, SidebarButton, uiHelpers } from "flowbite-svelte";
  import { ChartOutline, UsersGroupOutline, TheatreOutline, RectangleListOutline, ArrowRightToBracketOutline } from "flowbite-svelte-icons";
  import { page } from "$app/state";
  import { browser } from '$app/environment';
  import { onMount } from 'svelte';
  import { userStore } from '$lib/stores/userStore';
  import userService from '$lib/services/userService';
  import { goto } from '$app/navigation';

  let { children } = $props() 
  
  let activeUrl = $state(page.url.pathname);
  const spanClass = "flex-1 ms-3 whitespace-nowrap";
  const SidebarUi = uiHelpers();
  let isSidebarOpen = $state(false);
  const closeSidebar = SidebarUi.close;
  
  // Reactive statement to track authentication state
  let isAuthenticated = $state(false);
  
  $effect(() => {
    isSidebarOpen = SidebarUi.isOpen;
    activeUrl = page.url.pathname;
  });
  
  // Check if user is authenticated on mount
  onMount(() => {
    if (browser) {
      isAuthenticated = userService.isAuthenticated();
      
      // Update user store based on token presence
      if (isAuthenticated) {
        userStore.set({
          isAuthenticated: true,
          user: null // We could fetch user details here if needed
        });
      }
    }
  });
  
  // Subscribe to user store changes
  userStore.subscribe(state => {
    isAuthenticated = state.isAuthenticated;
  });
  
  // Handle logout
  function handleLogout() {
    userService.logout();
    goto('/login');
  }
</script>

<SidebarButton onclick={SidebarUi.toggle} class="mb-2" />
<div class="relative">
  <Sidebar {activeUrl} backdrop={false} isOpen={isSidebarOpen} closeSidebar={closeSidebar} params={{ x: -50, duration: 50 }} class="z-50 h-full" position="absolute" activeClass="p-2" nonActiveClass="p-2">
    <SidebarGroup>
      {#if isAuthenticated}
        <SidebarItem label="Dashboard" href="/dashboard">
          {#snippet icon()}
            <ChartOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
          {/snippet}
        </SidebarItem>
        <SidebarItem label="Contacts" href="/contacts">
          {#snippet icon()}
            <UsersGroupOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
          {/snippet}
        </SidebarItem>
        <SidebarItem label="Activities" {spanClass} href="/activities">
          {#snippet icon()}
            <TheatreOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
          {/snippet}
        </SidebarItem>
        <SidebarItem label="Notes" {spanClass} href="/notes">
          {#snippet icon()}
            <RectangleListOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
          {/snippet}
        </SidebarItem>
        <SidebarItem label="Logout" onclick={handleLogout}>
          {#snippet icon()}
            <ArrowRightToBracketOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
          {/snippet}
        </SidebarItem>
      {:else}
        <SidebarItem label="Login" href="/login">
          {#snippet icon()}
            <ChartOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
          {/snippet}
        </SidebarItem>
        <SidebarItem label="Register" href="/register">
          {#snippet icon()}
            <UsersGroupOutline class="h-5 w-5 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white" />
          {/snippet}
        </SidebarItem>
      {/if}
    </SidebarGroup>
  </Sidebar>
  <div class="min-h-screen overflow-auto px-4 md:ml-64">
    <div class="rounded-lg p-4">
        {@render children()}     </div>
  </div>
</div>