<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import { contactService, type Contact } from '$lib/services/contactService';
  import { Button, Card, Spinner, Tabs, TabItem, Badge } from 'flowbite-svelte';
  import { ArrowLeftOutline, EditOutline, TrashBinOutline } from "flowbite-svelte-icons";
  import ProfilePicture from '$lib/components/ProfilePicture.svelte';
  
  export let contactId: number;
  
  const dispatch = createEventDispatcher();
  
  let contact: Contact | null = null;
  let loading = true;
  let error: string | null = null;
  
  onMount(async () => {
    await loadContact();
  });
  
  async function loadContact() {
    loading = true;
    error = null;
    
    try {
      contact = await contactService.getContact(contactId);
    } catch (err) {
      console.error('Error loading contact:', err);
      error = err instanceof Error ? err.message : 'Failed to load contact details';
    } finally {
      loading = false;
    }
  }
  
  function getInitials(firstname: string, lastname: string): string {
    const firstInitial = firstname ? firstname.charAt(0).toUpperCase() : '';
    const lastInitial = lastname ? lastname.charAt(0).toUpperCase() : '';
    return `${firstInitial}${lastInitial}`;
  }
  
  function formatDate(dateString: string | undefined): string {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleDateString();
  }
  
  function goBack() {
    dispatch('back');
  }
  
  function editContact() {
    if (contact) {
      dispatch('edit', contact);
    }
  }
  
  function deleteContact() {
    if (contact) {
      if (confirm(`Are you sure you want to delete ${contact.firstname} ${contact.lastname}?`)) {
        contactService.deleteContact(contact.ID)
          .then(() => {
            dispatch('deleted', contact?.ID);
            goBack();
          })
          .catch(err => {
            error = err instanceof Error ? err.message : 'Failed to delete contact';
          });
      }
    }
  }
</script>

<div class="p-4">
  <div class="flex items-center gap-4 mb-6">
    <Button color="light" class="p-2" onclick={goBack}>
      <ArrowLeftOutline class="w-5 h-5" />
    </Button>
    <h1 class="text-3xl font-bold">Contact Details</h1>
  </div>
  
  {#if loading}
    <div class="flex justify-center my-8">
      <Spinner size="12" />
    </div>
  {:else if error}
    <div class="p-4 mb-4 text-sm text-red-700 bg-red-100 rounded-lg dark:bg-red-200 dark:text-red-800">
      <span class="font-medium">Error!</span> {error}
    </div>
  {:else if contact}
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Profile card -->
      <Card class="lg:col-span-1">          <div class="flex flex-col items-center text-center">
            <ProfilePicture 
              contactId={contact.ID} 
              photo={contact.photo}
              initials={getInitials(contact.firstname, contact.lastname)} 
              size="xl"
              styleclass="mb-4"
            />
            
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
              {contact.firstname} {contact.lastname}
            </h2>
          
          {#if contact.nickname}
            <p class="text-gray-600 dark:text-gray-400">"{contact.nickname}"</p>
          {/if}
          
          {#if contact.circles && contact.circles.length > 0}
            <div class="mt-3 flex flex-wrap justify-center gap-2">
              {#each contact.circles as circle}
                <Badge color="blue">{circle}</Badge>
              {/each}
            </div>
          {/if}
          
          <div class="mt-6 flex gap-2">
            <Button color="blue" onclick={editContact}>
              <EditOutline class="w-4 h-4 mr-2" />
              Edit
            </Button>
            <Button color="red" onclick={deleteContact}>
              <TrashBinOutline class="w-4 h-4 mr-2" />
              Delete
            </Button>
          </div>
        </div>
      </Card>
      
      <!-- Details tabs -->
      <Card class="lg:col-span-2">
        <Tabs style="underline">
          <TabItem open title="Details">
            <div class="space-y-4">
              {#if contact.email}
                <div>
                  <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400">Email</h3>
                  <p class="text-base text-gray-900 dark:text-white">{contact.email}</p>
                </div>
              {/if}
              
              {#if contact.phone}
                <div>
                  <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400">Phone</h3>
                  <p class="text-base text-gray-900 dark:text-white">{contact.phone}</p>
                </div>
              {/if}
              
              {#if contact.birthday}
                <div>
                  <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400">Birthday</h3>
                  <p class="text-base text-gray-900 dark:text-white">{formatDate(contact.birthday)}</p>
                </div>
              {/if}
              
              {#if contact.gender}
                <div>
                  <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400">Gender</h3>
                  <p class="text-base text-gray-900 dark:text-white">{contact.gender}</p>
                </div>
              {/if}
              
              {#if contact.address}
                <div>
                  <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400">Address</h3>
                  <p class="text-base text-gray-900 dark:text-white">{contact.address}</p>
                </div>
              {/if}
              
              {#if contact.how_we_met}
                <div>
                  <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400">How We Met</h3>
                  <p class="text-base text-gray-900 dark:text-white">{contact.how_we_met}</p>
                </div>
              {/if}
              
              {#if contact.food_preference}
                <div>
                  <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400">Food Preference</h3>
                  <p class="text-base text-gray-900 dark:text-white">{contact.food_preference}</p>
                </div>
              {/if}
              
              {#if contact.work_information}
                <div>
                  <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400">Work Information</h3>
                  <p class="text-base text-gray-900 dark:text-white">{contact.work_information}</p>
                </div>
              {/if}
              
              {#if contact.contact_information}
                <div>
                  <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400">Additional Contact Information</h3>
                  <p class="text-base text-gray-900 dark:text-white">{contact.contact_information}</p>
                </div>
              {/if}
            </div>
          </TabItem>
          
          <TabItem title="Notes">
            {#if contact.notes && contact.notes.length > 0}
              <div class="space-y-4">
                {#each contact.notes as note}
                  <Card>
                    <h3 class="text-lg font-semibold">{note.title}</h3>
                    <p class="text-gray-700 dark:text-gray-400">{note.content}</p>
                    <p class="text-xs text-gray-500 mt-2">{formatDate(note.CreatedAt)}</p>
                  </Card>
                {/each}
              </div>
            {:else}
              <p class="text-gray-500 dark:text-gray-400">No notes for this contact yet.</p>
            {/if}
          </TabItem>
          
          <TabItem title="Relationships">
            {#if contact.relationships && contact.relationships.length > 0}
              <div class="space-y-4">
                {#each contact.relationships as relationship}
                  <Card>
                    <h3 class="text-lg font-semibold">{relationship.relationship_type}</h3>
                    <p class="text-gray-700 dark:text-gray-400">{relationship.description}</p>
                  </Card>
                {/each}
              </div>
            {:else}
              <p class="text-gray-500 dark:text-gray-400">No relationships for this contact yet.</p>
            {/if}
          </TabItem>
          
          <TabItem title="Activities">
            {#if contact.activities && contact.activities.length > 0}
              <div class="space-y-4">
                {#each contact.activities as activity}
                  <Card>
                    <h3 class="text-lg font-semibold">{activity.title}</h3>
                    <p class="text-gray-700 dark:text-gray-400">{activity.description}</p>
                    <p class="text-xs text-gray-500 mt-2">
                      {formatDate(activity.date)}
                    </p>
                  </Card>
                {/each}
              </div>
            {:else}
              <p class="text-gray-500 dark:text-gray-400">No activities for this contact yet.</p>
            {/if}
          </TabItem>
        </Tabs>
      </Card>
    </div>
  {/if}
</div>
