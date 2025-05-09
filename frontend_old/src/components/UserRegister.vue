<template>
  <v-container>
    <v-card>
      <v-card-title>{{ $t("user.register_account") }}</v-card-title>
      <v-card-text>
        <v-form ref="form" v-model="valid" @submit.prevent="registerUser">
          <v-text-field
            v-model="user.username"
            :rules="[rules.required]"
            label="Username"
            required
          />
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
          <v-btn type="submit" color="primary">{{
            $t("user.register_account")
          }}</v-btn>
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
        username: "",
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
    async registerUser() {
      try {
        await userService.register(this.user);
        this.$router.push("/login"); // Redirect to login on successful registration
      } catch (error) {
        this.errorMessage = error.response.data.error || "Registration Failed.";
      }
    },
  },
};
</script>
