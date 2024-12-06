<template>
  <v-app>
    <HeaderNav @search="handleSearch" @resetFilters="resetFilters" />
    <v-main>
      <router-view /> <!-- Router will inject the matched component here -->
    </v-main>
  </v-app>
</template>

<script>
import HeaderNav from './components/HeaderNav.vue';
import { ref, provide } from 'vue';

export default {
  name: 'App',
  components: {
    HeaderNav,
  },
  setup() {
    const searchQuery = ref('');
    const clearFilters = ref(false);

    // Define handleSearch as a function to update the search query
    function handleSearch(query) {
      searchQuery.value = query;
    }

    function resetFilters() {
      clearFilters.value = true;
      // Reset the flag after handling it
      setTimeout(() => {
        clearFilters.value = false;
      }, 0);
    }

    provide('clearFilters', clearFilters);
    provide('searchQuery', searchQuery); // Provide searchQuery to child components
    provide('setSearchQuery', handleSearch); // Provide the search update function

    return { handleSearch, resetFilters }; // Return handleSearch for use in template
  },
};
</script>
