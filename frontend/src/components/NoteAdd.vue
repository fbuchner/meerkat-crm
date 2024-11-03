<template>
  <v-container>
    <v-card>
      <v-card-title>{{ noteId ? 'Edit Note' : 'Add a Note' }}</v-card-title>
      <v-card-text>
        <v-form @submit.prevent="saveNote">
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
        <v-btn color="primary" @click="saveNote">{{ noteId ? 'Save Changes' : 'Add Note' }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-container>
</template>

<script>
import noteService from '@/services/noteService';

export default {
  name: 'NoteAdd',
  props: {
    contactId: { type: Number, required: true },
    noteId: { type: Number, default: null },
    initialNote: { type: Object, default: () => ({ content: '', date: new Date() }) },
  },
  data() {
    return {
      newNoteContent: this.initialNote.content || '',
      newNoteDate: this.initialNote.date ? new Date(this.initialNote.date) : new Date(),
      formattedNoteDate: this.initialNote.date ? this.formatDate(new Date(this.initialNote.date)) : this.formatDate(new Date()),
      menu: false,
    };
  },
  watch: {
    newNoteDate(newDate) {
      this.formattedNoteDate = this.formatDate(newDate);
    },
  },
  methods: {
    formatDate(date) {
      return date ? new Intl.DateTimeFormat('de-DE').format(date) : '';
    },
    updateFormattedDate() {
      this.formattedNoteDate = this.formatDate(this.newNoteDate);
    },
    confirmDate() {
      this.menu = false;
    },
    async saveNote() {
      const formattedDate = this.newNoteDate.toISOString().split('T')[0];
      const noteData = {
        content: this.newNoteContent,
        date: formattedDate,
        contact_id: this.contactId,
      };

      try {
        if (this.noteId) {
          // Update the existing note
          await noteService.updateNote(this.noteId, noteData);
        } else {
          // Add a new note
          await noteService.addNote(noteData);
        }

        this.resetForm();
        this.$emit('noteAdded'); // Refresh notes
        this.$emit('close'); // Close dialog
      } catch (error) {
        console.error('Error saving note:', error);
      }
    },
    resetForm() {
      this.newNoteContent = '';
      this.newNoteDate = new Date();
      this.formattedNoteDate = this.formatDate(new Date());
    },
  },
};
</script>
