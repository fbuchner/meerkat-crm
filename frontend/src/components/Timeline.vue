<template>
    <v-card outlined>
      <v-card-title>
        {{ $t('contacts.timeline') }}
        <v-spacer></v-spacer>
        <v-btn @click="$emit('addNote')" color="primary" density="compact" prepend-icon="mdi-note-plus-outline">
          {{ $t('notes.add_note') }}
        </v-btn>
        <v-btn @click="$emit('addActivity')" color="primary" density="compact" prepend-icon="mdi-account-multiple-plus-outline" class="ml-2">
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
            <div class="timeline-date-section">
              <strong>{{ item.date }}</strong>
              <v-icon small class="edit-icon ml-2" @click="editItem(item)">{{ item.type === 'activity' ? 'mdi-pencil' : 'mdi-pencil' }}</v-icon>
              <v-icon small class="delete-icon ml-2" color="error" @click="deleteItem(item)">{{ item.type === 'activity' ? 'mdi-delete' : 'mdi-delete' }}</v-icon>
              <template v-if="item.type === 'activity'">
                <h3 class="text-subtitle-1">{{ item.title }}</h3>
                <p>{{ item.description }}<span v-if="item.location"> at {{ item.location }}</span></p>
              </template>
              <template v-else>
                <p>{{ item.content }}</p>
              </template>
            </div>
          </v-timeline-item>
        </v-timeline>
      </v-card-text>
    </v-card>
  </template>
  
  <script>
  export default {
    name: 'Timeline',
    props: {
      timelineItems: {
        type: Array,
        required: true,
      },
    },
    methods: {
      editItem(item) {
        this.$emit(item.type === 'activity' ? 'editActivity' : 'editNote', item.id);
      },
      deleteItem(item) {
        this.$emit(item.type === 'activity' ? 'deleteActivity' : 'deleteNote', item.id);
      },
    },
  };
  </script>
  
  <style scoped>
  .timeline-date-section {
    display: flex;
    flex-direction: column;
  }
  .edit-icon,
  .delete-icon {
    cursor: pointer;
  }
  </style>
  