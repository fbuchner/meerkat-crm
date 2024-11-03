<template>
  <v-container>
    <v-card>
      <v-card-title>Add an Activity</v-card-title>
      <v-card-text>
        <v-form @submit.prevent="addActivity">
          <v-text-field v-model="newActivityName" label="Activity Name" required></v-text-field>

          <v-textarea v-model="newActivityDescription" label="Activity Description" rows="3" auto-grow></v-textarea>

          <v-text-field v-model="newActivityLocation" label="Activity Location"></v-text-field>

          <v-dialog v-model="menu" max-width="290" persistent>
            <template v-slot:activator="{ props }">
              <v-text-field v-model="formattedActivityDate" label="Activity Date" prepend-icon="mdi-calendar" readonly
                v-bind="props" @click="menu = true"
                :rules="[v => !!newActivityDate || 'Activity date is required']"></v-text-field>
            </template>
            <v-date-picker v-model="newActivityDate" no-title @input="updateFormattedDate">
              <template v-slot:actions>
                <v-btn text color="primary" @click="menu = false">Cancel</v-btn>
                <v-btn text color="primary" @click="confirmDate">OK</v-btn>
              </template>
            </v-date-picker>
          </v-dialog>
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

export default {
  name: 'ActivityAdd',
  props: {
    contactId: {
      type: Number,
      required: true,
    },
    activityId: {
      type: Number,
      default: null, // Defaults to null when adding a new activity
    },
    initialActivity: {
      type: Object,
      default: () => ({ // Provide default structure when no data is passed
        title: '',
        description: '',
        date: new Date(),
        location: '',
      }),
    },
  },

  data() {
    return {
      newActivityName: this.initialActivity.title || '',
      newActivityDate: this.initialActivity.date ? new Date(this.initialActivity.date) : new Date(),
      formattedActivityDate: this.initialActivity.date ? this.formatDate(new Date(this.initialActivity.date)) : this.formatDate(new Date()),
      newActivityDescription: this.initialActivity.description || '',
      newActivityLocation: this.initialActivity.location || '',
      menu: false,
    };
  },

  watch: {
    newActivityDate(newDate) {
      this.formattedActivityDate = this.formatDate(newDate);
    },
  },
  methods: {
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
        contact_ids: [Number(this.contactId)],
      };

      try {
        if (this.activityId) {
          // Edit existing activity
          await activityService.updateActivity(this.activityId, activityData);
        } else {
          // Add new activity
          await activityService.addActivity(activityData);
        }

        this.resetForm();
        this.$emit('activityAdded'); // Emit event to refresh the contact
        this.$emit('close'); // Close dialog after adding or updating
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
    },
  },
};
</script>
