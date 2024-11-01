<template>
  <v-container>
    <v-row>
      <v-col>
        <v-card>
          <v-card-title>Add New Contact</v-card-title>
          <v-card-text>
            <v-form @submit.prevent="submitForm">
              <v-text-field label="First Name" v-model="contact.firstname" required></v-text-field>

              <v-text-field label="Last Name" v-model="contact.lastname"></v-text-field>

              <v-text-field label="Nickname" v-model="contact.nickname"></v-text-field>

              <v-select label="Gender" v-model="contact.gender" :items="['Male', 'Female', 'Unknown']"></v-select>

              <v-text-field label="Circles" v-model="circleInput" @keyup.space="addCircle"
                placeholder="Add a circle and press Space"></v-text-field>

              <v-chip-group v-if="contact.circles.length">
                <v-chip v-for="(circle, index) in contact.circles" :key="index" close
                  @click:close="removeCircle(index)">
                  {{ circle }}
                </v-chip>
              </v-chip-group>

              <v-text-field label="Email" v-model="contact.email" type="email"></v-text-field>

              <v-text-field label="Phone" v-model="contact.phone" type="tel"></v-text-field>

              <v-text-field
                label="Birthday"
                v-model="birthdayInput"
                placeholder="DD.MM.YYYY (year optional)"
                :error-messages="birthdayError"
                @blur="validateBirthday"
              ></v-text-field>

              <v-text-field label="Address" v-model="contact.address"></v-text-field>

              <v-textarea label="How We Met" v-model="contact.how_we_met"></v-textarea>

              <v-text-field label="Food Preference" v-model="contact.food_preference"></v-text-field>

              <v-text-field label="Work Information" v-model="contact.work_information"></v-text-field>

              <v-textarea label="Additional Contact Information" v-model="contact.contact_information"></v-textarea>

              <v-btn type="submit" color="primary">Add Contact</v-btn>

              <v-alert v-if="successMessage" type="success" dismissible class="mt-3">
                {{ successMessage }}
              </v-alert>
              <v-alert v-if="errorMessage" type="error" dismissible class="mt-3">
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
import contactService from '@/services/contactService';

export default {
  data() {
    return {
      contact: {
        firstname: '',
        lastname: '',
        nickname: '',
        gender: 'Unknown',
        email: '',
        phone: '',
        birthday: null,
        address: '',
        how_we_met: '',
        food_preference: '',
        work_information: '',
        contact_information: '',
        circles: [],
      },
      successMessage: '',
      errorMessage: '',
      circleInput: '',
      birthdayInput: '', 
      birthdayError: '', 
   
    };
  },
  methods: {
    submitForm() {
      this.validateBirthday();
      if (this.birthdayError) {
        this.errorMessage = "Please correct the birthday format.";
        return;
      }
      contactService
        .addContact(this.contact)
        .then(() => {
          this.successMessage = 'Contact added successfully!';
          this.errorMessage = '';
          this.resetForm();
        })
        .catch((error) => {
          this.errorMessage = 'Failed to add contact. Please try again.';
          this.successMessage = '';
          console.error(error);
        });
    },
    addCircle() {
      const circle = this.circleInput.trim();
      if (circle && !this.contact.circles.includes(circle)) {
        this.contact.circles.push(circle);
      }
      this.circleInput = '';
    },
    removeCircle(index) {
      this.contact.circles.splice(index, 1);
    },
    resetForm() {
      this.contact = {
        firstname: '',
        lastname: '',
        nickname: '',
        email: '',
        phone: '',
        birthday: '',
        address: '',
        how_we_met: '',
        food_preference: '',
        work_information: '',
        contact_information: '',
        circles: [],
      };
      this.circleInput = '';
    },
    validateBirthday() {
      // Regular expression to match "DD.MM.YYYY" or "DD.MM." format
      const datePattern = /^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])\.(\d{4})?$/;
      if (!this.birthdayInput.match(datePattern)) {
        this.birthdayError = "Please enter a valid date in DD.MM.YYYY or DD.MM. format.";
      } else {
        this.birthdayError = '';
        // Convert input to the YYYY-MM-DD format, setting the year to 0001 if omitted
        this.formatBirthday();
      }
    },
    formatBirthday() {
      const [day, month, year] = this.birthdayInput.split('.');
      this.contact.birthday = `${year || '0001'}-${month}-${day}`;
    },

  },
};
</script>
