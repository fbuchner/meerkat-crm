<template>
    <v-card outlined class="mt-4 relationship-section">
        <v-card-title>Relationships</v-card-title>
        <v-card-text>
            <!-- List of Existing Relationships -->
            <v-list dense>
                <v-list-item v-for="relationship in relationships" :key="relationship.ID" class="field-label">
                    <div v-if="relationship.contact_id != null && relationship.related_contact">
                        <strong>{{ relationship.type }}: </strong>
                        {{ relationship.related_contact.firstname }}
                        {{ relationship.related_contact.lastname }}
                    </div>

                    <div v-else>
                        <strong>{{ relationship.name }}</strong>
                        {{ relationship.type }} ({{ relationship.gender }})
                        <strong>{{ relationship.name }}</strong>
                    </div>

                    <template v-slot:append>
                        <v-icon small class="edit-icon ml-2" @click="editRelationship(relationship)">mdi-pencil</v-icon>
                        <v-icon small class="delete-icon ml-2" color="error"
                            @click="deleteRelationship(relationship.ID)">mdi-delete</v-icon>
                    </template>
                </v-list-item>
                    <!-- Icon to Add New Relationship -->
                    <v-icon small class="add-circle-icon mt-2" @click="openAddRelationshipDialog">
                mdi-plus-circle
            </v-icon>
            </v-list>
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

                    <v-window v-model="activeTab">
                        <!-- Manual Entry Tab -->
                        <v-window-item value="manual">
                            <v-form>
                                <v-combobox v-model="relationshipForm.type" :items="relationshipTypes"
                                    label="Relationship Type" outlined color="blue-grey-lighten-2"
                                    required></v-combobox>
                                <v-text-field v-model="relationshipForm.name" label="Name" required></v-text-field>
                                <v-select v-model="relationshipForm.gender" :items="['Male', 'Female', 'Unknown']"
                                    label="Gender" required></v-select>
                                <v-text-field v-model="relationshipForm.birthday" label="Birthday (Optional)"
                                    placeholder="DD.MM.YYYY or DD.MM."></v-text-field>
                            </v-form>
                        </v-window-item>

                        <!-- Select Existing Contact Tab -->
                        <v-window-item value="existing">
                            <v-form>
                                <v-select v-model="relationshipForm.type" :items="relationshipTypes"
                                    label="Relationship Type" required></v-select>
                                <v-autocomplete v-model="relationshipForm.related_contact" :items="filteredContacts"
                                    item-title="name" item-value="ID" label="Select Existing Contact" return-object
                                    outlined color="blue-grey-lighten-2" required>

                                    <!-- Dropdown Item Slot -->
                                    <template v-slot:item="{ props, item }">
                                        <v-list-item v-bind="props" :key="item.ID"
                                            :prepend-avatar="getAvatarURL(item.value)" :text="item.title"></v-list-item>
                                    </template>
                                </v-autocomplete>
                            </v-form>
                        </v-window-item>
                    </v-window>
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
import { backendURL } from '@/services/api';

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
            showAddRelationshipDialog: false,
            editingRelationship: null,
            relationships: [],
            relationshipForm: {
                name: '',
                type: '',
                gender: '',
                birthday: '',
                related_contact: null,
            },
            relationshipTypes: ['Child', 'Parent', 'Sibling', 'Partner', 'Friend'],
            contacts: [], // Contacts for existing contact selection
            backendURL,
        };
    },
    computed: {
        filteredContacts() {
            return this.contacts;
        },
    },
    mounted() {
        this.loadRelationships();

        this.loadContacts();
        //TODO: only execute the function when a relationship is added
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
        async loadRelationships() {
            try {
                const response = await contactService.getRelationships(this.contactId)
                this.relationships = response.data.relationships
            } catch (error) {
                console.error('Error fetching relationships:', error);
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

                const relationshipData = {
                    type: this.relationshipForm.type,
                    name: null,
                    gender: null,
                    birthday: null,
                    contact_id: this.relationshipForm.related_contact.ID
                }

                await contactService.addRelationship(this.contactId, relationshipData);

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
        getAvatarURL(ID) {
            return `${this.backendURL}/contacts/${ID}/profile_picture.jpg`;
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

.v-list-item .edit-icon,
.v-list-item .delete-icon {
    opacity: 0;
    /* Hide icons by default */
    transition: opacity 0.3s ease;
    cursor: pointer;
}

.v-list-item:hover .edit-icon,
.v-list-item:hover .delete-icon {
    opacity: 1;
    /* Show icons on hover */
}
</style>