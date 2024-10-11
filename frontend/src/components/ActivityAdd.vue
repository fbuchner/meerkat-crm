<template>
    <div class="add-activity">
      <h3>Add an Activity</h3>
      <input v-model="newActivityName" placeholder="Activity name" />
      <input v-model="newActivityDate" type="date" />
      <button @click="addActivity">Add Activity</button>
    </div>
  </template>
  
  <script>

import activityService from '@/services/activityService';

  
  export default {
    name: 'AddActivity',
    props: {
      contactId: {
        type: Number,
        required: true,
      },
    },
    data() {
      return {
        newActivityName: '',
        newActivityDate: '',
      };
    },
    
    methods: {
        async addActivity() {
            try {
                await activityService.addActivity({
                    name: this.newActivityName,
                    date: this.newActivityDate,
                    contacts: [this.contactId],
                });
                this.newActivityName = ''; // Clear the field
                this.newActivityDate = '';
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
