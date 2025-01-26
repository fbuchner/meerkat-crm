<template>
    <v-container v-if="contact">
        <!-- Top Section with Profile and Edit Button -->
        <v-row class="d-flex flex-column flex-md-row align-center text-center text-md-left">
            <v-col cols="12" md="3" class="d-flex justify-center">
                <!-- Profile Photo -->
                <ProfilePhoto :photo="contact.photo" :contactId="contact.ID" @photoUploaded="updatePhoto" />
            </v-col>

            <v-col cols="12" md="9" class="d-flex flex-column justify-center text-center text-md-left">
                <div class="d-flex align-center justify-center justify-md-start name-section field-label">
                    <!-- Contact Name with Edit Icon -->
                    <template v-if="!isEditingName">
                        <h1 class="text-h4 font-weight-bold">{{ contact.firstname }} {{ contact.lastname }}</h1>
                        <v-icon small class="edit-icon ml-2" @click="startEditingName">mdi-pencil</v-icon>
                        <v-icon small class="delete-icon ml-2" color="error" @click="deleteContact">mdi-delete</v-icon>
                    </template>
                    <template v-else>
                        <v-text-field v-model="editName" dense hide-details></v-text-field>
                        <v-icon small class="confirm-icon ml-2" @click="saveNameEdit">mdi-check</v-icon>
                        <v-icon small class="cancel-icon ml-2" @click="cancelNameEdit">mdi-close</v-icon>
                    </template>
                </div>

                <!-- Circles Section -->
                <div>
                    <!-- Display Existing Circles with Delete Option (Only if there are circles) -->
                    <v-chip-group v-if="contact.circles && contact.circles.length" column class="mt-2">
                        <v-chip v-for="(circle, index) in contact.circles" :key="index" class="mr-2" closable
                            @click:close="removeCircle(circle)">
                            {{ circle }}
                        </v-chip>
                    </v-chip-group>

                    <!-- Plus Icon to Toggle Add Circle Input (Always Visible) -->
                    <v-icon small class="add-circle-icon mt-2" @click="toggleAddCircle">
                        mdi-plus-circle
                    </v-icon>

                    <!-- Input Field for Adding New Circle with Add Button (Visible when showAddCircleInput is true) -->
                    <v-text-field v-if="showAddCircleInput" ref="addCircleInput" v-model="newCircle" label="Add Circle"
                        dense hide-details class="mt-2" @keydown.enter="addCircle" @blur="showAddCircleInput = false">
                        <!-- Add Button inside Text Field -->
                        <template v-slot:append-inner>
                            <v-btn icon @click="addCircle">
                                <v-icon>mdi-check</v-icon>
                            </v-btn>
                        </template>
                    </v-text-field>
                </div>

            </v-col>
        </v-row>

        <!-- Main Layout with Details and Timeline -->
        <v-row class="mt-4">
            <v-col cols="12" md="4">
                <RelationshipList :contactId="contact.ID" />

                <v-card outlined>
                    <v-card-title>{{ $t('contacts.contact_details') }}</v-card-title>
                    <v-card-text>
                        <v-list dense>
                            <v-list-item dense v-for="field in contactFieldSchema" :key="field.key" class="field-label">
                                <div>
                                    <strong>{{ field.label }}: </strong>
                                    <template v-if="!isEditing[field.key]">
                                        <span>
                                            {{ formatField(field, contact[field.key]) }}
                                            <v-icon small class="edit-icon ml-2" @click="startEditing(field.key)">
                                                mdi-pencil
                                            </v-icon>
                                        </span>
                                    </template>

                                </div>
                                <template v-if="isEditing[field.key]">
                                    <component :is="getFieldComponent(field)" v-model="editValues[field.key]"
                                        :items="field.options || []"
                                        :rules="field.key === 'birthday' ? [birthdayValidationRule] : []"
                                        :placeholder="field.key === 'birthday' ? 'DD.MM.YYYY or DD.MM.' : ''"
                                        density="compact" style="max-width: 300px; min-width: 200px; height: auto;">
                                    </component>
                                    <v-icon small class="confirm-icon ml-2"
                                        @click="saveEdit(field.key)">mdi-check</v-icon>
                                    <v-icon small class="cancel-icon ml-2"
                                        @click="cancelEdit(field.key)">mdi-close</v-icon>
                                </template>
                            </v-list-item>
                        </v-list>
                    </v-card-text>
                </v-card>
            </v-col>

            <!-- Right Column: Timeline for Notes and Activities -->
            <v-col cols="12" md="8">
                <v-tabs v-model="tab">
                    <v-tab value="one">{{ $t('contacts.timeline') }}</v-tab>
                    <v-tab value="two">{{ $t('contacts.reminders') }}</v-tab>
                </v-tabs>

                <v-card-text>
                    <v-tabs-window v-model="tab">
                        <v-tabs-window-item value="one">
                            <ContactTimeline :timelineItems="sortedTimelineItems" :contactId="contact.ID"
                                @refreshTimeline="fetchContact" />
                        </v-tabs-window-item>

                        <v-tabs-window-item value="two">
                            <ContactReminders :reminders="contact.reminders" :contactId="contact.ID"
                                @updateReminders="handleUpdateReminders" />
                        </v-tabs-window-item>
                    </v-tabs-window>
                </v-card-text>
            </v-col>
        </v-row>
    </v-container>
