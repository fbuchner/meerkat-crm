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

    <v-dialog v-model="showDialog" max-width="500px">
      <v-card>
        <v-card-title>
          {{ isEditing ? $t('reminders.edit_reminder') : $t('reminders.add_reminder') }}
        </v-card-title>

        <v-card-text>
          <v-form ref="reminderForm">
            <v-text-field
              label="Message"
              v-model="form.message"
              required
              :rules="[v => !!v || $t('reminders.validation.required')]"
            />

            <v-switch label="Send by Email" v-model="form.by_mail" />

            <v-dialog v-model="menu" max-width="290" persistent>
              <template v-slot:activator="{ props }">
                <v-text-field
                  v-model="formattedRemindAt"
                  label="Remind At"
                  prepend-icon="mdi-calendar"
                  readonly
                  v-bind="props"
                  @click="menu = true"
                  :rules="[v => !!form.remind_at || $t('reminders.validation.required')]"
                />
              </template>
              <v-date-picker v-model="newReminderDate" no-title @input="updateFormattedRemindAt">
                <template v-slot:actions>
                  <v-btn text color="primary" @click="menu = false">{{ $t('buttons.cancel') }}</v-btn>
                  <v-btn text color="primary" @click="confirmDate">{{ $t('buttons.ok') }}</v-btn>
                </template>
              </v-date-picker>
            </v-dialog>

            <v-select
              label="Recurrence"
              :items="$t('reminders.recurrence').split(',')"
              v-model="form.recurrence"
            />

            <v-switch label="Reoccur from Completion" v-model="form.reoccur_from_completion" />
          </v-form>
        </v-card-text>

        <v-card-actions>
          <v-spacer />
          <v-btn text @click="closeDialog">{{ $t('buttons.cancel') }}</v-btn>
          <v-btn color="primary" @click="saveReminder">
            {{ isEditing ? $t('buttons.save_changes') : $t('reminders.add_reminder') }}
          </v-btn>
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
      menu: false,
      isEditing: false,
      editingReminderId: null,
      newReminderDate: new Date(), // Initialize to today
      form: {
        message: "",
        by_mail: false,
        remind_at: new Date(), // Initialize to today
        recurrence: null,
        reoccur_from_completion: true,
      },
      formattedRemindAt: this.formatDate(new Date()), // Initialize formatted date
    };
  },

  methods: {
    formatDate(date) {
      return date ? new Intl.DateTimeFormat('de-DE').format(date) : '';
    },
    updateFormattedRemindAt() {
      this.formattedRemindAt = this.formatDate(this.newReminderDate);
    },
    confirmDate() {
      this.form.remind_at = this.newReminderDate;
      this.menu = false;
    },

    openAddDialog() {
      this.isEditing = false;
      this.resetForm();
      this.showDialog = true;
    },

    openEditDialog(reminder) {
      this.isEditing = true;
      this.editingReminderId = reminder.id;

      // Copy data into form and ensure remind_at is a Date.  Handle potential null values.
      this.form = {
        ...reminder,
        remind_at: reminder.remind_at ? new Date(reminder.remind_at) : new Date(),
      };
      this.formattedRemindAt = this.formatDate(this.form.remind_at); // Update formatted date

      this.showDialog = true;
    },

    closeDialog() {
      this.showDialog = false;
    },

    resetForm() {
      this.form = {
        message: "",
        by_mail: false,
        remind_at: new Date(),
        recurrence: null,
        reoccur_from_completion: true,
      };
      this.formattedRemindAt = this.formatDate(new Date()); // Reset formatted date
      this.newReminderDate = new Date(); // Reset the date picker too
    },

    async saveReminder() {
      const formValid = this.$refs.reminderForm.validate();
      if (!formValid) return;

      const newReminder = {
        ...this.form,
        remind_at: this.form.remind_at.toISOString(), // Always convert to ISO string
      };

      if (this.isEditing) {
        this.$emit("updateReminders", {
          action: "edit",
          reminder: { ...newReminder, id: this.editingReminderId },
        });
      } else {
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

  watch: {
    menu(value) {
      if (value) {
        this.newReminderDate = this.form.remind_at || new Date(); // Use form value or today
      }
    },
    newReminderDate(newDate) {
      this.formattedRemindAt = this.formatDate(newDate);
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
  margin-left: 8px;
}
</style>
