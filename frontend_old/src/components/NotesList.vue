<template>
  <v-container>
    <!-- Header Section with Add Note Button -->
    <v-row class="align-center justify-space-between mb-4">
      <v-col>
        <v-toolbar-title>{{ $t("notes.title") }}</v-toolbar-title>
      </v-col>
      <v-col class="text-right">
        <v-btn
          color="primary"
          @click="openAddNote"
          prepend-icon="mdi-note-plus-outline"
          >{{ $t("notes.add_note") }}</v-btn
        >
      </v-col>
    </v-row>

    <!-- Timeline of Notes -->
    <v-timeline density="compact" side="end">
      <v-timeline-item
        v-for="note in notes"
        :key="note.ID"
        dot-color="green lighten-3"
        icon="mdi-note-text"
      >
        <div class="timeline-date-section">
          <strong>{{ formatDate(note.date) }}</strong>
          <v-icon small class="edit-icon ml-2" @click="openEditNote(note)"
            >mdi-pencil</v-icon
          >
          <v-icon
            small
            class="delete-icon ml-2"
            color="error"
            @click="deleteNote(note.ID)"
            >mdi-delete</v-icon
          >
          <p>{{ note.content }}</p>
        </div>
      </v-timeline-item>
    </v-timeline>

    <!-- NoteAdd Component for Adding and Editing Notes -->
    <v-dialog v-model="showNoteDialog" max-width="500px" persistent>
      <noteAdd
        :noteId="editingNote ? editingNote.ID : null"
        :initialNote="editingNote || { content: '', date: new Date() }"
        @noteAdded="handleNoteAdded"
        @close="closeDialog"
      />
    </v-dialog>
  </v-container>
</template>

<script>
import noteService from "@/services/noteService";
import noteAdd from "@/components/NoteAdd.vue";
import { formatDate } from "@/utils/dateUtils";

export default {
  name: "NotesList",
  components: {
    noteAdd,
  },
  data() {
    return {
      notes: [], // List of notes
      showNoteDialog: false, // Controls visibility of the dialog
      editingNote: null, // Holds the note being edited, or null if adding a new note
    };
  },
  async mounted() {
    await this.fetchNotes();
  },
  methods: {
    async fetchNotes() {
      try {
        const response = await noteService.getUnassignedNotes();
        this.notes = (response.data || []).sort(
          (a, b) => new Date(b.date) - new Date(a.date)
        ); // Sort by date
      } catch (error) {
        console.error("Error fetching notes:", error);
      }
    },
    openAddNote() {
      this.editingNote = null; // Clear editing note for add mode
      this.showNoteDialog = true;
    },
    openEditNote(note) {
      this.editingNote = note; // Set the note to edit
      this.showNoteDialog = true;
    },
    handleNoteAdded(newNote) {
      if (this.editingNote) {
        // Find the index of the note being edited
        const index = this.notes.findIndex((n) => n.ID === this.editingNote.ID);
        if (index !== -1) {
          // Replace the existing note at the index with the updated note
          this.notes.splice(index, 1, { ...this.notes[index], ...newNote });
        }
      } else {
        // Add new note to the list
        this.notes.push(newNote);
      }

      // Sort notes in reverse chronological order
      this.notes.sort((a, b) => new Date(b.date) - new Date(a.date));

      this.closeDialog();
    },
    async deleteNote(noteId) {
      try {
        await noteService.deleteNote(noteId);
        this.notes = this.notes.filter((note) => note.ID !== noteId);
      } catch (error) {
        console.error("Error deleting note:", error);
      }
    },
    closeDialog() {
      this.showNoteDialog = false;
    },
    formatDate(date) {
      return formatDate(date); // Call the utility function
    },
  },
};
</script>

<style scoped>
.timeline-date-section {
  display: flex;
  align-items: center;
}

.edit-icon,
.delete-icon {
  opacity: 0;
  transition: opacity 0.3s ease;
  cursor: pointer;
}

.timeline-date-section:hover .edit-icon,
.timeline-date-section:hover .delete-icon {
  opacity: 1;
}
</style>
