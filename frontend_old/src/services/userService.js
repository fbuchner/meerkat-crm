import apiClient from "@/services/api";

export default {
  async register(userData) {
    try {
      const response = await apiClient.post(`/register`, userData);
      return response.data;
    } catch (error) {
      console.error("Error during registration:", error);
      throw error;
    }
  },

  async login(userData) {
    try {
      const response = await apiClient.post(`/login`, userData);
      localStorage.setItem("token", response.token); // Store JWT token
      return response.data;
    } catch (error) {
      console.error("Error during login:", error);
      throw error;
    }
  },

  logout() {
    localStorage.removeItem("token");
  },
};
