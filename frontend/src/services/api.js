import axios from "axios";

const backendURL = process.env.VUE_APP_BACKEND_URL || "http://localhost:8080";

const apiClient = axios.create({
  baseURL: backendURL, // Go server URL
  headers: {
    "Content-Type": "application/json",
  },
});

// Add auth token to all requests
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem("token");
    if (token) {
      config.headers["Authorization"] = `Bearer ${token}`; // Attach token to all requests
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Handle expired token
apiClient.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    const { response } = error;
    if (response && response.status === 401) {
      // Handle unauthorized access
      console.error("WARN: Token expired or unauthorized. Logging out...");
      localStorage.removeItem("token");
      this.$router.push("/login"); // Redirect after login
    }
    return Promise.reject(error);
  }
);
export { backendURL };
export default apiClient;
