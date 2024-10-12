<template>
  <header class="app-header">
    <div class="header-content">
      <div class="logo">
        <h1><router-link to="/" class="nav-item">perema</router-link></h1>
      </div>
      <div class="search-bar">
        <input
          type="text"
          placeholder="Search contacts..."
          v-model="searchQuery"
          @input="searchContacts"
        />
      </div>
      <nav class="nav-links">
        <router-link to="/contacts" class="nav-item">Contacts</router-link>
        <router-link to="/activities" class="nav-item">Activities</router-link>
        <router-link to="/notes" class="nav-item">Notes</router-link>
      </nav>
    </div>
  </header>
</template>

<script>
export default {
  data() {
    return {
      searchQuery: "",
    };
  },
  methods: {
    searchContacts() {
      // Emit search query to parent or handle search logic here
      this.$emit('search', this.searchQuery);
    },
  },
};
</script>

<style scoped>
.app-header {
  background-color: #1e2d8c;
  padding: 1rem;
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
}

.logo h1 {
  color: white;
  font-size: 1.5rem;
  margin: 0;
}

.search-bar {
  flex-grow: 1;
  max-width: 300px;
  margin: 0 1rem;
}

.search-bar input {
  width: 100%;
  padding: 0.5rem;
  border: none;
  border-radius: 4px;
}

.nav-links {
  display: flex;
  gap: 1rem;
  margin-left: auto;
}

.nav-item {
  color: white;
  text-decoration: none;
  font-weight: bold;
}

.nav-item:hover {
  text-decoration: underline;
}

/* Responsive Design */
@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    align-items: flex-start;
  }

  .search-bar {
    width: 100%;
    margin: 0.5rem 0;
  }

  .nav-links {
    width: 100%;
    justify-content: space-around;
  }
}
</style>
