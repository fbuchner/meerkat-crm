<template>
  <v-container>
    <v-row>
      <v-col>
        <v-card>
          <v-card-title>{{ $t("contacts.add_contact") }}</v-card-title>
          <v-card-text>
            <v-form @submit.prevent="submitForm">
              <v-text-field
                :label="$t('contacts.contact_fields.firstname')"
                v-model="contact.firstname"
                required
              ></v-text-field>

              <v-text-field
                :label="$t('contacts.contact_fields.lastname')"
                v-model="contact.lastname"
              ></v-text-field>

              <v-text-field
                :label="$t('contacts.contact_fields.nickname')"
                v-model="contact.nickname"
              ></v-text-field>

              <v-select
                :label="$t('contacts.contact_fields.gender')"
                v-model="contact.gender"
                :items="$t('contacts.contact_fields.genders').split(',')"
              ></v-select>

              <v-text-field
                :label="$t('contacts.circles.circles')"
                v-model="circleInput"
                @keyup.space="addCircle"
                :placeholder="$t('contacts.circles.add_circles')"
              ></v-text-field>

              <v-chip-group v-if="contact.circles.length">
                <v-chip
                  v-for="(circle, index) in contact.circles"
                  :key="index"
                  close
                  @click:close="removeCircle(index)"
                >
                  {{ circle }}
                </v-chip>
              </v-chip-group>

              <v-text-field
                :label="$t('contacts.contact_fields.email')"
                v-model="contact.email"
                type="email"
              ></v-text-field>

              <v-text-field
                :label="$t('contacts.contact_fields.phone')"
                v-model="contact.phone"
                type="tel"
              ></v-text-field>

              <v-text-field
                :label="$t('contacts.contact_fields.birthday')"
                v-model="birthdayInput"
                :placeholder="$t('contacts.birthday.birthday_format')"
                :error-messages="birthdayError"
                @blur="validateBirthday"
              ></v-text-field>

              <v-text-field
                :label="$t('contacts.contact_fields.address')"
                v-model="contact.address"
              ></v-text-field>

              <v-textarea
                :label="$t('contacts.contact_fields.how_we_met')"
                v-model="contact.how_we_met"
              ></v-textarea>

              <v-text-field
                :label="$t('contacts.contact_fields.food_preference')"
                v-model="contact.food_preference"
              ></v-text-field>

              <v-text-field
                :label="$t('contacts.contact_fields.work_information')"
                v-model="contact.work_information"
              ></v-text-field>

              <v-textarea
                :label="$t('contacts.contact_fields.additional_information')"
                v-model="contact.contact_information"
              ></v-textarea>

              <v-btn type="submit" color="primary">{{
                $t("contacts.add_contact")
              }}</v-btn>

              <v-alert
                v-if="successMessage"
                type="success"
                dismissible
                class="mt-3"
              >
                {{ successMessage }}
              </v-alert>
              <v-alert
                v-if="errorMessage"
                type="error"
                dismissible
                class="mt-3"
              >
                {{ errorMessage }}
              </v-alert>
            </v-form>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import contactService from "@/services/contactService";

export default {
  data() {
    return {
      contact: {
        firstname: "",
        lastname: "",
        nickname: "",
        gender: this.$t("contacts.contact_fields.gender_unknown"),
        email: "",
        phone: "",
        birthday: null, // Birthday is nullable here
        address: "",
        how_we_met: "",
        food_preference: "",
        work_information: "",
        contact_information: "",
        circles: [],
      },
      successMessage: "",
      errorMessage: "",
      circleInput: "",
      birthdayInput: "",
      birthdayError: "",
    };
  },
  methods: {
    submitForm() {
      this.validateBirthday();

      // If birthdayInput is empty, set contact.birthday to null
      if (!this.birthdayInput) {
        this.contact.birthday = null;
        this.birthdayError = ""; // Clear previous error (if any)
      } else if (this.birthdayError) {
        this.errorMessage = this.$t("contacts.birthday.birthday_format_error");
        return;
      }

      contactService
        .addContact(this.contact)
        .then(() => {
          this.successMessage = this.$t("contacts.add_contact_success");
          this.errorMessage = "";
          this.resetForm();
        })
        .catch((error) => {
          this.errorMessage = this.$t("contacts.add_contact_error");
          this.successMessage = "";
          console.error(error);
        });
    },
    addCircle() {
      const circle = this.circleInput.trim();
      if (circle && !this.contact.circles.includes(circle)) {
        this.contact.circles.push(circle);
      }
      this.circleInput = "";
    },
    removeCircle(index) {
      this.contact.circles.splice(index, 1);
    },
    resetForm() {
      this.contact = {
        firstname: "",
        lastname: "",
        nickname: "",
        email: "",
        phone: "",
        birthday: null, // Initialize birthday as null
        address: "",
        how_we_met: "",
        food_preference: "",
        work_information: "",
        contact_information: "",
        circles: [],
      };
      this.circleInput = "";
      this.birthdayInput = ""; // Reset birthday input
      this.birthdayError = ""; // Clear any birthday error
    },
    validateBirthday() {
      // Regular expression to match "DD.MM.YYYY" or "DD.MM." format
      const datePattern =
        /^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])\.(\d{4})?$/;
      if (this.birthdayInput && !this.birthdayInput.match(datePattern)) {
        this.birthdayError = this.$t(
          "contacts.birthday.birthday_format_warning"
        );
      } else {
        this.birthdayError = "";
        // Convert input to the YYYY-MM-DD format
        this.formatBirthday();
      }
    },
    formatBirthday() {
      const [day, month, year] = this.birthdayInput.split(".");
      this.contact.birthday = `${year || "0001"}-${month}-${day}`;
    },
  },
};
</script>
