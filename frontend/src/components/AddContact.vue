<template>
    <div class="add-contact">
      <h2>Add New Contact</h2>
      <form @submit.prevent="submitForm">
        <div>
          <label for="firstname">First Name:</label>
          <input type="text" v-model="contact.firstname" id="firstname" required />
        </div>
  
        <div>
          <label for="lastname">Last Name:</label>
          <input type="text" v-model="contact.lastname" id="lastname" required />
        </div>
  
        <div>
          <label for="nickname">Nickname:</label>
          <input type="text" v-model="contact.nickname" id="nickname" />
        </div>

        <div>
          <label for="gender">Gender:</label>
          <select v-model="contact.gender" id="gender">
            <option value="Male">Male</option>
            <option value="Female">Female</option>
            <option value="Unknown">Unknown</option>
          </select>
        </div>
  
        <div>
          <label for="email">Email:</label>
          <input type="email" v-model="contact.email" id="email" required />
        </div>
  
        <div>
          <label for="phone">Phone:</label>
          <input type="tel" v-model="contact.phone" id="phone" />
        </div>
  
        <div>
          <label for="birthday">Birthday:</label>
          <input type="date" v-model="contact.birthday" id="birthday" required />
        </div>
  
        <div>
          <label for="address">Address:</label>
          <input type="text" v-model="contact.address" id="address" />
        </div>
  
        <div>
          <label for="howWeMet">How We Met:</label>
          <textarea v-model="contact.how_we_met" id="howWeMet"></textarea>
        </div>
  
        <div>
          <label for="foodPreference">Food Preference:</label>
          <input type="text" v-model="contact.food_preference" id="foodPreference" />
        </div>
  
        <div>
          <label for="workInformation">Work Information:</label>
          <input type="text" v-model="contact.work_information" id="workInformation" />
        </div>
  
        <div>
          <label for="contactInformation">Additional Contact Information:</label>
          <textarea v-model="contact.contact_information" id="contactInformation"></textarea>
        </div>
  
        <button type="submit">Add Contact</button>
      </form>
  
      <p v-if="successMessage" class="success-message">{{ successMessage }}</p>
      <p v-if="errorMessage" class="error-message">{{ errorMessage }}</p>
    </div>
  </template>
  
  <script>
  import contactsService from '@/services/contactService';
  
  export default {
    data() {
      return {
        contact: {
          firstname: '',
          lastname: '',
          nickname: '',
          gender: '',
          email: '',
          phone: '',
          birthday: '',
          address: '',
          how_we_met: '',
          food_preference: '',
          work_information: '',
          contact_information: '',
        },
        successMessage: '',
        errorMessage: '',
      };
    },
    methods: {
      submitForm() {
        contactsService
          .createContact(this.contact)
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
        };
      },
    },
  };
  </script>
  
  <style scoped>
  .add-contact {
    max-width: 600px;
    margin: auto;
  }
  
  form div {
    margin-bottom: 15px;
  }
  
  form label {
    display: block;
    font-weight: bold;
  }
  
  form input,
  form textarea {
    width: 100%;
    padding: 8px;
    box-sizing: border-box;
  }
  
  button {
    padding: 10px 20px;
    background-color: #007bff;
    color: white;
    border: none;
    cursor: pointer;
  }
  
  button:hover {
    background-color: #0056b3;
  }
  
  .success-message {
    color: green;
  }
  
  .error-message {
    color: red;
  }
  </style>
  