import apiClient from './api';
import { browser } from '$app/environment';
import { userStore } from '$lib/stores/userStore';

interface LoginData {
  email: string;
  password: string;
}

interface RegisterData {
  username: string;
  email: string;
  password: string;
}

interface UserResponse {
  id: string;
  username: string;
  email: string;
  token: string;
}

export default {
  async register(userData: RegisterData): Promise<UserResponse> {
    try {
      const response = await apiClient.post('/register', userData);
      return response;
    } catch (error) {
      console.error('Error during registration:', error);
      throw error;
    }
  },

  async login(userData: LoginData): Promise<UserResponse> {
    try {
      const response = await apiClient.post('/login', userData);
      
      if (browser && response.token) {
        // Store JWT token in localStorage
        localStorage.setItem('token', response.token);
        
        // Update user store with authenticated user
        userStore.set({
          isAuthenticated: true,
          user: {
            id: response.id,
            username: response.username,
            email: response.email
          }
        });
      }
      
      return response;
    } catch (error) {
      console.error('Error during login:', error);
      throw error;
    }
  },

  logout(): void {
    if (browser) {
      localStorage.removeItem('token');
      
      // Update user store on logout
      userStore.set({
        isAuthenticated: false,
        user: null
      });
    }
  },

  // Check if user is authenticated
  isAuthenticated(): boolean {
    if (!browser) return false;
    return !!localStorage.getItem('token');
  }
};
