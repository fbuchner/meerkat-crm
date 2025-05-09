<template>
  <v-container>
    <!-- Header -->
    <v-row class="align-center justify-space-between mb-4">
      <v-col>
        <v-toolbar-title>{{ $t("contacts.title") }}</v-toolbar-title>
      </v-col>
      <v-col class="text-right">
        <v-btn
          color="primary"
          to="/add-contact"
          prepend-icon="mdi-account-plus-outline"
          >{{ $t("contacts.add_contact") }}</v-btn
        >
      </v-col>
    </v-row>

    <!-- Circle Filter -->
    <v-row class="mb-4">
      <v-col cols="12">
        <v-chip
          class="mr-2"
          outlined
          clickable
          @click="clearCircleFilter"
          :class="{ 'active-circle': activeCircle === null }"
        >
          {{ $t("contacts.circles.all_circles") }}
        </v-chip>
        <v-chip
          v-for="circle in circles"
          :key="circle"
          class="mr-2"
          outlined
          clickable
          @click="filterByCircle(circle)"
          :class="{ 'active-circle': activeCircle === circle }"
        >
          {{ circle }}
        </v-chip>
      </v-col>
    </v-row>

    <!-- Display Current Search Query -->
    <v-row v-if="searchQuery && searchQuery.trim()" class="mb-4">
      <v-col cols="12">
        <v-alert
          type="info"
          border="start"
          class="d-flex align-center"
          @click="clearSearch"
        >
          {{ $t("search.show_results") }} <strong>"{{ searchQuery }}"</strong>
        </v-alert>
      </v-col>
    </v-row>

    <!-- Contact Cards -->
    <v-row>
      <v-col
        v-for="contact in contacts"
        :key="contact.ID"
        cols="12"
        sm="6"
        md="4"
        lg="3"
      >
        <v-card
          class="contact-card"
          outlined
          elevation="2"
          @click="goToContact(contact.ID)"
        >
          <v-card-text>
            <!-- Profile Photo -->
            <v-row justify="center" class="mb-3">
              <div class="profile-picture">
                <ProfilePicture
                  :contactId="contact.ID"
                  width="80"
                  height="80"
                />
              </div>

              <!-- Contact Name -->
            </v-row>
            <v-row justify="center" class="mb-3">
              <div class="contact-name">
                {{ contact.firstname }} {{ contact.lastname }}
              </div>
            </v-row>

            <!-- Circles with Wrapping -->
            <div class="circle-chips mt-2">
              <v-chip
                v-for="circle in contact.circles"
                :key="circle"
                class="mr-2 mb-2 clickable-chip"
                @click.stop="filterByCircle(circle)"
              >
                {{ circle }}
              </v-chip>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- No Contacts Found -->
    <v-row
      v-if="contacts.length === 0 && showNoContactsMessage"
      justify="center"
      class="mt-4"
    >
      <v-col cols="12" class="text-center">
        <v-alert type="warning" border="start" class="d-flex align-center">
          {{ $t("search.no_results") }}
        </v-alert>
      </v-col>
    </v-row>

    <!-- Pagination -->
    <v-row justify="center" class="mt-4">
      <v-pagination
        v-model="page"
        :length="totalPages"
        @input="loadContacts"
      ></v-pagination>
    </v-row>
  </v-container>
</template>

<script>
import { inject, computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import contactService from "@/services/contactService";
import ProfilePicture from "@/components/ProfilePicture.vue";

export default {
  components: {
    ProfilePicture,
  },
  setup() {
    const contacts = ref([]);
    const circles = ref([]);
    const activeCircle = ref(null);
    const searchQuery = inject("searchQuery");
    const clearFilters = inject("clearFilters");
    const page = ref(1);
    const limit = ref(25);
    const total = ref(0);
    const router = useRouter(); // Access router to navigate programmatically
    const setSearchQuery = inject("setSearchQuery");

    const showNoContactsMessage = ref(false);
    let timeoutId = null;

    function debounce(func, delay) {
      let timeout;
      return (...args) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => func(...args), delay);
      };
    }

    function clearSearch() {
      setSearchQuery("");
      page.value = 1; // Reset to the first page
      debouncedLoadContacts();
    }

    const totalPages = computed(() => Math.ceil(total.value / limit.value));

    function loadContacts() {
      const search = searchQuery.value ? searchQuery.value.trim() : "";
      const circle = activeCircle.value ? activeCircle.value.trim() : "";

      contactService
        .getContacts({
          fields: [
            "ID",
            "firstname",
            "lastname",
            "nickname",
            "email",
            "circles",
          ],
          search: search,
          circle: circle,
          page: page.value,
          limit: 25,
        })
        .then((response) => {
          contacts.value = response.data.contacts;
          total.value = response.data.total;

          if (contacts.value.length === 0) {
            timeoutId = setTimeout(() => {
              showNoContactsMessage.value = true;
            }, 500);
          } else {
            // Clear the timeout if contacts are found
            if (timeoutId) {
              clearTimeout(timeoutId);
            }
            showNoContactsMessage.value = false;
          }
        })
        .catch((error) => {
          console.error("Failed to fetch contacts:", error);
        });
    }

    // Debounced version of loadContacts
    const debouncedLoadContacts = debounce(loadContacts, 300);

    // Watch searchQuery for changes and trigger debounced loading
    watch(searchQuery, () => {
      page.value = 1; // Reset to the first page
      debouncedLoadContacts();
    });

    function loadCircles() {
      contactService.getCircles().then((response) => {
        circles.value = response.data;
      });
    }

    function filterByCircle(circle) {
      activeCircle.value = circle;
      page.value = 1; // Reset to the first page
      debouncedLoadContacts();
    }

    function clearCircleFilter() {
      activeCircle.value = null;
      page.value = 1; // Reset to the first page
      debouncedLoadContacts();
    }

    function goToContact(contactId) {
      // Programmatically navigate to the contact view
      router.push({ name: "ContactView", params: { ID: contactId } });
    }

    watch(clearFilters, (newValue) => {
      if (newValue) {
        clearCircleFilter();
      }
    });

    return {
      contacts,
      circles,
      activeCircle,
      searchQuery,
      page,
      limit,
      total,
      totalPages,
      loadContacts,
      loadCircles,
      filterByCircle,
      clearCircleFilter,
      goToContact,
      clearSearch,
      debouncedLoadContacts,
      showNoContactsMessage,
    };
  },
  mounted() {
    this.loadContacts();
    this.loadCircles();
  },
};
</script>

<style scoped>
.contact-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding-top: 16px;
  padding-bottom: 16px;
  position: relative;
  cursor: pointer;
  /* Show pointer cursor for clickable card */
}

.contact-name {
  font-weight: 600;
  font-size: 1.1rem;
}

.clickable-chip {
  cursor: pointer;
  user-select: none;
}

.v-avatar img {
  object-fit: cover;
}

.circle-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.active-circle {
  background-color: #1976d2;
  color: white;
}
</style>
