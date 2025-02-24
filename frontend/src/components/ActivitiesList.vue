<template>
  <v-container>
    <!-- Header Section with Add Activity Button -->
    <v-row class="align-center justify-space-between mb-4">
      <v-col>
        <v-toolbar-title>{{ $t("activities.title") }}</v-toolbar-title>
      </v-col>
      <v-col class="text-right">
        <v-btn
          color="primary"
          @click="openAddActivity"
          prepend-icon="mdi-account-multiple-plus-outline"
        >
          {{ $t("activities.add_activity") }}
        </v-btn>
      </v-col>
    </v-row>

    <!-- Timeline of Activities -->
    <v-timeline density="compact" side="end">
      <v-timeline-item
        v-for="activity in activities"
        :key="activity.ID"
        dot-color="blue lighten-3"
        icon="mdi-calendar"
      >
        <div class="timeline-date-section">
          <strong>{{ formatDate(activity.date) }}</strong>
          <v-icon
            small
            class="edit-icon ml-2"
            @click="openEditActivity(activity)"
            >mdi-pencil</v-icon
          >
          <v-icon
            small
            class="delete-icon ml-2"
            color="error"
            @click="deleteActivity(activity.ID)"
            >mdi-delete</v-icon
          >

          <div>
            <h3 class="text-subtitle-1">
              <strong>{{ activity.title }}</strong>
            </h3>

            <!-- Display contacts if they exist and are populated -->
            <p v-if="activity.contacts && activity.contacts.length">
              with
              <span
                v-for="(contact, index) in activity.contacts"
                :key="contact.ID"
              >
                <router-link :to="`/contacts/${contact.ID}`"
                  >{{ contact.firstname }} {{ contact.lastname }}</router-link
                >
                <span v-if="index < activity.contacts.length - 1">, </span>
              </span>
            </p>

            <p>{{ activity.description }}</p>
          </div>
        </div>
      </v-timeline-item>
    </v-timeline>

    <!-- Pagination Controls -->
    <v-pagination
      v-model="page"
      :length="totalPages"
      @input="fetchActivities(page)"
      class="mt-4"
    ></v-pagination>

    <!-- ActivityAdd Component for Adding and Editing Activities -->
    <v-dialog v-model="showActivityDialog" max-width="500px" persistent>
      <ActivityAdd
        :activityId="editingActivity ? editingActivity.ID : null"
        :initialActivity="
          editingActivity || {
            title: '',
            description: '',
            date: new Date(),
            location: '',
            contacts: [],
          }
        "
        @activityAdded="handleActivityAdded"
        @close="closeDialog"
      />
    </v-dialog>
  </v-container>
</template>

<script>
import activityService from "@/services/activityService";
import ActivityAdd from "@/components/ActivityAdd.vue";
import { formatDate } from "@/utils/dateUtils";

export default {
  name: "ActivitiesList",
  components: {
    ActivityAdd,
  },
  data() {
    return {
      activities: [],
      showActivityDialog: false,
      editingActivity: null,
      page: 1,
      limit: 25,
      total: 0,
    };
  },
  computed: {
    totalPages() {
      return Math.ceil(this.total / this.limit);
    },
  },
  async mounted() {
    await this.fetchActivities(this.page);
  },
  methods: {
    async fetchActivities(page = 1) {
      try {
        const response = await activityService.getAllActivities(
          page,
          this.limit
        );
        this.activities = response.data.activities || [];
        this.total = response.data.total;
        this.page = response.data.page;
      } catch (error) {
        console.error("Error fetching activities:", error);
      }
    },
    openAddActivity() {
      this.editingActivity = null;
      this.showActivityDialog = true;
    },
    openEditActivity(activity) {
      this.editingActivity = activity;
      this.showActivityDialog = true;
    },
    handleActivityAdded(newActivity) {
      // Check if this is an edit or a new activity
      if (this.editingActivity) {
        const index = this.activities.findIndex(
          (activity) => activity.ID === newActivity.ID
        );
        if (index !== -1) {
          this.activities.splice(index, 1, newActivity); // Replace the existing activity
        }
      } else {
        this.activities.unshift(newActivity);
        this.total += 1;
      }
      // Sort activities by date in descending order
      this.activities.sort((a, b) => new Date(b.date) - new Date(a.date));
      this.closeDialog();
    },
    async deleteActivity(activityId) {
      try {
        await activityService.deleteActivity(activityId);
        this.activities = this.activities.filter(
          (activity) => activity.ID !== activityId
        );
      } catch (error) {
        console.error("Error deleting activity:", error);
      }
    },
    closeDialog() {
      this.showActivityDialog = false;
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
