<template>
  <v-app-bar app>
    <v-container>
      <v-row align="center" justify="space-between">
        <!-- Logo -->
        <v-col cols="auto">
          <h1>
            <v-btn text to="/" class="text-h5" @click="handleClearSearch"
              >perema</v-btn
            >
          </h1>
        </v-col>

        <!-- Search Bar (hidden on mobile) -->
        <v-col cols="3" class="d-none d-md-flex">
          <v-text-field
            v-model="searchQueryLocal"
            :placeholder="$t('search.search_text')"
            hide-details
            clearable
            density="compact"
            append-icon="mdi-magnify"
            autofocus
            @input="handleSearchInput"
            @click:clear="handleClearSearch"
          ></v-text-field>
        </v-col>

        <!-- Language Switcher -->
        <v-col cols="1" class="d-flex align-center">
          <v-menu v-model="menu" offset-y>
            <template v-slot:activator="{ props }">
              <v-btn text v-bind="props" class="language-switcher">
                {{ selectedLanguage }}
              </v-btn>
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
        </v-col>

        <!-- Desktop Navigation Links (hidden on mobile) -->
        <v-col cols="auto" class="d-none d-md-flex justify-end">
          <v-btn text to="/contacts">{{ $t("contacts.title") }}</v-btn>
          <v-btn text to="/activities">{{ $t("activities.title") }}</v-btn>
          <v-btn text to="/notes">{{ $t("notes.title") }}</v-btn>

          <!-- Logout Button -->
          <v-btn text @click="handleLogout">{{ $t("user.logout") }}</v-btn>
        </v-col>
      </v-row>
    </v-container>

    <!-- Bottom Navigation for Mobile -->
    <v-bottom-navigation v-if="isMobile" app color="primary">
      <v-btn icon to="/contacts">
        <v-icon>mdi-account-multiple</v-icon>
        <span>{{ $t("contacts.title") }}</span>
      </v-btn>
      <v-btn icon to="/activities">
        <v-icon>mdi-calendar-check</v-icon>
        <span>{{ $t("activities.title") }}</span>
      </v-btn>
      <v-btn icon to="/notes">
        <v-icon>mdi-note</v-icon>
        <span>{{ $t("notes.title") }}</span>
      </v-btn>
    </v-bottom-navigation>
  </v-app-bar>
</template>

<script>
import { inject, ref, watch, onMounted } from "vue";
import { useRouter } from "vue-router";
import { i18n } from "../main";
import { availableLanguages, loadLocaleMessages } from "@/locales";
export default {
  emits: ["search", "resetFilters"],
  setup(_, { emit }) {
    function debounce(func, delay) {
      let timeout;
      return (...args) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => func(...args), delay);
      };
    }

    const searchQuery = inject("searchQuery");
    const setSearchQuery = inject("setSearchQuery");
    const searchQueryLocal = ref(searchQuery.value);
    const router = useRouter();
    const selectedLanguage = ref(i18n.global.locale);
    const languages = availableLanguages;
    const menu = ref(false);

    function handleLogout() {
      localStorage.removeItem("token");
      router.push("/login");
    }

    function selectLanguage(newLang) {
      selectedLanguage.value = newLang;
      changeLanguage(newLang);
      menu.value = false;
    }

    async function changeLanguage(newLang) {
      localStorage.setItem("preferredLanguage", newLang);

      await loadLocaleMessages(i18n, newLang).then(() => {
        i18n.global.setLocaleMessage(
          newLang,
          i18n.global.getLocaleMessage(newLang)
        );
      });

      i18n.global.locale = newLang;
    }

    function handleSearchInput() {
      if (!router.currentRoute.value.path.startsWith("/contacts")) {
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

    const isMobile = ref(window.innerWidth <= 960);

    function handleResize() {
      isMobile.value = window.innerWidth <= 960;
    }

    function handleKeyPress(event) {
      if (document.activeElement === document.body && event.key.length === 1) {
        searchQueryLocal.value += event.key;
        handleSearchInput();
      }
    }

    onMounted(() => {
      window.addEventListener("resize", debounce(handleResize, 100));
      window.addEventListener("keypress", handleKeyPress);
    });
    return {
      searchQueryLocal,
      isMobile,
      handleSearchInput,
      handleClearSearch,
      selectedLanguage,
      changeLanguage,
      languages,
      selectLanguage,
      menu,
      handleLogout,
    };
  },
  beforeUnmount() {
    window.removeEventListener("resize", this.handleResize);
    window.removeEventListener("keypress", this.handleKeyPress);
  },
};
</script>

<style scoped>
.v-btn > span {
  display: block;
  font-size: 12px;
}
.language-switcher {
  font-size: 0.75rem;
  padding: 0;
}
</style>
