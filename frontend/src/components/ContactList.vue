<template>
  <v-container>
    <!-- Header -->
    <v-row class="align-center justify-space-between mb-4">
      <v-col>
        <v-toolbar-title>Contacts</v-toolbar-title>
      </v-col>
      <v-col class="text-right">
        <v-btn color="primary" to="/add-contact">Add Contact</v-btn>
      </v-col>
    </v-row>

    <!-- Circle Filter -->
    <v-row class="mb-4">
      <v-col cols="12" sm="6">
        <!-- Removed the internal search bar as it's provided from App.vue -->
      </v-col>
      <v-col cols="12" sm="6">
        <v-btn-toggle v-model="activeCircle" class="ml-4">
          <v-btn v-for="circle in circles" :key="circle" @click="filterByCircle(circle)"
            :class="{ active: activeCircle === circle }">
            {{ circle }}
          </v-btn>
          <v-btn @click="clearCircleFilter" :class="{ active: activeCircle === null }">All</v-btn>
        </v-btn-toggle>
      </v-col>
    </v-row>

    <!-- Contact List -->
    <v-list>
      <v-list-item v-for="contact in filteredContacts" :key="contact.ID" class="contact-item">
        <!-- Profile Photo -->
        <v-avatar size="50">
          <v-img :src="contact.photo || '/placeholder-avatar.png'" alt="Profile Photo" class="circular-frame"></v-img>
        </v-avatar>

        <!-- Contact Details -->
        <div class="contact-details">
          <router-link :to="{ name: 'ContactView', params: { ID: contact.ID } }" class="contact-link">
            {{ contact.firstname }} {{ contact.lastname }}
          </router-link>

          <!-- Circles under the name -->
          <div class="contact-circles">
            <v-chip-group row>
              <v-chip v-for="circle in contact.circles" :key="circle" @click.stop="filterByCircle(circle)"
                class="mr-2 clickable-chip">
                {{ circle }}
              </v-chip>
            </v-chip-group>
          </div>
        </div>

        <!-- Delete Button -->
        <v-list-item-action>
          <v-btn icon @click.stop="deleteContact(contact.ID)">
            <v-icon color="red">mdi-delete</v-icon>
          </v-btn>
        </v-list-item-action>
      </v-list-item>
    </v-list>

    <!-- Pagination -->
    <v-pagination v-model="page" :length="totalPages.value" @input="loadContacts" />
  </v-container>
</template>

<script>
import { inject, computed, ref } from 'vue';
import contactService from '@/services/contactService';

export default {
  setup() {
    const contacts = ref([]);
    const circles = ref([]);
    const activeCircle = ref(null);
    const searchQuery = inject('searchQuery');
    const page = ref(1);
    const limit = ref(25);
    const total = ref(0);

    const filteredContacts = computed(() => {
      return contacts.value.filter((contact) => {
        const matchesSearch = `${contact.firstname} ${contact.lastname}`
          .toLowerCase()
          .includes(searchQuery.value.toLowerCase());

        const matchesCircle =
          activeCircle.value === null ||
          (contact.circle && contact.circle === activeCircle.value);

        return matchesSearch && matchesCircle;
      });
    });

    const totalPages = (() => Math.ceil(total.value / limit.value));

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

    function deleteContact(ID) {
      contactService.deleteContact(ID).then(() => {
        loadContacts();
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
      deleteContact,
      filterByCircle,
      clearCircleFilter,
    };
  },
  mounted() {
    this.loadContacts();
    this.loadCircles();
  },
};
</script>

<style scoped>
.contact-list-item {
  border-bottom: 1px solid #e0e0e0;
  padding: 10px 0;
}

.contact-link {
  text-decoration: none;
  color: inherit;
  font-weight: bold;
}

.circular-frame {
  border-radius: 50%;
}

.v-chip {
  cursor: pointer;
  transition: background-color 0.2s;
}

.v-chip:hover {
  background-color: rgba(0, 0, 0, 0.1);
}
</style>
