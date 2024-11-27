<template>
    <v-card outlined class="mb-4 relationship-section">
        <v-card-title>
            Relationships
            <v-icon class="cursor-pointer" @click="toggleCollapse">
                {{ isCollapsed ? 'mdi-chevron-down' : 'mdi-chevron-up' }}
            </v-icon>
        </v-card-title>

        <v-expand-transition>
            <div v-if="!isCollapsed">
                <v-card-text>
                    <!-- List of Existing Relationships -->
                    <v-list dense>
                        <v-list-item dense v-for="relationship in relationships" :key="relationship.ID" class="field-label">
                            <div v-if="relationship.related_contact_id != null && relationship.related_contact">
                                <strong>{{ relationship.type }}: </strong>
                                {{ relationship.related_contact.firstname }}
                                {{ relationship.related_contact.lastname }}
                                <span v-if="relationship.related_contact.birthday">( {{
                                    formatBirthday(relationship.birthday) }}
                                    )</span>
                            </div>

                            <div v-else>
                                <strong>{{ relationship.type }}: </strong>
                                {{ relationship.name }}
                                <span v-if="relationship.birthday">({{ formatBirthday(relationship.birthday) }})</span>
                            </div>

                            <template v-slot:append>
                                <v-icon small class="edit-icon ml-2"
                                    @click="editRelationship(relationship)">mdi-pencil</v-icon>
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
            </div>
        </v-expand-transition>

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
                                <v-text-field v-model="formattedBirthday" label="Birthday (Optional)"
                                    placeholder="DD.MM.YYYY or DD.MM." :error-messages="birthdayError"
                                    @blur="validateBirthday"></v-text-field>
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
            birthdayError: '',
            relationshipTypes: ['Child', 'Parent', 'Sibling', 'Partner', 'Friend'],
            contacts: [], // Contacts for existing contact selection
            backendURL,
            isCollapsed: false,
        };
    },
    computed: {
        filteredContacts() {
            return this.contacts;
        },
        formattedBirthday: {
            // Getter: Format the date for display
            get() {
                // Convert YYYY-MM-DD to DD.MM.YYYY
                if (!this.relationshipForm.birthday) return ""; // Handle null or empty cases
                const [year, month, day] = this.relationshipForm.birthday.split("-");
                return `${day}.${month}.${year && year !== '0001' ? year : ''}`;
            },
            // Setter: Parse the date back to ISO format
            set(value) {
                if (!value) {
                    this.relationshipForm.birthday = null;
                }
                // Convert DD.MM.YYYY back to YYYY-MM-DD
                const parts = value.split(".");
                if (parts.length === 3) {
                    const [day, month, year] = parts;
                    this.relationshipForm.birthday = `${year || '0001'}-${month.padStart(2, '0')}-${day.padStart(2, '0')}`;
                }
            },
        },
    },
    mounted() {
        this.fetchRelationships();

        this.loadContacts();
        //TODO: only execute the function when a relationship is added
    },
    methods: {
        toggleCollapse() {
            this.isCollapsed = !this.isCollapsed;
        },
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
        async fetchRelationships() {
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
            const relationshipData = {
                type: null,
                name: null,
                gender: null,
                birthday: null,
                contact_id: this.contactId,
                related_contact_id: null,
            };

            try {
                if (this.activeTab === 'manual') {
                    if (!this.relationshipForm.name || !this.relationshipForm.type) {
                        throw new Error('Please provide both name and relationship type.');
                    }

                    if (this.birthdayError) {
                        throw new Error('Invalid birthday format.');
                    }

                    relationshipData.type = this.relationshipForm.type;
                    relationshipData.name = this.relationshipForm.name;
                    relationshipData.gender = this.relationshipForm.gender;
                    relationshipData.birthday = this.relationshipForm.birthday;

                } else if (this.activeTab === 'existing') {
                    if (!this.relationshipForm.related_contact || !this.relationshipForm.type) {
                        throw new Error('Please select an existing contact and provide the relationship type.');
                    }

                    relationshipData.type = this.relationshipForm.type;
                    relationshipData.related_contact_id = this.relationshipForm.related_contact.ID;
                }

                if (this.editingRelationship) {
                    await contactService.updateRelationship(this.contactId, this.editingRelationship.ID, relationshipData);
                } else {
                    await contactService.addRelationship(this.contactId, relationshipData);
                }

                this.fetchRelationships();
                this.closeAddRelationshipDialog();

                // Reset editing state
                this.editingRelationship = null;

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
            this.birthdayError = '';
        },
        validateBirthday() {
            // Regular expression to match "DD.MM.YYYY" or "DD.MM." format
            const datePattern = /^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])\.(\d{4})?$/;
            if (this.formattedBirthday != null && this.formattedBirthday != "" && !this.formattedBirthday.match(datePattern)) {
                this.birthdayError = "Please enter a valid date in DD.MM.YYYY or DD.MM. format.";
            } else {
                this.birthdayError = '';
            }
        },
        formatBirthday(value) {
            if (!value) return ""; // Handle null or empty cases
            const [year, month, day] = value.split("-");
            return `${day}.${month}.${year && year !== '0001' ? year : ''}`;
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

.cursor-pointer {
  cursor: pointer;
}

</style>
