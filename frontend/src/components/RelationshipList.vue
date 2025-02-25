<template>
  <v-card outlined class="mb-4 relationship-section">
    <v-card-title>
      {{ $t("relationships.title") }}
      <v-icon class="cursor-pointer" @click="toggleCollapse">
        {{ isCollapsed ? "mdi-chevron-down" : "mdi-chevron-up" }}
      </v-icon>
    </v-card-title>

    <v-expand-transition>
      <div v-if="!isCollapsed">
        <v-card-text>
          <!-- List of Existing Relationships -->
          <v-list dense>
            <v-list-item
              dense
              v-for="relationship in relationships"
              :key="relationship.ID"
              class="field-label"
            >
              <div
                v-if="
                  relationship.related_contact_id != null &&
                  relationship.related_contact
                "
              >
                <strong>{{ relationship.type }}: </strong>
                {{ relationship.related_contact.firstname }}
                {{ relationship.related_contact.lastname }}
                <span v-if="relationship.related_contact.birthday"
                  >(
                  {{ formatBirthday(relationship.related_contact.birthday) }}
                  )</span
                >
              </div>

              <div v-else>
                <strong>{{ relationship.type }}: </strong>
                {{ relationship.name }}
                <span v-if="relationship.birthday"
                  >({{ formatBirthday(relationship.birthday) }})</span
                >
              </div>

              <template v-slot:append>
                <v-icon
                  small
                  class="edit-icon ml-2"
                  @click="editRelationship(relationship)"
                  >mdi-pencil</v-icon
                >
                <v-icon
                  small
                  class="delete-icon ml-2"
                  color="error"
                  @click="deleteRelationship(relationship.ID)"
                  >mdi-delete</v-icon
                >
              </template>
            </v-list-item>
            <!-- Icon to Add New Relationship -->
            <v-icon
              small
              class="add-circle-icon mt-2"
              @click="openAddRelationshipDialog"
            >
              mdi-plus-circle
            </v-icon>
          </v-list>
        </v-card-text>
      </div>
    </v-expand-transition>

    <!-- Dialog to Add/Edit Relationship -->
    <v-dialog v-model="showAddRelationshipDialog" max-width="500px">
      <v-card>
        <v-card-title>{{
          editingRelationship ? "Edit Relationship" : "Add Relationship"
        }}</v-card-title>
        <v-card-text>
          <v-tabs v-model="activeTab" class="mb-4">
            <v-tab value="manual">{{ $t("relationships.manual_entry") }}</v-tab>
            <v-tab value="existing">{{
              $t("relationships.existing_contact")
            }}</v-tab>
          </v-tabs>

          <v-window v-model="activeTab">
            <!-- Manual Entry Tab -->
            <v-window-item value="manual">
              <v-form>
                <v-combobox
                  v-model="relationshipForm.type"
                  :items="relationshipTypes"
                  :label="$t('relationships.relationship_type')"
                  outlined
                  color="blue-grey-lighten-2"
                  required
                ></v-combobox>
                <v-text-field
                  v-model="relationshipForm.name"
                  :label="$t('relationships.relationship_name')"
                  required
                ></v-text-field>
                <v-select
                  v-model="relationshipForm.gender"
                  :items="$t('contacts.contact_fields.genders').split(',')"
                  :label="$t('contacts.contact_fields.gender')"
                  required
                ></v-select>
                <v-text-field
                  v-model="formattedBirthday"
                  :label="$t('contacts.contact_fields.birthday')"
                  :placeholder="$t('contacts.birthday.birthday_format')"
                  :error-messages="birthdayError"
                  @blur="validateBirthday"
                ></v-text-field>
              </v-form>
            </v-window-item>

            <!-- Select Existing Contact Tab -->
            <v-window-item value="existing">
              <v-form>
                <v-select
                  v-model="relationshipForm.type"
                  :items="relationshipTypes"
                  :label="$t('relationships.relationship_type')"
                  required
                ></v-select>
                <v-autocomplete
                  v-model="relationshipForm.related_contact"
                  :items="filteredContacts"
                  item-title="name"
                  item-value="ID"
                  :label="$t('relationships.existing_contact')"
                  return-object
                  outlined
                  color="blue-grey-lighten-2"
                  required
                >
                  <!-- Dropdown Item Slot -->
                  <template v-slot:item="{ props, item }">
                    <v-list-item
                      v-bind="props"
                      :key="item.ID"
                      :prepend-avatar="getAvatarURL(item.value)"
                      :text="item.title"
                    ></v-list-item>
                  </template>
                </v-autocomplete>
              </v-form>
            </v-window-item>
          </v-window>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn @click="closeAddRelationshipDialog">{{
            $t("buttons.cancel")
          }}</v-btn>
          <v-btn color="primary" @click="saveRelationship">{{
            editingRelationship
              ? $t("buttons.save")
              : $t("relationships.add_relationship")
          }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-card>
</template>

<script>
import contactService from "@/services/contactService";
import { backendURL } from "@/services/api";

export default {
  name: "RelationshipList",
  props: {
    contactId: {
      required: true,
    },
  },
  data() {
    return {
      activeTab: "manual",
      showAddRelationshipDialog: false,
      editingRelationship: null,
      relationships: [],
      relationshipForm: {
        name: "",
        type: "",
        gender: "",
        birthday: null,
        related_contact: null,
      },
      birthdayError: "",
      relationshipTypes: this.$t("relationships.relationship_types").split(","),
      contacts: [],
      searchContactQuery: "",
      backendURL,
      isCollapsed: false,
      debouncedLoadContacts: null,
    };
  },
  computed: {
    filteredContacts() {
      return this.contacts;
    },
    formattedBirthday: {
      get() {
        if (!this.relationshipForm.birthday) return null;
        const [year, month, day] = this.relationshipForm.birthday.split("-");
        return `${day}.${month}.${year && year !== "0001" ? year : ""}`;
      },
      set(value) {
        if (!value) {
          this.relationshipForm.birthday = null;
          return;
        }
        const parts = value.split(".");
        if (parts.length === 3) {
          const [day, month, year] = parts;
          this.relationshipForm.birthday = `${year || "0001"}-${month.padStart(
            2,
            "0"
          )}-${day.padStart(2, "0")}`;
        }
      },
    },
  },
  watch: {
    searchContactQuery(query) {
      if (this.debouncedLoadContacts) {
        this.debouncedLoadContacts(query);
      }
    },
  },
  mounted() {
    this.fetchRelationships();
    this.loadContacts();
  },
  created() {
    this.debouncedLoadContacts = this.debounce(this.loadContacts, 300);
  },
  methods: {
    debounce(func, delay) {
      let timeout;
      return (...args) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => func(...args), delay);
      };
    },
    async loadContacts(searchQuery = "") {
      try {
        const response = await contactService.getContacts({
          fields: ["ID", "firstname", "lastname"],
          search: searchQuery,
          limit: 15,
        });
        this.contacts = response.data.contacts.map((contact) => ({
          ID: contact.ID,
          name: `${contact.firstname} ${contact.lastname}`,
        }));
      } catch (error) {
        console.error("Error fetching contacts:", error);
      }
    },
    async fetchRelationships() {
      try {
        const response = await contactService.getRelationships(this.contactId);
        this.relationships = response.data.relationships;
      } catch (error) {
        console.error("Error fetching relationships:", error);
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
        if (this.activeTab === "manual") {
          if (!this.relationshipForm.name || !this.relationshipForm.type) {
            throw new Error("Please provide both name and relationship type.");
          }

          if (this.birthdayError) {
            throw new Error("Invalid birthday format.");
          }

          relationshipData.type = this.relationshipForm.type;
          relationshipData.name = this.relationshipForm.name;
          relationshipData.gender = this.relationshipForm.gender;
          relationshipData.birthday = this.relationshipForm.birthday ?? null;
        } else if (this.activeTab === "existing") {
          if (
            !this.relationshipForm.related_contact ||
            !this.relationshipForm.type
          ) {
            throw new Error(
              "Please select an existing contact and provide the relationship type."
            );
          }

          relationshipData.type = this.relationshipForm.type;
          relationshipData.related_contact_id =
            this.relationshipForm.related_contact.ID;
        }

        if (this.editingRelationship) {
          await contactService.updateRelationship(
            this.contactId,
            this.editingRelationship.ID,
            relationshipData
          );
        } else {
          await contactService.addRelationship(
            this.contactId,
            relationshipData
          );
        }

        this.fetchRelationships();
        this.closeAddRelationshipDialog();

        // Reset editing state
        this.editingRelationship = null;
      } catch (error) {
        console.error("Error saving relationship:", error);
      }
    },
    async deleteRelationship(relationshipId) {
      try {
        await contactService.deleteRelationship(this.contactId, relationshipId);
        this.fetchRelationships();
      } catch (error) {
        console.error("Error deleting relationship:", error);
      }
    },
    closeAddRelationshipDialog() {
      this.showAddRelationshipDialog = false;
      this.resetRelationshipForm();
    },
    resetRelationshipForm() {
      this.relationshipForm = {
        type: "",
        name: "",
        gender: "",
        birthday: null,
        related_contact: null,
      };
      this.birthdayError = "";
    },
    validateBirthday() {
      const datePattern =
        /^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])\.(\d{4})?$/;
      if (
        this.formattedBirthday &&
        !this.formattedBirthday.match(datePattern)
      ) {
        this.birthdayError = this.$t("contacts.birthday.birthday_warning");
      } else {
        this.birthdayError = "";
      }
    },
    formatBirthday(value) {
      if (!value) return "";
      const [year, month, day] = value.split("-");
      return `${day}.${month}.${year && year !== "0001" ? year : ""}`;
    },
    getAvatarURL(ID) {
      return `${this.backendURL}/contacts/${ID}/profile_picture.jpg`;
    },
    toggleCollapse() {
      this.isCollapsed = !this.isCollapsed;
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
  transition: opacity 0.3s ease;
  cursor: pointer;
}

.v-list-item:hover .edit-icon,
.v-list-item:hover .delete-icon {
  opacity: 1;
}

.cursor-pointer {
  cursor: pointer;
}
</style>
