<template>
  <v-navigation-drawer
    app
    :permanent="!isMobileView"
    :temporary="isMobileView && !isNavOpen"
    :width="isMobileView ? (isNavOpen ? 200 : 0) : isExpanded ? 200 : 60"
    class="my-side-nav"
    @mouseenter="expandNav"
    @mouseleave="collapseNav"
  >
    <v-list nav dense>
      <v-list-item v-if="isMobileView" @click="toggleNav" class="burger-menu">
        <v-icon>
          {{ isNavOpen ? "mdi-chevron-left" : "mdi-menu" }}
        </v-icon>
      </v-list-item>

      <!-- Contacts List Item -->
      <v-list-item class="nav-item" :to="{ path: '/contacts' }" link>
        <template #prepend>
          <v-icon>
            {{
              isExpanded || isMobileView
                ? "mdi-account-multiple"
                : "mdi-account-multiple-outline"
            }}
          </v-icon>
        </template>
        <v-list-item-title v-if="isExpanded || isMobileView">{{
          $t("contacts.title")
        }}</v-list-item-title>
      </v-list-item>

      <!-- Activities List Item -->
      <v-list-item class="nav-item" :to="{ path: '/activities' }" link>
        <template #prepend>
          <v-icon>
            {{
              isExpanded || isMobileView
                ? "mdi-calendar-check"
                : "mdi-calendar-check-outline"
            }}
          </v-icon>
        </template>
        <v-list-item-title v-if="isExpanded || isMobileView">{{
          $t("activities.title")
        }}</v-list-item-title>
      </v-list-item>

      <!-- Notes List Item -->
      <v-list-item class="nav-item" :to="{ path: '/notes' }" link>
        <template #prepend>
          <v-icon>
            {{ isExpanded || isMobileView ? "mdi-note" : "mdi-note-outline" }}
          </v-icon>
        </template>
        <v-list-item-title v-if="isExpanded || isMobileView">{{
          $t("notes.title")
        }}</v-list-item-title>
      </v-list-item>
    </v-list>
  </v-navigation-drawer>
</template>

<script>
export default {
  name: "SideNav",
  data() {
    return {
      isExpanded: false,
      isNavOpen: false, // Track the state of the nav on mobile view
      isMobileView: false, // Track if we are in mobile view
    };
  },
  methods: {
    expandNav() {
      this.isExpanded = true;
    },
    collapseNav() {
      this.isExpanded = false;
    },
    toggleNav() {
      this.isNavOpen = !this.isNavOpen; // Toggle sidebar visibility
    },
    checkMobileView() {
      this.isMobileView = window.innerWidth < 600; // Adjust the breakpoint as necessary
    },
  },
  mounted() {
    this.checkMobileView(); // Check on mount
    window.addEventListener("resize", this.checkMobileView); // Listen for window resize
  },
  beforeUnmount() {
    window.removeEventListener("resize", this.checkMobileView); // Clean up the listener
  },
};
</script>

<style scoped>
.my-side-nav {
  transition: width 0.3s ease;
}

.nav-item {
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.nav-item:hover {
  background-color: rgba(0, 0, 0, 0.1);
}

.burger-menu {
  display: none; /* Hidden by default */
}

/* Show the burger menu in mobile view */
@media (max-width: 600px) {
  .burger-menu {
    display: block; /* Show the burger menu */
  }
}

.v-list-item-title {
  white-space: nowrap;
}
</style>
