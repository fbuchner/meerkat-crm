<script lang="ts">
  import { Search, Select, Button, Spinner, PaginationNav } from 'flowbite-svelte';
  import { SearchOutline as SearchIcon, CirclePlusOutline } from "flowbite-svelte-icons";;

  export let search = '';
  export let circle = '';
  export let circles: string[] = [];
  export let loading = false;
  export let page = 1;
  export let limit = 25;
  export let total = 0;
  
  export let onSearch: (search: string) => void;
  export let onCircleChange: (circle: string) => void;
  export let onPageChange: (page: number) => void;
  export let onAddNew: () => void;
  
  // Calculate total pages
  $: totalPages = Math.ceil(total / limit) || 1;
  
  // Handle search input
  function handleSearch() {
    onSearch(search);
  }
  
  // Handle circle selection
  function handleCircleChange(event: Event) {
    const select = event.target as HTMLSelectElement;
    onCircleChange(select.value);
  }
</script>

<div class="mb-6 flex flex-col md:flex-row gap-4">
  <div class="flex-1">
    <Search bind:value={search} placeholder="Search contacts..." size="md" onkeyup={(e) => e.key === 'Enter' && handleSearch()}>
      <Button onclick={handleSearch}>
        <SearchIcon class="w-5 h-5" />
      </Button>
    </Search>
  </div>
  
  <div class="w-full md:w-48">
    <Select bind:value={circle} class="w-full" onchange={handleCircleChange}>
      <option value="">All circles</option>
      {#each circles as c}
        <option value={c}>{c}</option>
      {/each}
    </Select>
  </div>
  
  <div>
    <Button color="blue" onclick={onAddNew}>
      <CirclePlusOutline class="w-5 h-5 mr-2" />
      Add Contact
    </Button>
  </div>
</div>

{#if loading}
  <div class="flex justify-center my-8">
    <Spinner size="12" />
  </div>
{:else if total > 0}
  <div class="flex justify-between items-center my-4">
    <p class="text-sm text-gray-600 dark:text-gray-400">
      Showing {((page - 1) * limit) + 1} to {Math.min(page * limit, total)} of {total} contacts
    </p>
    
    {#if totalPages > 1}
      <PaginationNav 
        currentPage={page} 
        totalPages={totalPages} 
        onPageChange={onPageChange} 
        visiblePages={5}
      />
    {/if}
  </div>
{/if}
