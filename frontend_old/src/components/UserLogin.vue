<template>
  <v-container>
    <v-card>
      <v-card-title>{{ $t("user.login_account") }}</v-card-title>
      <v-card-text>
        <v-form ref="form" v-model="valid" @submit.prevent="loginUser">
          <v-text-field
            v-model="user.email"
            :rules="[rules.required, rules.email]"
            label="Email"
            required
          />
          <v-text-field
            v-model="user.password"
            :rules="[rules.required]"
            label="Password"
            type="password"
            required
          />
          <v-btn type="submit" color="primary">{{ $t("user.login") }}</v-btn>
        </v-form>
        <v-alert v-if="errorMessage" type="error">{{ errorMessage }}</v-alert>
      </v-card-text>
    </v-card>
  </v-container>
</template>

<script>
import userService from "@/services/userService";

export default {
  data() {
    return {
      valid: false,
      errorMessage: "",
      user: {
        email: "",
        password: "",
      },
      rules: {
        required: (value) => !!value || "Required.",
        email: (value) => /.+@.+\..+/.test(value) || "E-mail must be valid.",
      },
    };
  },
  methods: {
    async loginUser() {
      try {
        const response = await userService.login(this.user);
        localStorage.setItem("token", response.token); // Store token in local storage
        this.$router.push("/contacts"); // Redirect after login
      } catch (error) {
        this.errorMessage = error.response.data.error || "Login Failed.";
      }
    },
  },
};
</script>
