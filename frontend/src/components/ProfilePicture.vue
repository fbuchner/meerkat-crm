<template>
  <v-img
    v-if="imageSrc"
    :src="imageSrc"
    :alt="alt"
    :width="width"
    :height="height"
    class="profile-picture"
  ></v-img>
  <div v-else class="profile-picture"></div>
</template>

<script>
import { backendURL } from "@/services/api";

export default {
  name: "ProfilePicture",
  props: {
    contactId: {
      type: [Number, String],
      required: true,
    },
    width: {
      type: [Number, String],
      default: 100,
    },
    height: {
      type: [Number, String],
      default: 100,
    },
    alt: {
      type: String,
      default: "Profile Picture",
    },
  },
  data() {
    return {
      imageSrc: null,
    };
  },
  mounted() {
    this.fetchProfilePicture();
  },
  methods: {
    async fetchProfilePicture() {
      // Read the token from localStorage, similar to your Axios interceptors.
      const token = localStorage.getItem("token");
      const url = `${backendURL}/contacts/${this.contactId}/profile_picture`;
      try {
        // Fetch the image using fetch API with Authorization header.
        const response = await fetch(url, {
          method: "GET",
          headers: {
            // Only add the header if token exists.
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
        });
        if (!response.ok) {
          throw new Error("Failed to fetch profile picture");
        }
        const blob = await response.blob();
        this.imageSrc = URL.createObjectURL(blob);
      } catch (error) {
        console.error("Error fetching profile picture:", error);
        this.imageSrc = require("@/assets/placeholder-avatar.png");
      }
    },
  },
};
</script>

<style scoped>
.profile-picture {
  border-radius: 4px;
  object-fit: cover;
}
</style>
