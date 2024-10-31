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

export default {
  name: 'App',
  components: {
    HeaderNav,
  },
  methods: {
    handleSearch(query) {
      // Handle the search logic, e.g., filter contacts based on query
      console.log('Search query:', query);
    },
  },
};
</script>

