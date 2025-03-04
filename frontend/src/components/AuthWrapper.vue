<template>
  <v-container>
    <v-row align="center" justify="center">
      <v-col cols="12" md="6">
        <v-tabs
          v-model="selectedTabIndex"
          background-color="transparent"
          slider-color="primary"
          class="auth-tabs"
        >
          <v-tab @click="setTab(0)">{{ $t("user.register_account") }}</v-tab>
          <v-tab @click="setTab(1)">{{ $t("user.login") }}</v-tab>
        </v-tabs>

        <component :is="getSelectedComponent">
          <UserRegister v-if="selectedTabIndex === 0" />
          <UserLogin v-else-if="selectedTabIndex === 1" />
        </component>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import UserRegister from "./UserRegister.vue";
import UserLogin from "./UserLogin.vue";

export default {
  name: "AuthWrapper",
  components: {
    UserRegister,
    UserLogin,
  },
  data() {
    return {
      selectedTabIndex: 1, // 0 - Register, 1 - Login
    };
  },
  methods: {
    setTab(index) {
      this.selectedTabIndex = index;
    },
  },
  computed: {
    getSelectedComponent() {
      return this.selectedTabIndex === 0 ? "UserRegister" : "UserLogin";
    },
  },
};
</script>
