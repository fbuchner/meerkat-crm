<script lang="ts">
  import { Avatar, Badge } from 'flowbite-svelte';
  import type { Contact } from '$lib/services/contactService';
  
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
  
  // Format the contact info for display
  $: contactInfo = [contact.email, contact.phone]
    .filter(Boolean)
    .join(' • ');
</script>

<div class="flex items-center p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors cursor-pointer">
  <div class="flex-shrink-0 mr-4">
    {#if contact.photo}
      <Avatar src={contact.photo} size="lg" cornerStyle="rounded"  />
    {:else}
      <Avatar size="lg" cornerStyle="rounded" >{initials}</Avatar>
    {/if}
  </div>
  
  <div class="flex-1 min-w-0">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-white truncate">
      {fullName}
      {#if contact.nickname}
        <span class="text-sm font-normal text-gray-500 dark:text-gray-400 ml-2">
          ({contact.nickname})
        </span>
      {/if}
    </h3>
    
    {#if contactInfo}
      <p class="text-sm text-gray-500 dark:text-gray-400 truncate">
        {contactInfo}
      </p>
    {/if}
    
    {#if contact.circles && contact.circles.length > 0}
      <div class="mt-2 flex flex-wrap gap-2">
        {#each contact.circles as circle}
          <Badge color="blue">{circle}</Badge>
        {/each}
      </div>
    {/if}
  </div>
</div>
