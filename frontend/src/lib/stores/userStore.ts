import { writable } from 'svelte/store';

export interface User {
  id: string;
  username: string;
  email: string;
}

export interface UserState {
  isAuthenticated: boolean;
  user: User | null;
}

// Initial state
const initialState: UserState = {
  isAuthenticated: false,
  user: null
};

// Create the store
export const userStore = writable<UserState>(initialState);

// Helper to check authentication from within components
export function getIsAuthenticated(): boolean {
  let isAuthenticated = false;
  
  const unsubscribe = userStore.subscribe(state => {
    isAuthenticated = state.isAuthenticated;
  });
  
  // Unsubscribe to prevent memory leaks
  unsubscribe();
  
  return isAuthenticated;
}
