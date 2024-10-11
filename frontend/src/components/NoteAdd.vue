<template>
    <div class="add-note">
      <h3>Add a Note</h3>
      <textarea v-model="newNoteContent" placeholder="Write a note..."></textarea>
      <button @click="addNote">Add Note</button>
    </div>
  </template>
  
  <script>

import noteService from '@/services/noteService';
  
export default {
    name: 'AddNote',
    props: {
        contactId: {
            type: Number,
            required: true,
        },
    },
    data() {
        return {
            newNoteContent: '',
        };
    },
    methods: {
        async addNote() {
            try {
                await noteService.addNote(this.contactId, this.newNoteContent);
                this.newNoteContent = ''; // Clear the field
                this.$emit('noteAdded'); // Emit event to refresh contact
            } catch (error) {
                console.error('Error adding note:', error);
            }
        },
    },
};
  </script>
  
  <style scoped>
  .add-note {
    margin-top: 20px;
  }
  
  textarea {
    width: 100%;
    height: 80px;
    margin-bottom: 10px;
  }
  </style>