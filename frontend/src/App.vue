<template>
  <div id="app">
    <HeaderNav @search="handleSearch" />
    <router-view></router-view> <!-- Router will inject the matched component here -->
  </div>
</template>

<script>
import HeaderNav from './components/HeaderNav.vue';

export default {
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

