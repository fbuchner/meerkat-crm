<template>
  <v-card outlined>
    <v-card-title>
      <v-btn color="primary" density="compact" prepend-icon="mdi-plus" @click="openAddDialog">
        {{ $t('reminders.add_reminder') }}
      </v-btn>
    </v-card-title>
    <v-card-text>
      <div v-if="reminders.length === 0">
        {{ $t('reminders.no_reminders') }}
      </div>
      <v-list>
        <v-list-item v-for="reminder in reminders" :key="reminder.id" class="reminder-item">
          <template #title>
            <strong>{{ reminder.message }}</strong>
          </template>
          <template #subtitle>
            {{ reminder.recurrence || 'One-time' }} | {{ formatDate(reminder.remind_at) }}
          </template>
          <template #append>
            <v-icon small class="edit-icon" @click="openEditDialog(reminder)">mdi-pencil</v-icon>
            <v-icon small class="delete-icon" color="error" @click="deleteReminder(reminder.id)">mdi-delete</v-icon>
          </template>
        </v-list-item>
      </v-list>
    </v-card-text>

    <!-- Dialog for Adding/Editing Reminders -->
    <v-dialog v-model="showDialog" max-width="500px">
      <v-card>
        <v-card-title>{{ isEditing ? $t('reminders.edit_reminder') : $t('reminders.add_reminder') }}</v-card-title>
        <v-card-text>
          <v-form ref="reminderForm">
            <v-text-field label="Message" v-model="form.message" required
              :rules="[v => !!v || $t('reminders.validation.required')]" />
            <v-switch label="Send by Email" v-model="form.by_mail" />
            <v-menu v-model="menu" :close-on-content-click="false" transition="scale-transition" offset-y
              max-width="290px" min-width="auto">
              <template #activator="{ props }">
                <v-text-field v-bind="props" v-model="formattedRemindAt" label="Remind At" readonly />
              </template>
              <v-date-picker v-model="form.remind_at" no-title @input="menu = false" />
            </v-menu>
            <v-select label="Recurrence" :items="recurrenceOptions" v-model="form.recurrence" />
            <v-switch label="Reoccur from Completion" v-model="form.reoccur_from_completion" />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn text @click="closeDialog">{{ $t('buttons.cancel') }}</v-btn>
          <v-btn color="primary" @click="saveReminder">{{ $t('buttons.add') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-card>
</template>


<script>
export default {
  name: "ContactReminders",
  props: {
    reminders: {
      type: Array,
      required: true,
      default: () => [],
    },
    contactId: {
      type: [String, Number],
      required: true,
    },
  },
  data() {
    return {
      showDialog: false,
      isEditing: false,
      editingReminderId: null,
      form: {
        message: "",
        by_mail: false,
        remind_at: new Date().toISOString().split('T')[0],
        recurrence: null,
        reoccur_from_completion: true,
      },
      menu: false,
      recurrenceOptions: ["Daily", "Weekly", "Monthly", "Yearly"],
    };
  },
  computed: {
    formattedRemindAt() {
      return this.form.remind_at
        ? new Date(this.form.remind_at).toLocaleDateString()
        : "";
    },
  },
  methods: {
    openAddDialog() {
      this.isEditing = false;
      this.resetForm();
      this.showDialog = true;
    },
    openEditDialog(reminder) {
      this.isEditing = true;
      this.editingReminderId = reminder.id;
      this.form = { ...reminder };
      this.showDialog = true;
    },
    closeDialog() {
      this.showDialog = false;
    },
    resetForm() {
      this.form = {
        message: "",
        by_mail: false,
        remind_at: new Date().toISOString().split('T')[0],
        recurrence: null,
        reoccur_from_completion: true,
      };
    },
    async saveReminder() {
      const formValid = this.$refs.reminderForm.validate();
      if (!formValid) return;

      const newReminder = { ...this.form };

      if (this.isEditing) {
        // Update existing reminder
        this.$emit("updateReminders", {
          action: "edit",
          reminder: { ...newReminder, id: this.editingReminderId },
        });
      } else {
        // Add new reminder
        this.$emit("updateReminders", {
          action: "add",
          reminder: newReminder,
        });
      }
      this.closeDialog();
    },
    async deleteReminder(reminderId) {
      this.$emit("updateReminders", {
        action: "delete",
        reminderId,
      });
    },
  },
};
</script>

<style scoped>
.reminder-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.edit-icon,
.delete-icon {
  cursor: pointer;
}
</style>