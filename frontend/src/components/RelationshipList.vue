<template>
    <v-card outlined class="mt-4">
        <v-card-title>Relationships</v-card-title>
        <v-card-text>
            <!-- List of Existing Relationships -->
            <v-list dense>
                <v-list-item v-for="relationship in relationships" :key="relationship.ID">
                    <v-list-item-content>
                        <v-list-item-title>
                            <strong>{{ relationship.name }}</strong>
                            <span v-if="relationship.related_contact"> - Linked Contact: {{
                                relationship.related_contact.firstname }} {{ relationship.related_contact.lastname
                                }}</span>
                        </v-list-item-title>
                        <v-list-item-subtitle>{{ relationship.type }} ({{ relationship.gender }})</v-list-item-subtitle>
                    </v-list-item-content>
                    <v-list-item-action>
                        <v-icon small @click="editRelationship(relationship)">mdi-pencil</v-icon>
                        <v-icon small color="error" @click="deleteRelationship(relationship.id)">mdi-delete</v-icon>
                    </v-list-item-action>
                </v-list-item>
            </v-list>

            <!-- Button to Add New Relationship -->
            <v-btn color="primary" @click="openAddRelationshipDialog">Add Relationship</v-btn>
        </v-card-text>

        <!-- Dialog to Add/Edit Relationship -->
        <v-dialog v-model="showAddRelationshipDialog" max-width="500px">
            <v-card>
                <v-card-title>{{ editingRelationship ? 'Edit Relationship' : 'Add Relationship' }}</v-card-title>
                <v-card-text>
                    <v-tabs v-model="activeTab" class="mb-4">
                        <v-tab value="manual">Manual Entry</v-tab>
                        <v-tab value="existing">Select Existing Contact</v-tab>
                    </v-tabs>

                    <v-tabs-window v-model="activeTab">
                        <!-- Manual Entry Tab -->
                        <v-tabs-window-item value="manual">
                            <v-form>
                                <v-select v-model="relationshipForm.type" :items="relationshipTypes"
                                    label="Relationship Type" required></v-select>
                                <v-text-field v-model="relationshipForm.name" label="Name" required></v-text-field>
                                <v-select v-model="relationshipForm.gender" :items="['Male', 'Female', 'Unknown']"
                                    label="Gender" required></v-select>
                                <v-text-field v-model="relationshipForm.birthday" label="Birthday (Optional)"
                                    placeholder="DD.MM.YYYY or DD.MM."></v-text-field>
                            </v-form>
                        </v-tabs-window-item>

                        <!-- Select Existing Contact Tab -->
                        <v-tabs-window-item value="existing">
                            <v-form>
                                <v-select v-model="relationshipForm.type" :items="relationshipTypes"
                                    label="Relationship Type" required></v-select>
                                <v-autocomplete v-model="relationshipForm.related_contact" :items="filteredContacts"
                                    item-title="name" item-value="ID" label="Select Existing Contact"
                                    return-object outlined color="blue-grey-lighten-2" required></v-autocomplete>
                            </v-form>
                        </v-tabs-window-item>
                    </v-tabs-window>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn @click="closeAddRelationshipDialog">Cancel</v-btn>
                    <v-btn color="primary" @click="saveRelationship">{{ editingRelationship ? 'Save' : 'Add' }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-card>
</template>

<script>
import contactService from '@/services/contactService';

export default {
    name: 'RelationshipList',
    props: {
        contactId: {
            required: true,
        },
    },
    data() {
        return {
            activeTab: 'manual', // Track which tab is currently active
            relationships: [],
            showAddRelationshipDialog: false,
            editingRelationship: null,
            relationshipForm: {
                name: '',
                type: '',
                gender: '',
                birthday: '',
                related_contact: null,
            },
            relationshipTypes: ['Child', 'Parent', 'Sibling', 'Partner', 'Friend'],
            contacts: [], // Contacts for existing contact selection
        };
    },
    computed: {
        filteredContacts() {
            return this.contacts;
        },
    },
    mounted() {
        this.loadContacts();
    },
    methods: {
        async loadContacts() {
            try {
                const response = await contactService.getContacts({
                    fields: ['ID', 'firstname', 'lastname'],
                    limit: 5000,
                });
                this.contacts = response.data.contacts.map(contact => ({
                    ID: contact.ID,
                    name: `${contact.firstname} ${contact.lastname}`,
                }));
            } catch (error) {
                console.error('Error fetching contacts:', error);
            }
        },

        openAddRelationshipDialog() {
            this.showAddRelationshipDialog = true;
            this.editingRelationship = null;
            this.resetRelationshipForm();
        },
        editRelationship(relationship) {
            this.showAddRelationshipDialog = true;
            this.editingRelationship = relationship;
            this.relationshipForm = { ...relationship };
        },
        async saveRelationship() {
            try {
                // Implement save logic based on whether it's a manual entry or existing contact
                if (this.activeTab === 'manual') {
                    // Manual Entry - Save with manually entered data
                    if (!this.relationshipForm.name || !this.relationshipForm.type) {
                        throw new Error('Please provide both name and relationship type.');
                    }
                } else if (this.activeTab === 'existing') {
                    // Select Existing Contact - Save with linked contact
                    if (!this.relationshipForm.related_contact || !this.relationshipForm.type) {
                        throw new Error('Please select an existing contact and provide the relationship type.');
                    }
                }
                // Emit or call function to save relationship
                this.$emit('relationship-added');
                this.closeAddRelationshipDialog();
            } catch (error) {
                console.error('Error saving relationship:', error);
            }
        },
        async deleteRelationship(relationshipId) {
            try {
                await contactService.deleteRelationship(this.contactId, relationshipId);
                this.fetchRelationships();
            } catch (error) {
                console.error('Error deleting relationship:', error);
            }
        },
        closeAddRelationshipDialog() {
            this.showAddRelationshipDialog = false;
            this.resetRelationshipForm();
        },
        resetRelationshipForm() {
            this.relationshipForm = {
                type: '',
                name: '',
                gender: '',
                birthday: '',
                related_contact: null,
            };
        },
    },
};
</script>