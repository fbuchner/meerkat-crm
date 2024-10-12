<template>
  <div class="contacts-container">
    <div class="header">
      <h1>Contacts</h1>
      <router-link to="/add-contact" class="add-contact-button">Add Contact</router-link>
    </div>

    <div class="search-and-circles">
      <input
        type="text"
        v-model="searchQuery"
        placeholder="Search contacts..."
        class="search-input"
      />
      <div class="circles">
        <button
          v-for="circle in circles"
          :key="circle"
          @click="filterByCircle(circle)"
          class="circle-button"
          :class="{ active: activeCircle === circle }"
        >
          {{ circle }}
        </button>
        <button
          @click="clearCircleFilter"
          class="circle-button"
          :class="{ active: activeCircle === null }"
        >
          All
        </button>
      </div>
    </div>

    <ul class="contacts-list">
      <li
        v-for="contact in filteredContacts"
        :key="contact.ID"
        class="contact-item"
      >
        <router-link
          :to="{ name: 'ContactView', params: { ID: contact.ID } }"
          class="contact-link"
        >
          <div class="contact-info">
            {{ contact.firstname }} {{ contact.lastname }}
          </div>
        </router-link>
        <button @click.stop="deleteContact(contact.ID)" class="delete-button">
          Delete
        </button>
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
      circles: [],
      searchQuery: '',
      activeCircle: null, // Active circle filter
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
  },
  mounted() {
    this.loadContacts();
  },
  methods: {
    loadContacts() {
      contactService.getContacts().then((response) => {
        this.contacts = response.data;
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
    },
    clearCircleFilter() {
      this.activeCircle = null;
    },
  },
};
</script>

<style scoped>
.contacts-container {
  max-width: 800px;
  margin: 2rem auto;
  padding: 1rem;
  background-color: #ffffff;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  border-radius: 8px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.add-contact-button {
  text-decoration: none;
  background-color: #3ca20c;
  color: white;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  font-weight: bold;
  transition: background-color 0.3s;
}

.add-contact-button:hover {
  background-color: #1e2d8c;
}


.search-and-circles {
  margin-bottom: 1rem;
}

.search-input {
  width: calc(100% - 16px);
  padding: 0.5rem;
  margin-bottom: 1rem;
  border-radius: 4px;
  border: 1px solid #e5e7eb;
}

.circles {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.circle-button {
  background-color: #e5e7eb;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.3s;
}

.circle-button:hover {
  background-color: #d1d5db;
}

.circle-button.active {
  background-color: #1e2d8c;
  color: white;
}

.contacts-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.contact-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  margin-bottom: 0.5rem;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  transition: background-color 0.3s, box-shadow 0.3s;
}

.contact-link {
  text-decoration: none;
  color: inherit;
  flex-grow: 1;
}

.contact-item:hover {
  background-color: #f9fafb;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.contact-info {
  width: 100%;
}

.delete-button {
  background-color: #ef4444;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.3s;
}

.delete-button:hover {
  background-color: #1e2d8c;
}
</style>
