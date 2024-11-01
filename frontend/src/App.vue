<template>
  <v-app>
    <HeaderNav @search="handleSearch" />
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

    // Define handleSearch as a function to update the search query
    function handleSearch(query) {
      searchQuery.value = query;
      console.log('Search query:', query);
    }

    provide('searchQuery', searchQuery); // Provide searchQuery to child components
    provide('setSearchQuery', handleSearch); // Provide the search update function

    return { handleSearch }; // Return handleSearch for use in template
  },
};
</script>
