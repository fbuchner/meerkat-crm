<template>
  <div>
    <h1>Contacts</h1>
    <ul>
      <li v-for="contact in contacts" :key="contact.ID">
        <router-link :to="{ name: 'ContactView', params: { ID: contact.ID } }">
          {{ contact.firstname }} {{ contact.lastname }}
        </router-link>
        <button @click="deleteContact(contact.ID)">Delete</button>
      </li>
    </ul>
  </div>
</template>

<script>
import contactService from '@/services/contactService';

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
      contactService.getContacts().then((response) => {
        console.log(response); // Add this to check the structure of the response
        this.contacts = response.data;
      });
    },
    deleteContact(ID) {
      contactService.deleteContact(ID).then(() => {
        this.loadContacts();
      });
    },
  },
};
</script>

<style scoped>
ul {
  list-style: none;
  padding: 0;
}

li {
  margin-bottom: 10px;
}
</style>