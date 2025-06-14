import { browser } from '$app/environment';
import userService from '$lib/services/userService';
import { userStore } from '$lib/stores/userStore';

// Function to initialize auth state
export function initAuth() {
  if (browser) {
    const isAuthenticated = userService.isAuthenticated();
    
    // Update user store based on token presence
    userStore.set({
      isAuthenticated,
      user: null // We could fetch user details here if needed
    });
    
    return isAuthenticated;
  }
  
  return false;
}
