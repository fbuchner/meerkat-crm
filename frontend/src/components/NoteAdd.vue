<template>
    <v-container>
      <v-card>
        <v-card-title>Add a Note</v-card-title>
        <v-card-text>
          <v-form @submit.prevent="addNote">
            <v-textarea
              v-model="newNoteContent"
              label="Write a note..."
              auto-grow
              clearable
              required
            ></v-textarea>
  
            <v-btn type="submit" color="primary" class="mt-3">Add Note</v-btn>
          </v-form>
        </v-card-text>
      </v-card>
    </v-container>
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
  