</template>

<script>
import contactService from '@/services/contactService';
import { reactive } from 'vue';
import ProfilePhoto from './ProfilePhoto.vue';
import activityService from '@/services/activityService';
import reminderService from '@/services/reminderService';
import RelationshipList from '@/components/RelationshipList.vue';
import ContactTimeline from '@/components/ContactTimeline.vue';
import ContactReminders from '@/components/ContactReminders.vue';


export default {
    name: 'ContactView',
    props: {
        ID: {
            required: true,
        },
    },
    components: { ProfilePhoto, RelationshipList, ContactTimeline, ContactReminders },
    data() {
        return {
            contact: null,
            isEditing: reactive({}),
            editValues: reactive({}),
            isEditingName: false,
            editName: '',
            newCircle: '', // Holds the new circle being added
            showAddCircleInput: false, // Controls visibility of the add circle input
            tab: null,
        };
    },
    computed: {
        contactDetails() {
            return {
                Nickname: this.contact.nickname,
                Gender: this.contact.gender,
                Birthday: this.contact.birthday,
                Email: this.contact.email,
                Phone: this.contact.phone,
                Address: this.contact.address,
                'How We Met': this.contact.how_we_met,
                'Food Preference': this.contact.food_preference,
                'Work Information': this.contact.work_information,
                'Additional Information': this.contact.contact_information,
            };
        },
        formattedBirthday() {
            if (!this.contact || !this.contact.birthday) return '';
            const [year, month, day] = this.contact.birthday.split('-');
            return `${day}.${month}.${year && year !== '0001' ? year : ''}`;
        },
        contactFieldSchema() {
            return [
                { key: "nickname", label: this.$t('contacts.contact_fields.nickname'), type: "text" },
                { key: "gender", label: this.$t('contacts.contact_fields.gender'), type: "select", options: this.$t('contacts.contact_fields.genders').split(',') },
                { key: "birthday", label: this.$t('contacts.contact_fields.birthday'), type: "date", format: "DD.MM.YYYY" },
                { key: "email", label: this.$t('contacts.contact_fields.email'), type: "email" },
                { key: "phone", label: this.$t('contacts.contact_fields.phone'), type: "tel" },
                { key: "address", label: this.$t('contacts.contact_fields.address'), type: "text" },
                { key: "how_we_met", label: this.$t('contacts.contact_fields.how_we_met'), type: "textarea" },
                { key: "food_preference", label: this.$t('contacts.contact_fields.food_preference'), type: "text" },
                { key: "work_information", label: this.$t('contacts.contact_fields.work_information'), type: "text" },
                { key: "contact_information", label: this.$t('contacts.contact_fields.additional_information'), type: "textarea" },
            ];
        },
        sortedTimelineItems() {
            const activities = (this.contact.activities || []).map(activity => ({
                id: activity.ID,
                type: 'activity',
                date: activity.date,
                title: activity.title,
                description: activity.description,
                location: activity.location,
            }));

            const notes = (this.contact.notes || []).map(note => ({
                id: note.ID,
                type: 'note',
                date: note.date,
                content: note.content,
            }));

            return [...activities, ...notes].sort((a, b) => new Date(b.date) - new Date(a.date));
        },

    },
    mounted() {
        this.fetchContact();
    },
    methods: {
        async fetchContact() {
            try {
                const response = await contactService.getContact(this.ID);
                this.contact = response.data;
                this.editValues = { ...this.contact };
                if (!this.contact.circles) {
                    this.contact.circles = [];
                }
                this.editName = `${this.contact.firstname} ${this.contact.lastname}`;
            } catch (error) {
                console.error('Error fetching contact:', error);
            }
        },
        startEditingName() {
            this.isEditingName = true;
        },
        startEditing(key) {
            this.isEditing[key] = true;
            if (key === 'birthday') {
                this.editValues[key] = this.formattedBirthday;
            } else {
                this.editValues[key] = this.contact[key];
            }
        },
        saveEdit(key) {
            if (key === 'birthday') {
                const datePattern = /^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])\.(\d{4})?$/;
                const match = this.editValues[key].match(datePattern);
                if (match) {
                    const [, day, month, year] = match;
                    this.contact[key] = `${year || '0001'}-${month}-${day}`;
                } else {
                    console.warn("Invalid birthday format:", this.editValues[key]);
                    return; // Abort saving
                }
            } else {
                this.contact[key] = this.editValues[key];
            }
            this.isEditing[key] = false;
            contactService.updateContact(this.ID, { [key]: this.contact[key] });
        },
        cancelEdit(key) {
            this.isEditing[key] = false;
            this.editValues[key] = this.contactDetails[key]; // Revert to the original value
        },
        async saveNameEdit() {
            const [firstname, lastname] = this.editName.split(' ');
            try {
                await contactService.updateContact(this.ID, { ...this.contact, firstname, lastname });
                this.contact.firstname = firstname || this.contact.firstname;
                this.contact.lastname = lastname || this.contact.lastname;
                this.isEditingName = false;
            } catch (error) {
                console.error('Error updating name:', error);
            }
        },
        cancelNameEdit() {
            this.isEditingName = false;
            this.editName = `${this.contact.firstname} ${this.contact.lastname}`;
        },
        async deleteContact() {
            try {
                await contactService.deleteContact(this.contact.ID);
                // Route to the main page after successfully deleting the contact
                this.$router.push('/');
            } catch (error) {
                console.error('Error deleting contact:', error);
            }
        },
        async deleteActivity(activityId) {
            try {
                await activityService.deleteActivity(activityId);
                this.refreshContact(); // Refresh contact details after deletion
            } catch (error) {
                console.error('Error deleting activity:', error);
            }
        },
        // Refresh the contact details after adding, editing, or deleting
        refreshContact() {
            this.fetchContact();
        },
        // Toggle the add circle input visibility
        toggleAddCircle() {
            this.showAddCircleInput = !this.showAddCircleInput;

            if (this.showAddCircleInput) {
                // Use $nextTick to wait for the DOM to render the input
                this.$nextTick(() => {
                    if (this.$refs.addCircleInput) {
                        this.$refs.addCircleInput.focus();
                    }
                });
            }
        },

        // Add a New Circle
        async addCircle() {
            const trimmedCircle = this.newCircle.trim();
            if (!trimmedCircle) return;

            // Ensure circles is initialized as an array if it's null or undefined
            if (!this.contact.circles) {
                this.contact.circles = [];
            }

            try {
                // Add the new circle to the backend
                await contactService.updateContact(this.ID, { circles: [...this.contact.circles, trimmedCircle] });

                // Update the local contact data and reset input
                this.contact.circles.push(trimmedCircle);
                this.newCircle = '';
                this.showAddCircleInput = false;
            } catch (error) {
                console.error('Error adding circle:', error);
            }
        },

        // Remove a Circle
        async removeCircle(circle) {
            const updatedCircles = this.contact.circles.filter((c) => c !== circle);

            try {
                // Update the backend with the new list of circles
                await contactService.updateContact(this.ID, { circles: updatedCircles });

                // Update the local contact data
                this.contact.circles = updatedCircles;
            } catch (error) {
                console.error('Error removing circle:', error);
            }
        },
        formatField(field, value) {
            if (field.key === 'birthday' && value) {
                const [year, month, day] = value.split('-');
                return `${day}.${month}.${year !== '0001' ? year : ''}`;
            }
            return value;
        },
        getFieldComponent(field) {
            switch (field.type) {
                case 'select': return 'v-select';
                case 'textarea': return 'v-textarea';
                case 'email': case 'tel': case 'text': default: return 'v-text-field';
            }
        },
        birthdayValidationRule(value) {
            const datePattern = /^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])\.(\d{4})?$/;
            if (!datePattern.test(value)) {
                return this.$t('contacts.birthday.birthday_warning');
            }
            return true; // No error
        },
        updatePhoto(newPhotoUrl) {
            this.contact.photo = newPhotoUrl;
        },
        async handleUpdateReminders({ action, reminder, reminderId }) {
            try {
                if (action === 'add') {
                    const response = await reminderService.addReminder(this.contact.ID, reminder);
                    this.contact.reminders.push(response.data); // Add to the local state
                } else if (action === 'edit') {
                    await reminderService.updateReminder(this.contact.ID, reminder.id, reminder);
                    const index = this.contact.reminders.findIndex(r => r.id === reminder.id);
                    if (index !== -1) {
                        this.$set(this.contact.reminders, index, reminder); // Update the local state
                    }
                } else if (action === 'delete') {
                    await reminderService.deleteReminder(this.contact.ID, reminderId);
                    this.contact.reminders = this.contact.reminders.filter(r => r.id !== reminderId); // Update the local state
                }
            } catch (error) {
                console.error('Error handling reminders:', error);
            }
        },
    },
};
</script>

<style scoped>
.field-label {
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.edit-icon,
.delete-icon {
    opacity: 0;
    /* Hide icons by default */
    transition: opacity 0.3s ease;
    cursor: pointer;
}

.field-label:hover .edit-icon,
.field-label:hover .delete-icon {
    opacity: 1;
    /* Show icons on hover */
}

.field-label:hover .edit-icon {
    opacity: 1;
    /* Show on hover */
}


.circular-frame {
    border-radius: 50%;
    background-color: #f0f0f0;
    border: 2px solid #ccc;
}

.fixed-square {
    width: 150px;
    height: 150px;
    max-width: 150px;
    max-height: 150px;
}
</style>
