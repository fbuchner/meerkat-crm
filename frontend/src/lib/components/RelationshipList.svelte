<script lang="ts">
  import { onMount } from 'svelte';
  import { contactService } from '$lib/services/contactService';
  import { Button, Card, Input, Select, Textarea, Modal, ButtonGroup, Badge, Tabs, TabItem } from 'flowbite-svelte';
  import { PlusOutline, EditOutline, TrashBinOutline, CheckOutline, CloseOutline, ChevronDownOutline, ChevronUpOutline } from "flowbite-svelte-icons";
  import ProfilePicture from '$lib/components/ProfilePicture.svelte';

  // Props
  let { contactId } = $props<{
    contactId: number;
  }>();

  // State
  let relationships = $state<any[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let showModal = $state(false);
  let isCollapsed = $state(false);
  let activeTab = $state('manual');
  let editingRelationship = $state<any>(null);

  // Form state
  let relationshipForm = $state<{
    type: string;
    name: string;
    gender: string;
    birthday: string;
    related_contact_id: number | null;
    related_contact: { value: any; name: string } | null;
  }>({
    type: '',
    name: '',
    gender: '',
    birthday: '',
    related_contact_id: null,
    related_contact: null
  });

  // Available contacts for selection
  let availableContacts = $state<any[]>([]);

  // Relationship types
  const relationshipTypes = [
    'Parent', 'Child', 'Sibling', 'Spouse', 'Partner', 'Friend', 
    'Colleague', 'Neighbor', 'Relative', 'Acquaintance', 'Other'
  ];

  const genderOptions = ['Male', 'Female', 'Other', 'Prefer not to say'];

  onMount(async () => {
    await fetchRelationships();
    await loadAvailableContacts();
  });

  async function fetchRelationships() {
    loading = true;
    error = null;
    
    try {
      const response = await contactService.getRelationships(contactId);
      relationships = response.relationships || [];
    } catch (err) {
      console.error('Error fetching relationships:', err);
      error = err instanceof Error ? err.message : 'Failed to load relationships';
    } finally {
      loading = false;
    }
  }

  async function loadAvailableContacts() {
    try {
      const response = await contactService.getContacts({
        fields: ['ID', 'firstname', 'lastname'],
        limit: 100
      });
      availableContacts = response.contacts
        .filter(contact => contact.ID !== contactId)
        .map(contact => ({
          value: contact.ID,
          name: `${contact.firstname} ${contact.lastname}`,
          ...contact
        }));
    } catch (err) {
      console.error('Error loading contacts:', err);
    }
  }

  function formatBirthday(birthday: string | undefined): string {
    if (!birthday) return '';
    const [year, month, day] = birthday.split('-');
    return `${day}.${month}.${year !== '0001' ? year : ''}`;
  }

  function parseBirthday(formattedBirthday: string): string | null {
    if (!formattedBirthday.trim()) return null;
    
    const datePattern = /^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])\.(\d{4})?$/;
    const match = formattedBirthday.match(datePattern);
    
    if (match) {
      const [, day, month, year] = match;
      return `${year || '0001'}-${month}-${day}`;
    }
    
    return null;
  }

  function openAddModal() {
    editingRelationship = null;
    resetForm();
    showModal = true;
  }

  function openEditModal(relationship: any) {
    editingRelationship = relationship;
    relationshipForm = {
      type: relationship.type || '',
      name: relationship.name || '',
      gender: relationship.gender || '',
      birthday: formatBirthday(relationship.birthday),
      related_contact_id: relationship.related_contact_id || null,
      related_contact: relationship.related_contact ? {
        value: relationship.related_contact.ID,
        name: `${relationship.related_contact.firstname} ${relationship.related_contact.lastname}`
      } : null
    };
    
    activeTab = relationship.related_contact_id ? 'existing' : 'manual';
    showModal = true;
  }

  function resetForm() {
    relationshipForm = {
      type: '',
      name: '',
      gender: '',
      birthday: '',
      related_contact_id: null,
      related_contact: null
    };
    activeTab = 'manual';
  }

  function closeModal() {
    showModal = false;
    resetForm();
    editingRelationship = null;
  }

  async function saveRelationship() {
    try {
      const relationshipData: any = {
        type: relationshipForm.type,
        contact_id: contactId,
        name: null,
        gender: null,
        birthday: null,
        related_contact_id: null
      };

      if (activeTab === 'manual') {
        if (!relationshipForm.name || !relationshipForm.type) {
          error = 'Please provide both name and relationship type.';
          return;
        }

        relationshipData.name = relationshipForm.name;
        relationshipData.gender = relationshipForm.gender || null;
        relationshipData.birthday = parseBirthday(relationshipForm.birthday);
      } else if (activeTab === 'existing') {
        if (!relationshipForm.related_contact || !relationshipForm.type) {
          error = 'Please select an existing contact and provide the relationship type.';
          return;
        }

        relationshipData.related_contact_id = relationshipForm.related_contact.value;
      }

      if (editingRelationship) {
        await contactService.updateRelationship(contactId, editingRelationship.ID, relationshipData);
      } else {
        await contactService.addRelationship(contactId, relationshipData);
      }

      await fetchRelationships();
      closeModal();
      error = null;
    } catch (err) {
      console.error('Error saving relationship:', err);
      error = err instanceof Error ? err.message : 'Failed to save relationship';
    }
  }

  async function deleteRelationship(relationshipId: number) {
    if (!confirm('Are you sure you want to delete this relationship?')) {
      return;
    }

    try {
      await contactService.deleteRelationship(contactId, relationshipId);
      await fetchRelationships();
      error = null;
    } catch (err) {
      console.error('Error deleting relationship:', err);
      error = err instanceof Error ? err.message : 'Failed to delete relationship';
    }
  }

  function toggleCollapse() {
    isCollapsed = !isCollapsed;
  }
