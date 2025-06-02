<script>
  import { onMount } from 'svelte';
  import { Spinner } from 'flowbite-svelte';
  
</script>

<div class="h-screen w-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
  <div class="text-center">
    <Spinner size="12" color="blue" />
    <p class="mt-4 text-gray-600 dark:text-gray-400">Loading...</p>
  </div>
</div>