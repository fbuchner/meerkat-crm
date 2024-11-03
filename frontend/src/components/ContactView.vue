<template>
    <v-container v-if="contact">
        <!-- Top Section with Profile and Edit Button -->
        <v-row class="d-flex flex-column flex-md-row align-center text-center text-md-left">
            <v-col cols="12" md="3" class="d-flex justify-center">
                <v-img :src="contact.photo || ''" alt="Profile Photo" width="150" height="150"
                    class="circular-frame mb-2 fixed-square" contain></v-img>
            </v-col>
            <v-col cols="12" md="9" class="d-flex flex-column justify-center text-center text-md-left">
                <div class="d-flex align-center justify-center justify-md-start name-section field-label">
                    <!-- Contact Name with Edit Icon -->
                    <template v-if="!isEditingName">
                        <h1 class="text-h4 font-weight-bold">{{ contact.firstname }} {{ contact.lastname }}</h1>
                        <v-icon small class="edit-icon ml-2" @click="startEditingName">mdi-pencil</v-icon>
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
            <!-- Left Column: Details -->
            <v-col cols="12" md="4">
                <v-card outlined>
                    <v-card-title>Contact Details</v-card-title>
                    <v-card-text>
                        <v-list dense>
                            <v-list-item v-for="(value, key) in contactDetails" :key="key" class="field-label">
                                <div>
                                    <strong>{{ key }}: </strong>
                                    <span v-if="!isEditing[key]" @click="startEditing(key)">
                                        {{ value }}
                                        <v-icon small class="edit-icon"
                                            @click.stop="startEditing(key)">mdi-pencil</v-icon>
                                    </span>
                                    <div v-else class="edit-field">
                                        <v-text-field v-model="editValues[key]" dense hide-details></v-text-field>
                                        <v-icon small class="confirm-icon" @click="saveEdit(key)">mdi-check</v-icon>
                                        <v-icon small class="cancel-icon" @click="cancelEdit(key)">mdi-close</v-icon>
                                    </div>
                                </div>
                            </v-list-item>
                        </v-list>
                    </v-card-text>
                </v-card>
            </v-col>

            <!-- Right Column: Timeline for Notes and Activities -->
            <v-col cols="12" md="8">
                <v-card outlined>
                    <v-card-title>
                        Timeline
                        <v-spacer></v-spacer>
                        <v-btn @click="openAddNote" density="compact" prepend-icon="mdi-note-plus-outline">Add
                            note</v-btn>
                        <v-btn @click="openAddActivity" density="compact"
                            prepend-icon="mdi-account-multiple-plus-outline" class="ml-2">Add activity</v-btn>
                    </v-card-title>
                    <v-card-text>
                        <v-timeline density="compact" side="end">
                            <v-timeline-item v-for="item in sortedTimelineItems" :key="item.id"
                                :dot-color="item.type === 'activity' ? 'blue lighten-3' : 'green lighten-3'"
                                :icon="item.type === 'activity' ? 'mdi-calendar' : 'mdi-note-text'">

                                <div class="timeline-date-section" v-if="item.type === 'activity'">
                                    <strong>{{ item.date }}</strong>
                                    <v-icon small class="edit-icon ml-2"
                                        @click="editActivity(item.id)">mdi-pencil</v-icon>
                                    <v-icon small class="delete-icon ml-2" color="error"
                                        @click="deleteActivity(item.id)">mdi-delete</v-icon>
                                    <h3 class="text-subtitle-1">{{ item.title }}</h3>
                                    <p>{{ item.description }}<span v-if="item.location"> at {{ item.location }}</span>
                                    </p>
                                </div>

                                <div class="timeline-date-section" v-else>
                                    <strong>{{ item.date }}</strong><v-icon small class="edit-icon ml-2"
                                        @click="editNote(item.id)">mdi-pencil</v-icon>
                                    <v-icon small class="delete-icon ml-2" color="error"
                                        @click="deleteNote(item.id)">mdi-delete</v-icon>
                                    <p>{{ item.content }}</p>
                                </div>
                            </v-timeline-item>
                        </v-timeline>
                    </v-card-text>
                </v-card>
            </v-col>
        </v-row>

        <!-- Dialog Modals for Adding Activity and Note -->
        <v-dialog v-model="showAddActivity" max-width="500px" persistent>
            <ActivityAdd :contactId="contact.ID" :activityId="editingActivityId"
                :initialActivity="editingActivityData || {}" @activityAdded="refreshContact"
                @close="showAddActivity = false" />
        </v-dialog>

        <v-dialog v-model="showAddNote" max-width="500px" persistent>
            <NoteAdd :contactId="contact.ID" :noteId="editingNoteId" :initialNote="editingNoteData || {}"
                @noteAdded="refreshContact" @close="showAddNote = false" />
        </v-dialog>




    </v-container>
