<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { Card, Alert } from "flowbite-svelte";
  import { contactsStore, contactFilters, selectedContact } from '$lib/stores/contactStore';
  import { contactService, type Contact } from '$lib/services/contactService';
  import ContactCard from '$lib/components/ContactCard.svelte';
  import ContactsFilter from '$lib/components/ContactsFilter.svelte';
  
  let circles: string[] = [];
  let search = '';
  let circle = '';
  let page = 1;
  let limit = 25;
  let loading = true;
  let error: string | null = null;
  
  // Subscribe to the stores
  $: ({ contacts, total } = $contactsStore);
  
  // Load contacts on mount and when filters change
  onMount(async () => {
    try {
      // Load the available circles
      await fetchCircles();
      
      // Initial contacts load
      await fetchContacts();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load contacts';
    } finally {
      loading = false;
    }
  });
  
  async function fetchCircles() {
    try {
      const response = await contactService.getCircles();
      circles = response.circles || [];
    } catch (err) {
      console.error('Error fetching circles:', err);
      // Non-critical error, so we'll just log it
    }
  }
  
  async function fetchContacts() {
    loading = true;
    error = null;
    
    try {
      // Update the filters store
      $contactFilters = { search, circle, page, limit };
      
      // Fetch contacts with current filters
      const response = await contactService.getContacts({
        search,
        circle,
        page,
        limit,
        includes: []
      });
      
      // Update the contacts store
      contactsStore.set({
        contacts: response.contacts,
        total: response.total,
        page: response.page,
        limit: response.limit,
        loading: false,
        error: null
      });
    } catch (err) {
      console.error('Error fetching contacts:', err);
      error = err instanceof Error ? err.message : 'Failed to load contacts';
      contactsStore.update(state => ({ ...state, loading: false, error }));
    } finally {
      loading = false;
    }
  }
  
  // Handler functions for filters
  function handleSearch(newSearch: string) {
    search = newSearch;
    page = 1; // Reset to first page on new search
    fetchContacts();
  }
  
  function handleCircleChange(newCircle: string) {
    circle = newCircle;
    page = 1; // Reset to first page on circle change
    fetchContacts();
  }
  
  function handlePageChange(newPage: number) {
    page = newPage;
    fetchContacts();
  }
  
  function handleAddNew() {
    // This will be implemented later for adding a new contact
    console.log('Add new contact');
  }
  
  function viewContactDetails(contact: Contact) {
    selectedContact.set(contact);
    goto(`/contacts/${contact.ID}`);
  }
</script>

<div class="p-4">
  <h1 class="text-3xl font-bold mb-6">Contacts</h1>
  
  <ContactsFilter
    {search}
    {circle}
    {circles}
    {loading}
    {page}
    {limit}
    {total}
    onSearch={handleSearch}
    onCircleChange={handleCircleChange}
    onPageChange={handlePageChange}
    onAddNew={handleAddNew}
  />
  
  {#if error}
    <Alert color="red" class="my-4">
      <span class="font-medium">Error!</span> {error}
    </Alert>
  {/if}
  
  {#if !loading && contacts.length === 0}
    <Card class="my-4">
      <h5 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">No contacts found</h5>
      <p class="text-gray-700 dark:text-gray-400">
        {search || circle 
          ? 'No contacts match your search criteria. Try a different search or filter.'
          : 'You have no contacts yet. Add your first contact to get started.'}
      </p>
    </Card>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 my-4">
      {#each contacts as contact (contact.ID)}
        <div on:click={() => viewContactDetails(contact)} on:keydown={(e) => e.key === 'Enter' && viewContactDetails(contact)} tabindex="0" role="button">
          <ContactCard {contact} />
        </div>
      {/each}
    </div>
  {/if}
</div>
