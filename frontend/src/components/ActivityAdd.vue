<template>
  <v-container>
    <v-row>
      <v-col>
        <v-card>
          <v-card-title>Add an Activity</v-card-title>
          <v-card-text>
            <v-form @submit.prevent="addActivity">
              <v-text-field
                v-model="newActivityName"
                label="Activity Name"
                required
              ></v-text-field>

              <v-menu
                ref="menu"
                v-model="menu"
                :close-on-content-click="false"
                :nudge-right="40"
                lazy
                transition="scale-transition"
                offset-y
                min-width="auto"
              >
                <template v-slot:activator="{ on, attrs }">
                  <v-text-field
                    v-model="newActivityDate"
                    label="Activity Date"
                    prepend-icon="mdi-calendar"
                    readonly
                    v-bind="attrs"
                    v-on="on"
                  ></v-text-field>
                </template>
                <v-date-picker
                  v-model="newActivityDate"
                  no-title
                  @input="menu = false"
                ></v-date-picker>
              </v-menu>

              <v-text-field
                v-model="newActivityDescription"
                label="Activity Description"
              ></v-text-field>

              <v-text-field
                v-model="newActivityLocation"
                label="Activity Location"
              ></v-text-field>

              <v-btn type="submit" color="primary">Add Activity</v-btn>
            </v-form>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

  
  <script>

import activityService from '@/services/activityService';

  
  export default {
    name: 'AddActivity',
    props: {
      contactId: {
        required: true,
      },
    },
    data() {
      const today = new Date().toISOString().split('T')[0]; // Get today's date in 'YYYY-MM-DD' format
      return {
        newActivityName: '',
        newActivityDate: today,
        newActivityDescription: '',
        newActivityLocation: '',
      };
    },
    
    methods: {
        async addActivity() {
            try {
                await activityService.addActivity({
                    name: this.newActivityName,
                    description: this.newDescription,
                    date: this.newActivityDate,
                    location: this.newActivityLocation,
                    contact_ids: [Number(this.contactId)],
                });
                this.newActivityName = ''; // Clear the field
                this.newActivityDate = '';
                this.newActivityDescription = ''; 
                this.newActivityLocation = '';
                this.$emit('activityAdded'); // Emit event to refresh contact
            } catch (error) {
                console.error('Error adding activity:', error);
            }
        },
    },
  };
  </script>
  