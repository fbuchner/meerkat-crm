// src/auth.ts
// Cookie-based JWT auth service for mycorrhizal crm frontend
// Token is stored in httpOnly cookie (not accessible from JS for security)
// User info is cached in localStorage for UI purposes

const API_SERVER_URL = process.env.REACT_APP_API_URL || '';
export const API_BASE_URL = `${API_SERVER_URL}/api/v1`;

const USER_INFO_KEY = 'user_info';

export interface LoginResponse {
  language?: string;
  date_format?: string;
}

export interface LogoutResponse {
  redirect_url?: string;
}

export interface UserInfo {
  user_id: number;
  username: string;
  is_admin: boolean;
}

export async function loginUser(identifier: string, password: string): Promise<LoginResponse> {
  const response = await fetch(`${API_BASE_URL}/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include', // Send and receive cookies
    body: JSON.stringify({ identifier, password }),
  });
  if (!response.ok) {
    throw new Error('Login failed');
  }
  const data = await response.json();

  // Fetch user info and cache it (since we can't read the httpOnly cookie)
  await fetchAndCacheUserInfo();

  return {
    language: data.language,
    date_format: data.date_format,
  };
}

// Fetch current user info from the server and cache it
export async function fetchAndCacheUserInfo(): Promise<UserInfo | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/users/me`, {
      credentials: 'include',
    });
    if (!response.ok) {
      return null;
    }
    const data = await response.json();
    const userInfo: UserInfo = {
      user_id: data.ID,
      username: data.Username,
      is_admin: data.is_admin || false,
    };
    localStorage.setItem(USER_INFO_KEY, JSON.stringify(userInfo));
    return userInfo;
  } catch {
    return null;
  }
}

// No-op: token is now in httpOnly cookie, not stored in localStorage
export function saveToken(_token: string) {
  // Kept for backward compatibility - token is now set via httpOnly cookie by server
}

// Returns null - token is in httpOnly cookie (not accessible from JS)
// Use isAuthenticated() to check login status
export function getToken(): string | null {
  // Return a placeholder if user info exists (indicates logged in)
  // This maintains compatibility with existing code that checks for token presence
  const userInfo = localStorage.getItem(USER_INFO_KEY);
  return userInfo ? 'cookie-auth' : null;
}

// Check if user is authenticated (has cached user info)
export function isAuthenticated(): boolean {
  return localStorage.getItem(USER_INFO_KEY) !== null;
}

// Returns the provider's RP-Initiated Logout URL when this session was
// authenticated via OIDC, so the caller can navigate there to also end the
// IdP's own session (otherwise "Sign in with SSO" silently re-authenticates
// without a prompt).
export async function logoutUser(): Promise<string | undefined> {
  let redirectURL: string | undefined;
  try {
    const response = await fetch(`${API_BASE_URL}/logout`, {
      method: 'POST',
      credentials: 'include',
    });
    const data: LogoutResponse | null = await response.json().catch(() => null);
    redirectURL = data?.redirect_url;
  } catch {
    // Ignore errors - clear local state anyway
  }
  localStorage.removeItem(USER_INFO_KEY);
  return redirectURL;
}

export async function logoutAndRedirect() {
  const redirectURL = await logoutUser();
  // A real top-level navigation (not fetch) is required so the IdP can clear
  // its own first-party session cookie.
  window.location.href = redirectURL || '/login';
}

interface DecodedToken {
  user_id: number;
  username: string;
  is_admin: boolean;
  exp: number;
}

// Returns cached user info (previously decoded from token)
export function decodeToken(): DecodedToken | null {
  const userInfoStr = localStorage.getItem(USER_INFO_KEY);
  if (!userInfoStr) {
    return null;
  }

  try {
    const userInfo: UserInfo = JSON.parse(userInfoStr);
    return {
      user_id: userInfo.user_id,
      username: userInfo.username,
      is_admin: userInfo.is_admin,
      exp: 0, // Expiry handled server-side via cookie
    };
  } catch {
    return null;
  }
}

// Check if the current user is an admin based on cached user info
export function isAdmin(): boolean {
  const decoded = decodeToken();
  return decoded?.is_admin || false;
}
