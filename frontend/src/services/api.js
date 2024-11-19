import axios from 'axios';

const backendURL = "http://localhost:8080"

const apiClient = axios.create({
  baseURL: backendURL, // Go server URL
  headers: {
    'Content-Type': 'application/json',
  },
});

export { backendURL };
export default apiClient;
