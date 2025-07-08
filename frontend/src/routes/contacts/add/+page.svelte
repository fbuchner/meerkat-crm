<script lang="ts">
  import { goto } from '$app/navigation';
  import ContactAdd from '$lib/components/ContactAdd.svelte';
  import { Button } from 'flowbite-svelte';
  import { ArrowLeftOutline } from 'flowbite-svelte-icons';

  function handleContactAdded() {
    // Navigate back to contacts list after successful addition
    goto('/contacts');
  }
</script>

<div class="container mx-auto p-4">
  <div class="mb-4">
    <Button color="light" class="flex items-center" onclick={() => goto('/contacts')}>
      <ArrowLeftOutline class="w-4 h-4 mr-2" />
      Back to Contacts
    </Button>
  </div>
  
  <ContactAdd on:contactAdded={handleContactAdded} />
</div>
