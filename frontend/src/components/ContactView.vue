<template>
    <v-container v-if="contact">
        <!-- Top Section with Profile and Edit Button -->
        <v-row class="d-flex flex-column flex-md-row align-center text-center text-md-left">
            <v-col cols="12" md="3" class="d-flex justify-center">
                <v-img :src="contact.photo || ''" alt="Profile Photo" width="150" height="150"
                    class="circular-frame mb-2 fixed-square" contain></v-img>
            </v-col>
            <v-col cols="12" md="9" class="d-flex flex-column justify-center text-center text-md-left">
                <div class="d-flex align-center justify-center justify-md-start">
                    <h1 class="text-h4 font-weight-bold">{{ contact.firstname }} {{ contact.lastname }}</h1>
                    <v-btn color="primary" class="ml-4" @click="editContact">Edit Contact</v-btn>
                </div>
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
                            <v-list-item><strong>Nickname:</strong> {{ contact.nickname }}</v-list-item>
                            <v-list-item><strong>Gender:</strong> {{ contact.gender }}</v-list-item>
                            <v-list-item><strong>Birthday:</strong> {{ contact.birthday }}</v-list-item>
                            <v-list-item><strong>Email:</strong> {{ contact.email }}</v-list-item>
                            <v-list-item><strong>Phone:</strong> {{ contact.phone }}</v-list-item>
                            <v-list-item><strong>Address:</strong> {{ contact.address }}</v-list-item>
                            <v-list-item><strong>How We Met:</strong> {{ contact.how_we_met }}</v-list-item>
                            <v-list-item><strong>Food Preference:</strong> {{ contact.food_preference }}</v-list-item>
                            <v-list-item><strong>Work Information:</strong> {{ contact.work_information }}</v-list-item>
                            <v-list-item><strong>Additional Information:</strong> {{ contact.contact_information
                                }}</v-list-item>
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
                            <v-subheader>Activities</v-subheader>
                            <v-list-item v-for="activity in contact.activities" :key="activity.ID">
                                <v-list-item-content>
                                    <v-list-item-title>{{ activity.date }} - {{ activity.name }}</v-list-item-title>
                                    <v-list-item-subtitle>{{ activity.description }} at {{ activity.location
                                        }}</v-list-item-subtitle>
                                    <v-btn small text @click="editActivity(activity.ID)">Edit</v-btn>
                                    <v-btn small text color="error" @click="deleteActivity(activity.ID)">Delete</v-btn>
                                </v-list-item-content>
                            </v-list-item>

                            <!-- Notes Timeline -->
                            <v-subheader class="mt-4">Notes</v-subheader>
                            <v-list-item v-for="note in contact.notes" :key="note.ID">
                                <v-list-item-content>
                                    <v-list-item-title>{{ note.date }}: {{ note.content }}</v-list-item-title>
                                    <v-btn small text @click="editNote(note.ID)">Edit</v-btn>
                                    <v-btn small text color="error" @click="deleteNote(note.ID)">Delete</v-btn>
                                </v-list-item-content>
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
        };
    },
    mounted() {
        this.fetchContact();
    },
    methods: {
        async fetchContact() {
            try {
                const response = await contactService.getContact(this.ID);
                this.contact = response.data;
            } catch (error) {
                console.error('Error fetching contact:', error);
            }
        },
        editContact() {
            // Trigger edit contact functionality
        },
        editActivity(activityId) {
            this.editingActivityId = activityId;
        },
        async deleteActivity(activityId) {
            try {
                await contactService.deleteActivity(activityId);
                this.fetchContact();
            } catch (error) {
                console.error('Error deleting activity:', error);
            }
        },
        editNote(noteId) {
            this.editingNoteId = noteId;
        },
        async deleteNote(noteId) {
            try {
                await contactService.deleteNote(noteId);
                this.fetchContact();
            } catch (error) {
                console.error('Error deleting note:', error);
            }
        },
    },
};
</script>

<style scoped>
.rounded {
    border-radius: 8px;
}

.mb-2 {
    margin-bottom: 16px;
}

.circular-frame {
  border-radius: 50%;
  background-color: #f0f0f0; /* Light gray background for empty frame */
  border: 2px solid #ccc;    /* Optional: subtle border */
}

.fixed-square {
  width: 150px;
  height: 150px;
  max-width: 150px;
  max-height: 150px;
}

</style>