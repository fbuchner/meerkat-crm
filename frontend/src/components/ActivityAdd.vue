<template>
    <div class="add-activity">
      <h3>Add an Activity</h3>
      <input v-model="newActivityName" placeholder="Activity name" />
      <input v-model="newActivityDate" type="date" />
      <input v-model="newActivityDescription" placeholder="Activity description" />
      <input v-model="newActivityLocation" placeholder="Activity location" />
      <button @click="addActivity">Add Activity</button>
    </div>
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
  
  <style scoped>
  .add-activity {
    margin-top: 20px;
  }
  
  input {
    display: block;
    margin-bottom: 10px;
    padding: 5px;
    width: 100%;
  }
  </style>