</template>

<script>
import contactService from '@/services/contactService';
import { reactive } from 'vue';
import ActivityAdd from '@/components/ActivityAdd.vue';
import NoteAdd from '@/components/NoteAdd.vue';
import activityService from '@/services/activityService';
import noteService from '@/services/noteService';

export default {
    name: 'ContactView',
    props: {
        ID: {
            required: true,
        },
    },
    components: { ActivityAdd, NoteAdd },
    data() {
        return {
            contact: null,
            showAddActivity: false,
            showAddNote: false,
            isEditing: reactive({}),
            editValues: reactive({}),
            isEditingName: false,
            editName: '',
            newCircle: '', // Holds the new circle being added
            showAddCircleInput: false, // Controls visibility of the add circle input
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
                Object.keys(this.contactDetails).forEach((key) => {
                    this.isEditing[key] = false;
                    this.editValues[key] = this.contactDetails[key];
                });
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
            this.editValues[key] = this.contactDetails[key];
        },
        async saveEdit(key) {
            try {
                // Update the local contact data with the edited value
                this.contact[key.toLowerCase()] = this.editValues[key];

                // Send the updated data to the backend
                await contactService.updateContact(this.ID, { [key.toLowerCase()]: this.editValues[key] });

                // End the editing mode for this attribute
                this.isEditing[key] = false;
            } catch (error) {
                console.error('Error saving edit:', error);
                // Optionally, show an error message to the user
            }
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
        openAddActivity() {
            this.editingActivityId = null; // Reset for add mode
            this.editingActivityData = {}; // Reset for add mode
            this.showAddActivity = true;
        },
        openAddNote() {
            this.editingNoteId = null; // Reset for add mode
            this.editingNoteData = {}; // Reset for add mode
            this.showAddNote = true;
        },
        async editActivity(activityId) {
            const activity = this.contact.activities.find((a) => a.ID === activityId);
            this.editingActivityId = activityId;
            this.editingActivityData = {
                title: activity.title,
                description: activity.description,
                date: activity.date,
                location: activity.location,
            };
            this.showAddActivity = true;
        },

        async deleteActivity(activityId) {
            try {
                await activityService.deleteActivity(activityId);
                this.refreshContact(); // Refresh contact details after deletion
            } catch (error) {
                console.error('Error deleting activity:', error);
            }
        },
        async editNote(noteId) {
            const note = this.contact.notes.find((n) => n.ID === noteId);
            this.editingNoteId = noteId;
            this.editingNoteData = {
                content: note.content,
                date: note.date,
            };
            this.showAddNote = true;
        },
        async deleteNote(noteId) {
            try {
                await noteService.deleteNote(noteId);
                this.refreshContact(); // Refresh contact details after deletion
            } catch (error) {
                console.error('Error deleting note:', error);
            }
        },
        // Refresh the contact details after adding, editing, or deleting
        refreshContact() {
            this.showAddActivity = false;
            this.showAddNote = false;
            this.editingActivityId = null;
            this.editingActivityData = null;
            this.editingNoteId = null;
            this.editingNoteData = null;
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

.timeline-date-section:hover .edit-icon,
.timeline-date-section:hover .delete-icon {
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
