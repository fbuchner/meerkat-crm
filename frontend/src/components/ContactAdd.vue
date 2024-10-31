<template>
  <v-container>
    <v-row>
      <v-col>
        <v-card>
          <v-card-title>Add New Contact</v-card-title>
          <v-card-text>
            <v-form @submit.prevent="submitForm">
              <v-text-field
                label="First Name"
                v-model="contact.firstname"
                required
              ></v-text-field>

              <v-text-field
                label="Last Name"
                v-model="contact.lastname"
              ></v-text-field>

              <v-text-field
                label="Nickname"
                v-model="contact.nickname"
              ></v-text-field>

              <v-select
                label="Gender"
                v-model="contact.gender"
                :items="['Male', 'Female', 'Unknown']"
              ></v-select>

              <v-text-field
                label="Circles"
                v-model="circleInput"
                @keyup.space="addCircle"
                placeholder="Add a circle and press Space"
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
                label="Email"
                v-model="contact.email"
                type="email"
              ></v-text-field>

              <v-text-field
                label="Phone"
                v-model="contact.phone"
                type="tel"
              ></v-text-field>

              <v-menu
                ref="menu"
                v-model="menu"
                :close-on-content-click="false"
                transition="scale-transition"
                offset-y
                min-width="290px"
              >
                <template v-slot:activator="{ on, attrs }">
                  <v-text-field
                    v-model="formattedBirthday"
                    label="Birthday"
                    prepend-icon="mdi-calendar"
                    readonly
                    v-bind="attrs"
                    v-on="on"
                  ></v-text-field>
                </template>
                <v-date-picker 
                  v-model="contact.birthday" 
                  no-title 
                  @input="updateBirthday"
                ></v-date-picker>
              </v-menu>


              <v-text-field
                label="Address"
                v-model="contact.address"
              ></v-text-field>

              <v-textarea
                label="How We Met"
                v-model="contact.how_we_met"
              ></v-textarea>

              <v-text-field
                label="Food Preference"
                v-model="contact.food_preference"
              ></v-text-field>

              <v-text-field
                label="Work Information"
                v-model="contact.work_information"
              ></v-text-field>

              <v-textarea
                label="Additional Contact Information"
                v-model="contact.contact_information"
              ></v-textarea>

              <v-btn type="submit" color="primary">Add Contact</v-btn>

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
        birthday: '', // stores the selected date in YYYY-MM-DD format
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
      menu: false, // controls the open/close state of the date picker menu
      formattedBirthday: '', // formatted birthday for display
    };
  },
  methods: {
    submitForm() {
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
      this.formattedBirthday = '';
    },
    updateBirthday(date) {
      // Update the contact's birthday and format it for display
      this.contact.birthday = date;
      this.formattedBirthday = this.formatDate(date);
      this.menu = false;
    },
    formatDate(date) {
      const options = { year: 'numeric', month: 'long', day: 'numeric' };
      return new Date(date).toLocaleDateString(undefined, options);
    },
  },
};
</script>
