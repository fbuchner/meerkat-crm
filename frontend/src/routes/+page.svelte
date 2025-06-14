<script>
  import { onMount } from 'svelte';
  import { Spinner } from 'flowbite-svelte';
  import { goto } from '$app/navigation';
  import userService from '$lib/services/userService';
  import { browser } from '$app/environment';
  
  onMount(() => {
    if (browser) {
      // Redirect based on authentication status
      const timer = setTimeout(() => {
        if (userService.isAuthenticated()) {
          goto('/dashboard');
        } else {
          goto('/login');
        }
      }, 500);
      
      return () => clearTimeout(timer);
    }
  });
</script>

<div class="h-screen w-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
  <div class="text-center">
    <Spinner size="12" color="blue" />
    <p class="mt-4 text-gray-600 dark:text-gray-400">Redirecting...</p>
  </div>
</div>