<!-- src/components/ContactList.vue -->
<template>
  <div>
    <h1>Contacts</h1>
    <ul>
      <li v-for="contact in contacts" :key="contact.id">
        {{ contact.firstname }} {{ contact.lastname }}
        <button @click="deleteContact(contact.id)">Delete</button>
      </li>
    </ul>
  </div>
</template>

<script>
import contactsService from '@/services/contactService';

export default {
  data() {
    return {
      contacts: [],
    };
  },
  mounted() {
    this.loadContacts();
  },
  methods: {
    loadContacts() {
      contactsService.getContacts().then((response) => {
        this.contacts = response.data;
      });
    },
    deleteContact(id) {
      contactsService.deleteContact(id).then(() => {
        this.loadContacts();
      });
    },
  },
};
</script>
