<script lang="ts">
  import { 
    Button, 
    Card, 
    Label, 
    Input, 
    Select, 
    Textarea, 
    Alert, 
    Badge 
  } from 'flowbite-svelte';
  import { contactService, type Contact } from '$lib/services/contactService';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  // Initial contact state
  let contact: Partial<Contact> = {
    firstname: "",
    lastname: "",
    nickname: "",
    gender: "Unknown",
    email: "",
    phone: "",
    birthday: "",
    address: "",
    how_we_met: "",
    food_preference: "",
    work_information: "",
    contact_information: "",
    circles: [],
  };

  // Form state
  let circleInput = "";
  let birthdayInput = "";
  let birthdayError = "";
  let successMessage = "";
  let errorMessage = "";
  let genders = ["Unknown", "Male", "Female", "Non-binary", "Other"];

  // Form submission handler
  async function submitForm() {
    validateBirthday();

    // If birthdayInput is empty, set contact.birthday to null
    if (!birthdayInput) {
      contact.birthday = "";
      birthdayError = ""; // Clear previous error (if any)
    } else if (birthdayError) {
      errorMessage = "Please fix the birthday format error before submitting.";
      return;
    }

    try {
      await contactService.addContact(contact);
      successMessage = "Contact added successfully!";
      errorMessage = "";
      resetForm();
      dispatch('contactAdded');
    } catch (error) {
      errorMessage = "Error adding contact. Please try again.";
      successMessage = "";
      console.error(error);
    }
  }

  // Circle management
  function addCircle() {
    const circle = circleInput.trim();
    if (circle && contact.circles && !contact.circles.includes(circle)) {
      contact.circles = [...(contact.circles || []), circle];
    }
    circleInput = "";
  }

  function removeCircle(index: number) {
    if(contact.circles) {
      contact.circles = contact.circles.filter((_, i) => i !== index);
    }
  }

  // Handle space key for adding circles
  function handleCircleInput(event: KeyboardEvent) {
    if (event.key === ' ' && circleInput.trim()) {
      event.preventDefault();
      addCircle();
    }
  }

  // Reset form to initial state
  function resetForm() {
    contact = {
      firstname: "",
      lastname: "",
      nickname: "",
      gender: "Unknown",
      email: "",
      phone: "",
      birthday: "",
      address: "",
      how_we_met: "",
      food_preference: "",
      work_information: "",
      contact_information: "",
      circles: [],
    };
    circleInput = "";
    birthdayInput = "";
    birthdayError = "";
  }

  // Birthday validation
  function validateBirthday() {
    // Regular expression to match "DD.MM.YYYY" or "DD.MM." format
    const datePattern = /^(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])\.(\d{4})?$/;
    
    if (birthdayInput && !datePattern.test(birthdayInput)) {
      birthdayError = "Please use DD.MM.YYYY or DD.MM. format";
    } else {
      birthdayError = "";
      // Convert input to the YYYY-MM-DD format if not empty
      if (birthdayInput) {
        formatBirthday();
      }
    }
  }

  // Format birthday to server expected format
  function formatBirthday() {
    if (!birthdayInput) return;
    
    const parts = birthdayInput.split(".");
    if (parts.length >= 2) {
      const day = parts[0];
      const month = parts[1];
      const year = parts[2] || "0001";
      contact.birthday = `${year}-${month}-${day}`;
    }
  }
</script>

<Card class="max-w-2xl mx-auto p-6 bg-white shadow-md rounded-lg">
  <h2 class="text-2xl font-bold mb-4">Add New Contact</h2>
  
  <form on:submit|preventDefault={submitForm} class="space-y-4">
    <!-- First Name -->
    <div>
      <Label for="firstname" class="mb-2">First Name <span class="text-red-500">*</span></Label>
      <Input id="firstname" required bind:value={contact.firstname} />
    </div>

    <!-- Last Name -->
    <div>
      <Label for="lastname" class="mb-2">Last Name</Label>
      <Input id="lastname" bind:value={contact.lastname} />
    </div>

    <!-- Nickname -->
    <div>
      <Label for="nickname" class="mb-2">Nickname</Label>
      <Input id="nickname" bind:value={contact.nickname} />
    </div>

    <!-- Gender -->
    <div>
      <Label for="gender" class="mb-2">Gender</Label>
      <Select id="gender" bind:value={contact.gender}>
        {#each genders as gender}
          <option value={gender}>{gender}</option>
        {/each}
      </Select>
    </div>

    <!-- Circles -->
    <div>
      <Label for="circles" class="mb-2">Circles</Label>
      <Input 
        id="circles" 
        bind:value={circleInput} 
        onkeydown={handleCircleInput}
        placeholder="Type and press space to add circles"
      />
      
      {#if contact.circles && contact.circles.length > 0}
        <div class="mt-2 flex flex-wrap gap-2">
          {#each contact.circles as circle, index}
            <Badge 
              color="blue" 
              class="cursor-pointer flex items-center"
            >
              {circle}
              <button 
                type="button" 
                class="ml-1.5 inline-flex items-center p-0.5 text-sm bg-transparent rounded-sm"
                on:click={() => removeCircle(index)}
                aria-label="Remove circle"
              >
                <svg class="w-3.5 h-3.5" aria-hidden="true" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
                  <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"></path>
                </svg>
              </button>
            </Badge>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Email -->
    <div>
      <Label for="email" class="mb-2">Email</Label>
      <Input id="email" type="email" bind:value={contact.email} />
    </div>

    <!-- Phone -->
    <div>
      <Label for="phone" class="mb-2">Phone</Label>
      <Input id="phone" type="tel" bind:value={contact.phone} />
    </div>

    <!-- Birthday -->
    <div>
      <Label for="birthday" class="mb-2">Birthday</Label>
      <Input 
        id="birthday" 
        bind:value={birthdayInput} 
        placeholder="DD.MM.YYYY or DD.MM."
        onblur={validateBirthday}
        color={birthdayError ? "red" : "primary"}
      />
      {#if birthdayError}
        <p class="text-red-500 text-sm mt-1">{birthdayError}</p>
      {/if}
    </div>

    <!-- Address -->
    <div>
      <Label for="address" class="mb-2">Address</Label>
      <Input id="address" bind:value={contact.address} />
    </div>

    <!-- How We Met -->
    <div>
      <Label for="how_we_met" class="mb-2">How We Met</Label>
      <Textarea id="how_we_met" bind:value={contact.how_we_met} rows={3} />
    </div>

    <!-- Food Preference -->
    <div>
      <Label for="food_preference" class="mb-2">Food Preference</Label>
      <Input id="food_preference" bind:value={contact.food_preference} />
    </div>

    <!-- Work Information -->
    <div>
      <Label for="work_information" class="mb-2">Work Information</Label>
      <Input id="work_information" bind:value={contact.work_information} />
    </div>

    <!-- Additional Information -->
    <div>
      <Label for="contact_information" class="mb-2">Additional Information</Label>
      <Textarea id="contact_information" bind:value={contact.contact_information} rows={4} />
    </div>

    <!-- Submit Button -->
    <div class="mt-6">
      <Button type="submit" color="blue">Add Contact</Button>
    </div>

    <!-- Success/Error Messages -->
    {#if successMessage}
      <Alert color="green" dismissable>{successMessage}</Alert>
    {/if}
    
    {#if errorMessage}
      <Alert color="red" dismissable>{errorMessage}</Alert>
    {/if}
  </form>
</Card>
