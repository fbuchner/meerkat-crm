<template>
  <v-container>
    <!-- Header -->
    <v-row class="align-center justify-space-between mb-4">
      <v-col>
        <v-toolbar-title>Contacts</v-toolbar-title>
      </v-col>
      <v-col class="text-right">
        <v-btn color="primary" to="/add-contact" prepend-icon="mdi-account-plus-outline">Add Contact</v-btn>
      </v-col>
    </v-row>

    <!-- Circle Filter -->
    <v-row class="mb-4">
      <v-col cols="12" sm="12">
        <v-btn-toggle v-model="activeCircle" class="ml-4">
          <v-btn v-for="circle in circles" :key="circle" @click="filterByCircle(circle)"
            :class="{ active: activeCircle === circle }">
            {{ circle }}
          </v-btn>
          <v-btn @click="clearCircleFilter" :class="{ active: activeCircle === null }">All</v-btn>
        </v-btn-toggle>
      </v-col>
    </v-row>

   <!-- Contact Cards -->
<v-row>
  <v-col
    v-for="contact in filteredContacts"
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
        <v-avatar size="80" class="mb-3">
          <v-img :src="`${backendURL}/contacts/${contact.ID}/profile_picture.jpg`" alt="Profile Photo"></v-img>
        </v-avatar>

        <!-- Contact Name -->
        <div class="contact-name">
          {{ contact.firstname }} {{ contact.lastname }}
        </div>

        <!-- Circles with Wrapping -->
        <div class="circle-chips mt-2">
          <v-chip
            v-for="circle in contact.circles"
            :key="circle"
            class="mr-2 mb-2"
            @click.stop="filterByCircle(circle)"
          >
            {{ circle }}
          </v-chip>
        </div>
      </v-card-text>
    </v-card>
  </v-col>
</v-row>

    <!-- Pagination -->
    <v-row justify="center" class="mt-4">
      <v-pagination v-model="page" :length="totalPages" @input="loadContacts"></v-pagination>
    </v-row>
  </v-container>
</template>

<script>
import { inject, computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import contactService from '@/services/contactService';
import { backendURL } from '@/services/api';

export default {
  setup() {
    const contacts = ref([]);
    const circles = ref([]);
    const activeCircle = ref(null);
    const searchQuery = inject('searchQuery');
    const page = ref(1);
    const limit = ref(25);
    const total = ref(0);
    const router = useRouter(); // Access router to navigate programmatically

    const filteredContacts = computed(() => {
      return contacts.value.filter((contact) => {
        const matchesSearch = `${contact.firstname} ${contact.lastname}`
          .toLowerCase()
          .includes(searchQuery.value.toLowerCase());

        const matchesCircle =
          activeCircle.value === null ||
          (contact.circles && contact.circles.includes(activeCircle.value));

        return matchesSearch && matchesCircle;
      });
    });

    const totalPages = computed(() => Math.ceil(total.value / limit.value));

    function loadContacts() {
      contactService.getContacts(page.value, limit.value).then((response) => {
        contacts.value = response.data.contacts;
        total.value = response.data.total;
      });
    }

    function loadCircles() {
      contactService.getCircles().then((response) => {
        circles.value = response.data;
      });
    }

    function filterByCircle(circle) {
      activeCircle.value = circle;
      loadContacts();
    }

    function clearCircleFilter() {
      activeCircle.value = null;
      loadContacts();
    }

    function goToContact(contactId) {
      // Programmatically navigate to the contact view
      router.push({ name: 'ContactView', params: { ID: contactId } });
    }

    return {
      contacts,
      circles,
      activeCircle,
      searchQuery,
      page,
      limit,
      total,
      filteredContacts,
      totalPages,
      loadContacts,
      loadCircles,
      filterByCircle,
      clearCircleFilter,
      goToContact,
      backendURL,
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

</style>