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

          <v-dialog v-model="menu" max-width="290" persistent>
            <template v-slot:activator="{ props }">
              <v-text-field
                v-model="formattedNoteDate"
                label="Note Date"
                prepend-icon="mdi-calendar"
                readonly
                v-bind="props"
                @click="menu = true"
                :rules="[v => !!newNoteDate || 'Note date is required']"
              ></v-text-field>
            </template>
            <v-date-picker
              v-model="newNoteDate"
              no-title
              @input="updateFormattedDate"
            >
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
        <v-btn color="primary" @click="addNote">Add Note</v-btn> <!-- Call addNote directly -->
      </v-card-actions>
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
      newNoteDate: new Date(), // Initialize as Date object for today's date
      formattedNoteDate: this.formatDate(new Date()), // Initialize formatted date for display
      menu: false, // Controls the visibility of the date picker dialog
    };
  },
  watch: {
    newNoteDate(newDate) {
      this.formattedNoteDate = this.formatDate(newDate); // Update display date when date changes
    },
  },
  methods: {
    formatDate(date) {
      return date ? new Intl.DateTimeFormat('de-DE').format(date) : ''; // Format as "DD/MM/YYYY" or similar
    },
    updateFormattedDate() {
      this.formattedNoteDate = this.formatDate(this.newNoteDate); // Update the display date
    },
    confirmDate() {
      this.menu = false; // Close the date picker dialog
    },
    async addNote() {
      try {
        const formattedDate = this.newNoteDate.toISOString().split('T')[0]; // Format date as "YYYY-MM-DD" for API
        await noteService.addNote(this.contactId, {
          content: this.newNoteContent,
          date: formattedDate,
          contact_id: this.contactId,
        });
        this.newNoteContent = ''; // Clear the content field
        this.newNoteDate = new Date(); // Reset the date to today's date
        this.$emit('noteAdded'); // Emit event to refresh the notes
        this.$emit('close'); // Close dialog after adding the note
      } catch (error) {
        console.error('Error adding note:', error);
      }
    },
  },
};
</script>
