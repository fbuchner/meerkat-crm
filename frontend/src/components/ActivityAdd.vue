<template>
  <v-container>
    <v-card>
      <v-card-title>{{ activityId ? $t('activities.edit_activity') : $t('activities.add_activity') }}</v-card-title>
      <v-card-text>
        <v-form @submit.prevent="addActivity">
          <!-- Activity Details -->
          <v-text-field v-model="newActivityName" :label="$t('activities.activity_name')"
            :rules="[() => !!newActivityName || $t('activities.activity_name_required')]"></v-text-field>
          <v-textarea v-model="newActivityDescription" :label="$t('activities.activity_description')" rows="3" auto-grow></v-textarea>
          <v-text-field v-model="newActivityLocation" :label="$t('activities.activity_location')"></v-text-field>

          <!-- Date Picker -->
          <v-dialog v-model="menu" max-width="290" persistent>
            <template v-slot:activator="{ props }">
              <v-text-field v-model="formattedActivityDate" :label="$t('activities.activity_date')" prepend-icon="mdi-calendar" readonly
                v-bind="props" @click="menu = true"
                :rules="[v => !!newActivityDate || $t('activities.activity_date_required')]"></v-text-field>
            </template>
            <v-date-picker v-model="newActivityDate" no-title @input="updateFormattedDate">
              <template v-slot:actions>
                <v-btn text color="primary" @click="menu = false">{{ $t('buttons.cancel') }}</v-btn>
                <v-btn text color="primary" @click="confirmDate">{{ $t('buttons.ok') }}</v-btn>
              </template>
            </v-date-picker>
          </v-dialog>


          <!-- Contact Selector -->
          <v-autocomplete v-model="selectedContacts" :items="filteredContacts" item-title="name" item-value="ID"
            :label="$t('activities.select_contacts')" chips closable-chips multiple outlined color="blue-grey-lighten-2"
            v-model:search-input="searchContactQuery">
            <!-- Chip Slot -->
            <template v-slot:chip="{ props, item }">
              <v-chip v-bind="props" outlined :prepend-avatar="getAvatarURL(item.value)" :text="item.title">
              </v-chip>
            </template>

            <!-- Dropdown Item Slot -->
            <template v-slot:item="{ props, item }">
              <v-list-item v-bind="props" :prepend-avatar="getAvatarURL(item.value)" :text="item.title"></v-list-item>
            </template>
          </v-autocomplete>
        </v-form>
      </v-card-text>

      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn text color="primary" @click="$emit('close')">{{ $t('buttons.cancel') }}</v-btn>
        <v-btn color="primary" @click="addActivity">{{ activityId ? $t('buttons.save_changes') : $t('activities.add_activity') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-container>
</template>
<script>

import activityService from '@/services/activityService';
import contactService from '@/services/contactService';
import { backendURL } from '@/services/api';

export default {
  name: 'ActivityAdd',
  props: {
    contactId: {
      type: Number,
      required: false,
    },
    activityId: {
      type: Number,
      default: null,
    },
    initialActivity: {
      type: Object,
      default: () => ({
        title: '',
        description: '',
        date: new Date(),
        location: '',
        contact_ids: [],
      }),
    },
  },
  data() {
    return {
      newActivityName: this.initialActivity.title || '',
      newActivityDate: this.initialActivity.date ? new Date(this.initialActivity.date) : new Date(),
      formattedActivityDate: this.initialActivity.date
        ? this.formatDate(new Date(this.initialActivity.date))
        : this.formatDate(new Date()),
      newActivityDescription: this.initialActivity.description || '',
      newActivityLocation: this.initialActivity.location || '',
      menu: false,
      selectedContacts: this.initialActivity.contact_ids || [], // Array of selected contact objects
      allContactNames: [], // Array of all available contacts
      backendURL,
      searchContactQuery: '',
      debouncedLoadContacts: null,
    };
  },
  computed: {
    filteredContacts() {
      // Exclude already selected contacts from the list
      const selectedIds = this.selectedContacts.map(contact => contact.ID);
      const filtered = this.allContactNames.filter(contact => !selectedIds.includes(contact.ID));
      return filtered;
    },
  },
  async mounted() {
    await this.debouncedLoadContacts();
    if (this.contactId) {
      this.preselectCurrentContact();
    }
  },
  watch: {
    newActivityDate(newDate) {
      this.formattedActivityDate = this.formatDate(newDate);
    },
    searchContactQuery(query) {
      if (this.debouncedLoadContacts) {
        this.debouncedLoadContacts(query);
      }
    },
  },
  methods: {
    debounce(func, delay) {
      let timeout;
      return (...args) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => func(...args), delay);
      };
    },
    async loadContacts(searchQuery = '') {
      try {
        const response = await contactService.getContacts({
          fields: ['ID', 'firstname', 'lastname', 'nickname'],
          search: searchQuery,
          limit: 15,
        });
        this.allContactNames = response.data.contacts.map(contact => ({
          ID: contact.ID,
          name: `${contact.firstname} ${contact.lastname}`,
        }));
      } catch (error) {
        console.error('Error fetching contacts:', error);
      }
    },
    preselectCurrentContact() {
      const currentContact = this.allContactNames.find(contact => contact.ID === this.contactId);
      if (currentContact && !this.selectedContacts.some(c => c.ID === currentContact.ID)) {
        this.selectedContacts.push(currentContact);
      }
    },
    formatDate(date) {
      return date ? new Intl.DateTimeFormat('de-DE').format(date) : '';
    },
    updateFormattedDate() {
      this.formattedActivityDate = this.formatDate(this.newActivityDate);
    },
    confirmDate() {
      this.menu = false;
    },
    getAvatarURL(ID) {
      return `${this.backendURL}/contacts/${ID}/profile_picture.jpg`;
    },
    async addActivity() {
      const formattedDate = this.newActivityDate.toISOString().split('T')[0];
      const activityData = {
        title: this.newActivityName,
        description: this.newActivityDescription,
        date: formattedDate,
        location: this.newActivityLocation,
        contact_ids: this.selectedContacts
      };
      try {
        let savedActivity;
        if (this.activityId) {
          savedActivity = await activityService.updateActivity(this.activityId, activityData);
        } else {
          savedActivity = await activityService.addActivity(activityData);
        }
        this.$emit('activityAdded', savedActivity.data.activity);
        this.$emit('close');
        this.resetForm();
      } catch (error) {
        console.error('Error saving activity:', error);
      }
    },
    resetForm() {
      this.newActivityName = '';
      this.newActivityDate = new Date();
      this.newActivityDescription = '';
      this.newActivityLocation = '';
      this.formattedActivityDate = this.formatDate(new Date());
      this.selectedContacts = [];
    },
  },
  created() {
      this.debouncedLoadContacts = this.debounce(this.loadContacts, 300);
      this.loadContacts(); // Initial load of contacts
      if (this.contactId) {
        this.preselectCurrentContact();
      }
    },
};
</script>
