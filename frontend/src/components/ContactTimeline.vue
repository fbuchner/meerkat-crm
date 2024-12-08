<template>
 <v-card outlined>
    <v-card-title>
      {{ $t('contacts.timeline') }}
      <v-spacer></v-spacer>
      <v-btn @click="openAddNote" color="primary" density="compact" prepend-icon="mdi-note-plus-outline">
        {{ $t('notes.add_note') }}
      </v-btn>
      <v-btn @click="openAddActivity" color="primary" density="compact" prepend-icon="mdi-account-multiple-plus-outline" class="ml-2">
        {{ $t('activities.add_activity') }}
      </v-btn>
    </v-card-title>
    <v-card-text>
      <v-timeline density="compact" side="end">
        <v-timeline-item
          v-for="item in timelineItems"
          :key="item.id"
          :dot-color="item.type === 'activity' ? 'blue lighten-3' : 'green lighten-3'"
          :icon="item.type === 'activity' ? 'mdi-calendar' : 'mdi-note-text'"
        >
          <div class="timeline-item">
            <div class="timeline-header">
              <strong>{{ item.date }}</strong>
              <div class="icon-group">
                <v-icon small class="edit-icon ml-2" @click="editItem(item)">mdi-pencil</v-icon>
                <v-icon small class="delete-icon ml-2" color="error" @click="deleteItem(item)">mdi-delete</v-icon>
              </div>
            </div>
            <div class="timeline-content">
              <template v-if="item.type === 'activity'">
                <h3 class="text-subtitle-1">{{ item.title }}</h3>
                <span v-if="item.location">{{ $t('contacts.activity_at') }} {{ item.location }}</span>
                <p>{{ item.description }}</p>
              </template>
              <template v-else>
                <p>{{ item.content }}</p>
              </template>
            </div>
          </div>
        </v-timeline-item>
      </v-timeline>
    </v-card-text>

    <!-- Dialog Modals -->
    <v-dialog v-model="showAddActivity" max-width="500px" persistent>
      <ActivityAdd
        :contactId="contactId"
        :activityId="editingActivityId"
        :initialActivity="editingActivityData || {}"
        @activityAdded="refreshTimeline"
        @close="closeActivityDialog"
      />
    </v-dialog>

    <v-dialog v-model="showAddNote" max-width="500px" persistent>
      <NoteAdd
        :contactId="contactId"
        :noteId="editingNoteId"
        :initialNote="editingNoteData || {}"
        @noteAdded="refreshTimeline"
        @close="closeNoteDialog"
      />
    </v-dialog>
  </v-card>
</template>

<script>
import ActivityAdd from '@/components/ActivityAdd.vue';
import NoteAdd from '@/components/NoteAdd.vue';
import activityService from '@/services/activityService';
import noteService from '@/services/noteService';

export default {
  name: 'ContactTimeline',
  props: {
    timelineItems: {
      type: Array,
      required: true,
    },
    contactId: {
      type: [String, Number],
      required: true,
    },
  },
  components: { ActivityAdd, NoteAdd },
  data() {
    return {
      showAddActivity: false,
      showAddNote: false,
      editingActivityId: null,
      editingActivityData: null,
      editingNoteId: null,
      editingNoteData: null,
    };
  },
  methods: {
    openAddActivity() {
      this.editingActivityId = null;
      this.editingActivityData = {};
      this.showAddActivity = true;
    },
    openAddNote() {
      this.editingNoteId = null;
      this.editingNoteData = {};
      this.showAddNote = true;
    },
    closeActivityDialog() {
      this.showAddActivity = false;
    },
    closeNoteDialog() {
      this.showAddNote = false;
    },
    async editItem(item) {
      if (item.type === 'activity') {
        this.editingActivityId = item.id;
        this.editingActivityData = { ...item }; // Clone item data
        this.showAddActivity = true;
      } else {
        this.editingNoteId = item.id;
        this.editingNoteData = { ...item }; // Clone item data
        this.showAddNote = true;
      }
    },
    async deleteItem(item) {
      try {
        if (item.type === 'activity') {
          await activityService.deleteActivity(item.id);
        } else {
          await noteService.deleteNote(item.id);
        }
        this.refreshTimeline();
      } catch (error) {
        console.error(`Error deleting ${item.type}:`, error);
      }
    },
    refreshTimeline() {
      this.$emit('refreshTimeline');
    },
  },
};
</script>

<style scoped>
/* Ensure proper spacing and alignment */
.timeline-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.edit-icon,
.delete-icon {
    opacity: 0;
    /* Hide icons by default */
    transition: opacity 0.3s ease;
    cursor: pointer;
}

.timeline-item:hover .edit-icon,
.timeline-item:hover .delete-icon {
    opacity: 1;
    /* Show icons on hover */
}

.field-label:hover .edit-icon {
    opacity: 1;
    /* Show on hover */
}

.timeline-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.icon-group {
  display: flex;
  gap: 8px;
}

.timeline-content {
  word-wrap: break-word;
}

/* Ensure icons are clickable and spaced correctly */
.edit-icon,
.delete-icon {
  cursor: pointer;
}
</style>