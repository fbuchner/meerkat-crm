<template>
  <v-container>
    <v-row class="align-center justify-space-between mb-4">
      <v-col>
        <v-toolbar-title>Contacts</v-toolbar-title>
      </v-col>
      <v-col class="text-right">
        <v-btn color="primary" to="/add-contact">Add Contact</v-btn>
      </v-col>
    </v-row>

    <v-row class="mb-4">
      <v-col cols="12" sm="6">
        <v-text-field
          v-model="searchQuery"
          label="Search contacts..."
          clearable
        ></v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-btn-toggle v-model="activeCircle" class="ml-4">
          <v-btn
            v-for="circle in circles"
            :key="circle"
            @click="filterByCircle(circle)"
            :class="{ active: activeCircle === circle }"
          >
            {{ circle }}
          </v-btn>
          <v-btn @click="clearCircleFilter" :class="{ active: activeCircle === null }">All</v-btn>
        </v-btn-toggle>
      </v-col>
    </v-row>

    <v-list>
      <v-list-item
        v-for="contact in filteredContacts"
        :key="contact.ID"
        :to="{ name: 'ContactView', params: { ID: contact.ID } }"
        link
      >
        <v-list-item-title>
          {{ contact.firstname }} {{ contact.lastname }}
        </v-list-item-title>

        <v-list-item-action>
          <v-btn icon @click.stop="deleteContact(contact.ID)">
            <v-icon color="red">mdi-delete</v-icon>
          </v-btn>
        </v-list-item-action>
      </v-list-item>
    </v-list>

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
import contactService from '@/services/contactService';

export default {
  data() {
    return {
      contacts: [],
      circles: [],
      searchQuery: '',
      activeCircle: null, // Active circle filter
      page: 1,
      limit: 25,
      total: 0,
    };
  },
  computed: {
    filteredContacts() {
      return this.contacts.filter((contact) => {
        const matchesSearch = `${contact.firstname} ${contact.lastname}`
          .toLowerCase()
          .includes(this.searchQuery.toLowerCase());

        const matchesCircle =
          this.activeCircle === null ||
          (contact.circle && contact.circle === this.activeCircle);

        return matchesSearch && matchesCircle;
      });
    },
    totalPages() {
      return Math.ceil(this.total / this.limit);
    },
  },
  mounted() {
    this.loadContacts();
    this.loadCircles();
  },
  methods: {
    loadContacts() {
      contactService.getContacts(this.page, this.limit).then((response) => {
        this.contacts = response.data.contacts;
        this.total = response.data.total;
      });
    },
    loadCircles() {
      contactService.getCircles().then((response) => {
        this.circles = response.data;
      });
    },
    deleteContact(ID) {
      contactService.deleteContact(ID).then(() => {
        this.loadContacts();
      });
    },
    filterByCircle(circle) {
      this.activeCircle = circle;
      this.loadContacts();
    },
    clearCircleFilter() {
      this.activeCircle = null;
      this.loadContacts();
    },
  },
};
</script>
