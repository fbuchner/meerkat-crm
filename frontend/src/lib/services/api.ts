import { PUBLIC_API_URL } from '$env/static/public';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';

// Create a base API client
const apiBaseUrl = PUBLIC_API_URL || '/';

// Utility for handling API responses
async function handleResponse(response: Response) {
  if (!response.ok) {
    // If unauthorized, redirect to login (token expired)
    if (response.status === 401) {
      if (browser) {
        localStorage.removeItem('token');
        goto('/login');
      }
    }
    
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || `API error ${response.status}`);
  }
  return response.json();
}

// Base API request function
async function apiRequest(
  endpoint: string, 
  method: string = 'GET', 
  data: any = null, 
  options: RequestInit = {}
) {
  const url = `${apiBaseUrl}${endpoint}`;
  
  // Create a clean headers object
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  
  // Safely add headers from options if they exist
  if (options.headers) {
    // Convert HeadersInit to a standard object if needed
    const optionHeaders = options.headers as Record<string, string>;
    Object.keys(optionHeaders).forEach(key => {
      headers[key] = optionHeaders[key];
    });
  }

  // Add authorization token if it exists
  if (browser) {
    const token = localStorage.getItem('token');
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
  }

  const config: RequestInit = {
    method,
    headers,
    ...options,
  };

  if (data && method !== 'GET') {
    config.body = JSON.stringify(data);
  }

  const response = await fetch(url, config);
  return handleResponse(response);
}

export default {
  get: (endpoint: string, options = {}) => apiRequest(endpoint, 'GET', null, options),
  post: (endpoint: string, data: any, options = {}) => apiRequest(endpoint, 'POST', data, options),
  put: (endpoint: string, data: any, options = {}) => apiRequest(endpoint, 'PUT', data, options),
  delete: (endpoint: string, options = {}) => apiRequest(endpoint, 'DELETE', null, options),
};
