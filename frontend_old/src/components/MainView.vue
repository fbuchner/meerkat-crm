<template>
  <v-container>
    <AuthWrapper />
  </v-container>
</template>

<script>
import { onMounted } from "vue";
import { useRouter } from "vue-router";
import AuthWrapper from "@/components/AuthWrapper.vue";

export default {
  name: "MainView",
  components: {
    AuthWrapper,
  },
  setup() {
    const router = useRouter();

    onMounted(() => {
      const token = localStorage.getItem("token");

      // If a token exists, redirect to the contact list.
      if (token) {
        router.push("/contacts");
      }
    });
  },
};
</script>
