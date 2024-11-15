<template>
  <v-container>
    <v-card>
      <v-card-title>{{ activityId ? 'Edit Activity' : 'Add an Activity' }}</v-card-title>
      <v-card-text>
        <v-form @submit.prevent="addActivity">
          <!-- Activity Details -->
          <v-text-field v-model="newActivityName" label="Activity Name" required></v-text-field>
          <v-textarea v-model="newActivityDescription" label="Activity Description" rows="3" auto-grow></v-textarea>
          <v-text-field v-model="newActivityLocation" label="Activity Location"></v-text-field>

          <!-- Date Picker -->
          <v-dialog v-model="menu" max-width="290" persistent>
            <template v-slot:activator="{ props }">
              <v-text-field v-model="formattedActivityDate" label="Activity Date" prepend-icon="mdi-calendar" readonly
                v-bind="props" :rules="[v => !!newActivityDate || 'Activity date is required']"
                @click="menu = true"></v-text-field>
            </template>
            <v-date-picker v-model="newActivityDate" no-title @input="updateFormattedDate">
              <template v-slot:actions>
                <v-btn text color="primary" @click="menu = false">Cancel</v-btn>
                <v-btn text color="primary" @click="confirmDate">OK</v-btn>
              </template>
            </v-date-picker>
          </v-dialog>

          <!-- Contact Selector -->
          <v-autocomplete v-model="selectedContacts" :items="filteredContacts" item-title="name" item-value="ID"
            label="Select Contacts" chips closable-chips multiple outlined color="blue-grey-lighten-2">
            <!-- Chip Slot -->
            <template v-slot:chip="{ props, item }">
              <v-chip v-bind="props" outlined :prepend-avatar="item.photo || '../placeholder-avatar.png'" :text="item.name">
              </v-chip>
            </template>

            <!-- Dropdown Item Slot -->
            <template v-slot:item="{ props, item }">
              <v-list-item v-bind="props" :prepend-avatar="item.photo || '../placeholder-avatar.png'" :text="item.name"></v-list-item>
            </template>
          </v-autocomplete>
        </v-form>
      </v-card-text>

      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn text color="primary" @click="$emit('close')">Cancel</v-btn>
        <v-btn color="primary" @click="addActivity">{{ activityId ? 'Save Changes' : 'Add Activity' }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-container>
</template>
<script>

import activityService from '@/services/activityService';
import contactService from '@/services/contactService';

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
    await this.loadContacts();
    if (this.contactId) {
      this.preselectCurrentContact();
    }
  },
  methods: {
    async loadContacts() {
      try {
        //TODO use lazy loading based on query
        const response = await contactService.getContacts({
          fields: ['ID', 'photo', 'firstname', 'lastname'],
          limit: 5000,
        });
        this.allContactNames = response.data.contacts.map(contact => ({
          ID: contact.ID,
          photo: contact.photo,
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
    async addActivity() {
      const formattedDate = this.newActivityDate.toISOString().split('T')[0];
      const activityData = {
        title: this.newActivityName,
        description: this.newActivityDescription,
        date: formattedDate,
        location: this.newActivityLocation,
        contact_ids: this.selectedContacts
      };
      console.log('Activity Data:', activityData);
      try {
        let savedActivity;
        if (this.activityId) {
          savedActivity = await activityService.updateActivity(this.activityId, activityData);
        } else {
          savedActivity = await activityService.addActivity(activityData);
        }

        this.$emit('activityAdded', savedActivity.data);
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
};
</script>