</script>

<Card class="mb-4">
  <div class="flex items-center justify-between mb-4">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Relationships</h3>
    <button 
      class="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
      onclick={toggleCollapse}
    >
      {#if isCollapsed}
        <ChevronDownOutline class="w-4 h-4" />
      {:else}
        <ChevronUpOutline class="w-4 h-4" />
      {/if}
    </button>
  </div>

  {#if !isCollapsed}
    {#if loading}
      <p class="text-gray-500">Loading relationships...</p>
    {:else if error}
      <div class="p-4 mb-4 text-sm text-red-700 bg-red-100 rounded-lg dark:bg-red-200 dark:text-red-800">
        <span class="font-medium">Error!</span> {error}
      </div>
    {:else}
      <div class="space-y-3">
        {#each relationships as relationship}
          <div class="group flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <div class="flex items-center gap-3">
              {#if relationship.related_contact_id && relationship.related_contact}
                <ProfilePicture 
                  contactId={relationship.related_contact.ID} 
                  initials={`${relationship.related_contact.firstname?.charAt(0) || ''}${relationship.related_contact.lastname?.charAt(0) || ''}`}
                  size="sm"
                />
                <div>
                  <p class="font-medium text-gray-900 dark:text-white">
                    <strong>{relationship.type}:</strong> 
                    {relationship.related_contact.firstname} {relationship.related_contact.lastname}
                  </p>
                  {#if relationship.related_contact.birthday}
                    <p class="text-sm text-gray-500">
                      ({formatBirthday(relationship.related_contact.birthday)})
                    </p>
                  {/if}
                </div>
              {:else}
                <div class="w-8 h-8 bg-gray-300 dark:bg-gray-600 rounded-full flex items-center justify-center">
                  <span class="text-sm font-medium text-gray-600 dark:text-gray-300">
                    {relationship.name?.charAt(0)?.toUpperCase() || '?'}
                  </span>
                </div>
                <div>
                  <p class="font-medium text-gray-900 dark:text-white">
                    <strong>{relationship.type}:</strong> {relationship.name}
                  </p>
                  {#if relationship.birthday}
                    <p class="text-sm text-gray-500">
                      ({formatBirthday(relationship.birthday)})
                    </p>
                  {/if}
                </div>
              {/if}
            </div>
            
            <div class="opacity-0 group-hover:opacity-100 transition-opacity flex gap-2">
              <button 
                class="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
                onclick={() => openEditModal(relationship)}
              >
                <EditOutline class="w-4 h-4 text-gray-500" />
              </button>
              <button 
                class="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
                onclick={() => deleteRelationship(relationship.ID)}
              >
                <TrashBinOutline class="w-4 h-4 text-red-500" />
              </button>
            </div>
          </div>
        {/each}

        <button 
          class="flex items-center gap-2 p-3 w-full text-left text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg transition-colors"
          onclick={openAddModal}
        >
          <PlusOutline class="w-4 h-4" />
          Add Relationship
        </button>
      </div>
    {/if}
  {/if}
</Card>

<!-- Add/Edit Relationship Modal -->
<Modal bind:open={showModal} size="md" autoclose={false}>
  <div class="p-6">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
      {editingRelationship ? 'Edit Relationship' : 'Add Relationship'}
    </h3>

    <Tabs style="underline" class="mb-4">
      <TabItem value="manual" title="Manual Entry" />
      <TabItem value="existing" title="Existing Contact" />
    </Tabs>

    {#if activeTab === 'manual'}
      <div class="space-y-4">
        <div>
          <label for="relationship-type-existing" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Relationship Type
          </label>
          <Select 
            id="relationship-type-existing"
            bind:value={relationshipForm.type}
            items={relationshipTypes.map(type => ({ value: type, name: type }))}
            placeholder="Select relationship type"
          />
        </div>

        <div>
          <label for="relationship-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Name
          </label>
          <Input 
            id="relationship-name"
            bind:value={relationshipForm.name}
            placeholder="Enter name"
          />
        </div>

        <div>
          <label for="relationship-gender" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Gender
          </label>
          <Select 
            id="relationship-gender"
            bind:value={relationshipForm.gender}
            items={genderOptions.map(gender => ({ value: gender, name: gender }))}
            placeholder="Select gender"
          />
        </div>

        <div>
          <label for="relationship-birthday" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Birthday
          </label>
          <Input 
            id="relationship-birthday"
            bind:value={relationshipForm.birthday}
            placeholder="DD.MM.YYYY or DD.MM."
          />
        </div>
      </div>
    {:else}
      <div class="space-y-4">
        <div>
          <label for="relationship-type-existing-tab" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Relationship Type
          </label>
          <Select 
            id="relationship-type-existing-tab"
            bind:value={relationshipForm.type}
            items={relationshipTypes.map(type => ({ value: type, name: type }))}
            placeholder="Select relationship type"
          />
        </div>

        <div>
          <label for="relationship-existing-contact" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Existing Contact
          </label>
          <Select 
            id="relationship-existing-contact"
            bind:value={relationshipForm.related_contact}
            items={availableContacts}
            placeholder="Select existing contact"
          />
        </div>
      </div>
    {/if}

    <div class="flex justify-end gap-3 mt-6">
      <Button color="light" onclick={closeModal}>
        Cancel
      </Button>
      <Button color="blue" onclick={saveRelationship}>
        {editingRelationship ? 'Save Changes' : 'Add Relationship'}
      </Button>
    </div>
  </div>
</Modal>