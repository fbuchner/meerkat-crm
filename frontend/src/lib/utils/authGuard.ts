import { redirect } from '@sveltejs/kit';
import type { RequestEvent } from '@sveltejs/kit';
import { browser } from '$app/environment';

// This function will check if the user is authenticated
// and redirect to login if not
export function checkAuth(event: RequestEvent) {
  if (browser) {
    const token = localStorage.getItem('token');
    
    if (!token) {
      throw redirect(303, '/login');
    }
  }
}
