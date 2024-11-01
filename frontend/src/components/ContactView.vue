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

                <!-- Circles Chips -->
                <div v-if="contact.circles && contact.circles.length">
                    <v-chip-group row>
                        <v-chip v-for="circle in contact.circles" :key="circle" class="mr-2">{{ circle }}</v-chip>
                    </v-chip-group>
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
                                    <strong>{{ key }}:</strong>
                                    <span v-if="!isEditing[key]" @click="startEditing(key)">
                                        {{ value }}
                                        <v-icon small class="edit-icon" @click.stop="startEditing(key)">mdi-pencil</v-icon>
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
                    <v-card-title>Timeline</v-card-title>
                    <v-card-text>
                        <v-list dense>
                            <!-- Activities Timeline -->
                            <v-card-subtitle>Activities</v-card-subtitle>
                            <v-divider></v-divider>
                            <v-list-item v-for="activity in contact.activities" :key="activity.ID">
                                <div>
                                    <strong>{{ activity.date }} - {{ activity.name }}</strong><br>
                                    <span>{{ activity.description }} at {{ activity.location }}</span>
                                </div>
                                <v-btn small text @click="editActivity(activity.ID)">Edit</v-btn>
                                <v-btn small text color="error" @click="deleteActivity(activity.ID)">Delete</v-btn>
                            </v-list-item>

                            <!-- Divider between sections -->
                            <v-divider class="my-4"></v-divider>

                            <!-- Notes Timeline -->
                            <v-card-subtitle>Notes</v-card-subtitle>
                            <v-list-item v-for="note in contact.notes" :key="note.ID">
                                <div>
                                    <strong>{{ note.date }}:</strong> {{ note.content }}
                                </div>
                                <v-btn small text @click="editNote(note.ID)">Edit</v-btn>
                                <v-btn small text color="error" @click="deleteNote(note.ID)">Delete</v-btn>
                            </v-list-item>
                        </v-list>
                    </v-card-text>
                </v-card>
            </v-col>
        </v-row>
    </v-container>
</template>

<script>
import contactService from '@/services/contactService';
import { reactive } from 'vue';

export default {
    name: 'ContactView',
    props: {
        ID: {
            required: true,
        },
    },
    data() {
        return {
            contact: null,
            editingActivityId: null,
            editingNoteId: null,
            isEditing: reactive({}), // Track edit state for each field
            editValues: reactive({}), // Store current edit values for each field
            isEditingName: false, // Track edit state for name
            editName: '', // Temp storage for editing the name
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
    },
    mounted() {
        this.fetchContact();
    },
    methods: {
        async fetchContact() {
            try {
                const response = await contactService.getContact(this.ID);
                this.contact = response.data;

                // Initialize editing states and edit values based on contact details
                Object.keys(this.contactDetails).forEach((key) => {
                    this.isEditing[key] = false;
                    this.editValues[key] = this.contactDetails[key];
                });

                // Set up the name for editing
                this.editName = `${this.contact.firstname} ${this.contact.lastname}`;
            } catch (error) {
                console.error('Error fetching contact:', error);
            }
        },
        startEditingName() {
            this.isEditingName = true;
        },
        async saveNameEdit() {
            const [firstname, lastname] = this.editName.split(' ');
            try {
                await contactService.updateContact(this.ID, {
                    ...this.contact,
                    firstname,
                    lastname,
                });
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
        startEditing(key) {
            this.isEditing[key] = true;
            this.editValues[key] = this.contactDetails[key];
        },
        async saveEdit(key) {
            const updatedData = { ...this.contact, [key.toLowerCase()]: this.editValues[key] };
            try {
                await contactService.updateContact(this.ID, updatedData);
                this.contact[key.toLowerCase()] = this.editValues[key];
                this.isEditing[key] = false;
            } catch (error) {
                console.error(`Error updating ${key}:`, error);
            }
        },
        cancelEdit(key) {
            this.isEditing[key] = false;
            this.editValues[key] = this.contactDetails[key];
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

.edit-icon {
    cursor: pointer;
    opacity: 0; /* Hide by default */
    transition: opacity 0.3s ease;
}

.field-label:hover .edit-icon {
    opacity: 1; /* Show on hover */
}

.edit-field {
    display: flex;
    align-items: center;
    gap: 8px;
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
