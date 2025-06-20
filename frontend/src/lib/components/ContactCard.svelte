<script lang="ts">
  import { Badge } from 'flowbite-svelte';
  import type { Contact } from '$lib/services/contactService';
  import ProfilePicture from '$lib/components/ProfilePicture.svelte';
  
  export let contact: Contact;
  
  function getInitials(firstname: string, lastname: string): string {
    const firstInitial = firstname ? firstname.charAt(0).toUpperCase() : '';
    const lastInitial = lastname ? lastname.charAt(0).toUpperCase() : '';
    return `${firstInitial}${lastInitial}`;
  }

  // Format the contact name for display
  $: fullName = [contact.firstname, contact.lastname]
    .filter(Boolean)
    .join(' ');
    
  $: initials = getInitials(contact.firstname, contact.lastname);
  
</script>

<div class="flex items-center p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors cursor-pointer">
  <div class="flex-shrink-0 mr-4">
    <ProfilePicture 
      contactId={contact.ID} 
      {initials} 
      size="lg" 
    />
  </div>
  
  <div class="flex-1 min-w-0">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-white truncate">
      {fullName}
      {#if contact.nickname}
        <p class="text-sm font-normal text-gray-500 dark:text-gray-400">
          ({contact.nickname})
        </p>
      {/if}
    </h3>
    
    
    {#if contact.circles && contact.circles.length > 0}
      <div class="mt-2 flex flex-wrap gap-2">
        {#each contact.circles as circle}
          <Badge color="blue">{circle}</Badge>
        {/each}
      </div>
    {/if}
  </div>
</div>
