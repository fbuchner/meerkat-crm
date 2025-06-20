<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { selectedContact } from '$lib/stores/contactStore';
  import type { Contact } from '$lib/services/contactService';
  import ContactDetail from '$lib/components/ContactDetail.svelte';
  
  const contactId = parseInt($page.params.id);
  
  function handleBack() {
    goto('/contacts');
  }
  
  function handleDeleted() {
    goto('/contacts');
  }
  
  function handleEdit(contact: Contact) {
    selectedContact.set(contact);
    // In the future, you could implement edit functionality
    // by navigating to an edit route like: goto(`/contacts/${contact.ID}/edit`);
    console.log('Edit contact:', contact);
  }
</script>

<ContactDetail 
  contactId={contactId} 
  back={handleBack}
  deleted={handleDeleted}
  edit={handleEdit}
/>
