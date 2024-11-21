<template>
  <v-app-bar app>
    <v-container>
      <v-row align="center" justify="space-between">
        <!-- Logo -->
        <v-col cols="auto">
          <h1><v-btn text to="/" class="text-h5">perema</v-btn></h1>
        </v-col>

        <!-- Search Bar (hidden on mobile) -->
        <v-col cols="3" class="d-none d-md-flex">
          <v-text-field
            v-model="searchQuery"
            placeholder="Search contacts..."
            hide-details
            clearable
            density="compact"
            append-icon="mdi-magnify"
            autofocus
            @input="handleSearchInput"
            @click:clear="handleClearSearch"
          ></v-text-field>
        </v-col>

        <!-- Desktop Navigation Links (hidden on mobile) -->
        <v-col cols="auto" class="d-none d-md-flex justify-end">
          <v-btn text to="/contacts">Contacts</v-btn>
          <v-btn text to="/activities">Activities</v-btn>
          <v-btn text to="/notes">Notes</v-btn>
        </v-col>
      </v-row>
    </v-container>
  </v-app-bar>

  <!-- Bottom Navigation for Mobile -->
  <v-bottom-navigation
    v-if="isMobile"
    app
    color="primary"
  >
    <v-btn icon to="/contacts">
      <v-icon>mdi-account-multiple</v-icon>
      <span>Contacts</span>
    </v-btn>
    <v-btn icon to="/activities">
      <v-icon>mdi-calendar-check</v-icon>
      <span>Activities</span>
    </v-btn>
    <v-btn icon to="/notes">
      <v-icon>mdi-note</v-icon>
      <span>Notes</span>
    </v-btn>
  </v-bottom-navigation>
</template>

<script>
import { inject, ref } from 'vue';
import { useRouter } from 'vue-router';

export default {
  setup() {
    const searchQuery = ref('');
    const setSearchQuery = inject('setSearchQuery');
    const router = useRouter();

    function handleSearchInput() {
      if (!router.currentRoute.value.path.startsWith('/contacts')) {
        router.push('/contacts');
      }
      setSearchQuery(searchQuery.value);
    }

    function handleClearSearch() {
      searchQuery.value = '';
      setSearchQuery('');
    }

    const isMobile = ref(window.innerWidth <= 960);

    function handleResize() {
      isMobile.value = window.innerWidth <= 960;
    }

    window.addEventListener('resize', handleResize);

    return {
      searchQuery,
      isMobile,
      handleSearchInput,
      handleClearSearch,
    };
  },
  beforeUnmount() {
    window.removeEventListener('resize', this.handleResize);
  },
};
</script>

<style scoped>
/* Center icon and label in the bottom navigation items */
.v-btn > span {
  display: block;
  font-size: 12px;
}
</style>
