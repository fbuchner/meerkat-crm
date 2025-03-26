<template>
  <v-navigation-drawer
    app
    :permanent="!isMobileView"
    :temporary="isMobileView && !isNavOpen"
    :width="computedWidth"
    class="my-side-nav"
  >
    <v-list nav dense>
      <!-- Search Bar: Display only when nav is open -->
      <v-list-item v-if="isNavOpen" class="side-search">
        <v-text-field
          v-model="searchQueryLocal"
          :placeholder="$t('search.search_text')"
          hide-details
          clearable
          density="dense"
          @input="handleSearchInput"
          @click:clear="handleClearSearch"
        ></v-text-field>
      </v-list-item>

      <!-- Burger menu for mobile -->
      <v-list-item v-if="isMobileView" @click="toggleNav" class="burger-menu">
        <v-icon>
          {{ isNavOpen ? "mdi-chevron-left" : "mdi-menu" }}
        </v-icon>
      </v-list-item>

      <!-- Navigation Items -->
      <v-list-item class="nav-item" :to="{ path: '/contacts' }" link>
        <template #prepend>
          <v-icon>mdi-account-multiple</v-icon>
        </template>
        <v-list-item-title v-if="!isMobileView || isNavOpen">
          {{ $t("contacts.title") }}
        </v-list-item-title>
      </v-list-item>

      <v-list-item class="nav-item" :to="{ path: '/activities' }" link>
        <template #prepend>
          <v-icon>mdi-calendar-check</v-icon>
        </template>
        <v-list-item-title v-if="!isMobileView || isNavOpen">
          {{ $t("activities.title") }}
        </v-list-item-title>
      </v-list-item>

      <v-list-item class="nav-item" :to="{ path: '/notes' }" link>
        <template #prepend>
          <v-icon>mdi-note</v-icon>
        </template>
        <v-list-item-title v-if="!isMobileView | isNavOpen">
          {{ $t("notes.title") }}
        </v-list-item-title>
      </v-list-item>

      <!-- Language Menu -->
      <v-menu v-model="menuOpen" offset-y>
        <template v-slot:activator="{ props }">
          <v-list-item v-bind="props">
            <template #prepend>
              <v-icon>mdi-translate</v-icon>
            </template>
            <v-list-item-title>{{
              $t("settings.language.title")
            }}</v-list-item-title>
          </v-list-item>
        </template>
        <v-list>
          <v-list-item
            v-for="lang in languages"
            :key="lang"
            @click="selectLanguage(lang)"
          >
            <v-list-item-title>{{ lang }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>

      <!-- Logout Button -->
      <v-list-item @click="handleLogout" class="logout-button">
        <template #prepend>
          <v-icon>mdi-logout</v-icon>
        </template>
        <v-list-item-title v-if="!isMobileView || isNavOpen">
          {{ $t("user.logout") }}
        </v-list-item-title>
      </v-list-item>
    </v-list>
  </v-navigation-drawer>
</template>

<script>
import { ref, computed, onMounted, onBeforeUnmount, inject, watch } from "vue";
import { i18n } from "../main";
import { availableLanguages, loadLocaleMessages } from "@/locales";
import { useRouter } from "vue-router";

export default {
  name: "SideNav",
  emits: ["search", "resetFilters"],
  setup(_, { emit }) {
    function debounce(func, delay) {
      let timeout;
      return (...args) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => func(...args), delay);
      };
    }

    // Inject search state from parent (App.vue)
    const searchQuery = inject("searchQuery");
    const setSearchQuery = inject("setSearchQuery");
    const router = useRouter();

    // Local reactive search variable
    const searchQueryLocal = ref(searchQuery.value);

    const debouncedSearchInput = debounce(handleSearchInput, 300);
    function handleSearchInput() {
      if (!router.currentRoute.value.path.endsWith("/contacts")) {
        router.push("/contacts");
      }
      setSearchQuery(searchQueryLocal.value);
      emit("search", searchQueryLocal.value);
    }

    function handleClearSearch() {
      searchQueryLocal.value = "";
      setSearchQuery("");
      emit("resetFilters");
      emit("search", ""); // Emit a search event with an empty query when cleared
    }

    // Watch for changes to the injected searchQuery and update the local value
    watch(searchQuery, (newValue) => {
      searchQueryLocal.value = newValue;
    });

    function handleLogout() {
      localStorage.removeItem("token");
      router.push("/login");
    }

    // Language menu state and logic
    const menuOpen = ref(false);
    const languages = availableLanguages;

    async function selectLanguage(newLang) {
      await changeLanguage(newLang);
      menuOpen.value = false;
    }

    async function changeLanguage(newLang) {
      localStorage.setItem("preferredLanguage", newLang);
      const messagesLoaded = await loadLocaleMessages(i18n, newLang);
      // Set the new locale only after loading messages
      i18n.global.locale = newLang;
      if (messagesLoaded) {
        i18n.global.setLocaleMessage(newLang, messagesLoaded);
      }
    }

    // Mobile view management
    const isNavOpen = ref(false);
    const isMobileView = ref(false);

    function toggleNav() {
      isNavOpen.value = !isNavOpen.value;
    }

    function checkMobileView() {
      isMobileView.value = window.innerWidth < 600;
      if (!isMobileView.value) {
        isNavOpen.value = true;
      }
    }

    onMounted(() => {
      checkMobileView();
      if (!isMobileView.value) {
        isNavOpen.value = true;
      }
      window.addEventListener("resize", debounce(checkMobileView, 100));
    });

    onBeforeUnmount(() => {
      window.removeEventListener("resize", checkMobileView);
    });

    return {
      searchQueryLocal,
      handleSearchInput: debouncedSearchInput,
      handleClearSearch,
      menuOpen,
      languages,
      selectLanguage,
      isNavOpen,
      toggleNav,
      isMobileView,
      handleLogout,
      computedWidth: computed(() =>
        isMobileView.value ? (isNavOpen.value ? 200 : 0) : 200
      ),
    };
  },
};
</script>

<style scoped>
.my-side-nav {
  transition: width 0.3s ease;
}
.side-search {
  padding: 8px 16px;
}
.nav-item {
  cursor: pointer;
  transition: background-color 0.2s ease;
}
.nav-item:hover {
  background-color: rgba(0, 0, 0, 0.1);
}
.burger-menu {
  display: none;
}
@media (max-width: 600px) {
  .burger-menu {
    display: block;
  }
}
.v-list-item-title {
  white-space: nowrap;
}
.language-switcher {
  font-size: 0.75rem;
  padding: 0;
}
</style>
