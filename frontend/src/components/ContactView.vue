<template>
    <div class="contact-view" v-if="contact">
        <h1>{{ contact.firstname }} {{ contact.lastname }}</h1>
        <img :src="contact.photo" alt="Profile Photo" class="contact-photo" />
        <p><strong>Nickname:</strong> {{ contact.nickname }}</p>
        <p><strong>Email:</strong> {{ contact.email }}</p>
        <p><strong>Phone:</strong> {{ contact.phone }}</p>
        <p><strong>Birthday:</strong> {{ contact.birthday }}</p>
        <p><strong>Gender:</strong> {{ contact.gender }}</p>
        <p><strong>Address:</strong> {{ contact.address }}</p>
        <strong>Partner:</strong>
        <div v-if="contact.partner"> {{ contact.partner.name }} ({{ contact.partner.gender }} - {{
            contact.partner.birthday }})</div>
        <p><strong>How We Met:</strong> {{ contact.how_we_met }}</p>
        <p><strong>Food Preference:</strong> {{ contact.food_preference }}</p>
        <p><strong>Work Information:</strong> {{ contact.work_information }}</p>
        <p><strong>Additional Contact Information:</strong> {{ contact.contact_information }}</p>

        <div v-if="contact.relationships?.length">
            <h3>Relationships</h3>
            <ul>
                <li v-for="relationship in contact.relationships" :key="relationship.name">
                    {{ relationship.type }}: {{ relationship.name }} ({{ relationship.gender }} - {{
                        relationship.birthday }})
                </li>
            </ul>
        </div>

        <div v-if="contact.circles?.length">
            <h3>Circles</h3>
            <ul>
                <li v-for="circle in contact.circles" :key="circle.ID">
                    {{ circle.name }}
                </li>
            </ul>
        </div>

        <!-- Add the ActivityAdd component here -->
        <ActivityAdd :contactId="ID" @activityAdded="fetchContact" />

        <!-- Displaying Activities with Edit/Delete Options -->
        <div v-if="contact.activities?.length">
            <h3>Activities</h3>
            <ul>
                <li v-for="activity in contact.activities" :key="activity.ID">
                    <div v-if="editingActivityId === activity.ID">
                        <input v-model="activity.name" placeholder="Activity name" />
                        <input v-model="activity.date" type="date" />
                        <input v-model="activity.description" placeholder="Activity description" />
                        <input v-model="activity.location" placeholder="Activity location" />
                        <button @click="saveActivity(activity)">Save</button>
                        <button @click="cancelEditActivity">Cancel</button>
                    </div>
                    <div v-else>
                        {{ activity.name }} - {{ activity.date }}
                        <p>{{ activity.description }}</p>
                        <p>{{ activity.location }}</p>
                        <button @click="editActivity(activity.ID)">Edit</button>
                        <button @click="deleteActivity(activity.ID)">Delete</button>
                    </div>
                </li>
            </ul>
        </div>

       <!-- Add the NoteAdd component here -->
        <NoteAdd :contactId="ID" @noteAdded="fetchContact" />

        <div v-if="contact.notes?.length">
            <h3>Notes</h3>
            <ul>
                <li v-for="note in contact.notes" :key="note.ID">
                    <div v-if="editingNoteId === note.ID">
                        <textarea v-model="note.content"></textarea>
                        <button @click="saveNote(note)">Save</button>
                        <button @click="cancelEditNote">Cancel</button>
                    </div>
                    <div v-else>
                        {{ note.date }}: {{ note.content }}
                        <button @click="editNote(note.ID)">Edit</button>
                        <button @click="deleteNote(note.ID)">Delete</button>
                    </div>
                </li>
            </ul>
        </div>
    </div>
</template>

<script>
import contactService from '@/services/contactService';
import ActivityAdd from '@/components/ActivityAdd.vue'; 
import NoteAdd from '@/components/NoteAdd.vue'; 

export default {
    name: 'ContactView',
    props: {
        ID: {
            required: true,
        },
    },
    components: {
        ActivityAdd, 
        NoteAdd,
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
        editActivity(activityId) {
            this.editingActivityId = activityId;
        },
        cancelEditActivity() {
            this.editingActivityId = null;
        },
        async saveActivity(activity) {
            try {
                await contactService.updateActivity(activity.ID, {
                    name: activity.name,
                    date: activity.date,
                });
                this.editingActivityId = null;
                this.fetchContact(); // Refresh contact to get the updated activity
            } catch (error) {
                console.error('Error saving activity:', error);
            }
        },
        async deleteActivity(activityId) {
            try {
                await contactService.deleteActivity(activityId);
                this.fetchContact(); // Refresh contact to remove deleted activity
            } catch (error) {
                console.error('Error deleting activity:', error);
            }
        },
        editNote(noteId) {
            this.editingNoteId = noteId;
        },
        cancelEditNote() {
            this.editingNoteId = null;
        },
        async saveNote(note) {
            try {
                await contactService.updateNote(note.ID, { content: note.content });
                this.editingNoteId = null;
                this.fetchContact(); // Refresh contact data
            } catch (error) {
                console.error('Error saving note:', error);
            }
        },
        async deleteNote(noteId) {
            try {
                await contactService.deleteNote(noteId);
                this.fetchContact(); // Refresh contact data
            } catch (error) {
                console.error('Error deleting note:', error);
            }
        },
    },
};
</script>

<style scoped>
.contact-view {
    max-width: 600px;
    margin: 0 auto;
    padding: 20px;
    border: 1px solid #ccc;
    border-radius: 8px;
}

.contact-photo {
    width: 150px;
    height: 150px;
    border-radius: 50%;
    object-fit: cover;
}
</style>