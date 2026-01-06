// Reminder-related API calls
import { apiFetch, API_BASE_URL, getAuthHeaders } from './client';

export interface Reminder {
  ID: number;
  message: string;
  by_mail: boolean;
  remind_at: string; // ISO date string
  recurrence: 'once' | 'weekly' | 'monthly' | 'quarterly' | 'six-months' | 'yearly';
  reoccur_from_completion: boolean;
  completed: boolean;
  email_sent: boolean;
  last_sent?: string; // ISO date string
  contact_id: number;
  CreatedAt?: string;
  UpdatedAt?: string;
  DeletedAt?: string | null;
}

export interface ReminderFormData {
  message: string;
  by_mail: boolean;
  remind_at: string; // ISO date string
  recurrence: 'once' | 'weekly' | 'monthly' | 'quarterly' | 'six-months' | 'yearly';
  reoccur_from_completion: boolean;
  contact_id: number;
}

export interface RemindersResponse {
  reminders: Reminder[];
}

// Get all reminders across all contacts
export async function getAllReminders(token: string): Promise<Reminder[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/reminders`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch reminders');
  }

  const data: RemindersResponse = await response.json();
  return data.reminders || [];
}

// Get upcoming reminders (next 7 days or at least next 10 reminders)
export async function getUpcomingReminders(token: string): Promise<Reminder[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/reminders/upcoming`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch upcoming reminders');
  }

  const data: RemindersResponse = await response.json();
  return data.reminders || [];
}

// Get reminders for a specific contact
export async function getRemindersForContact(contactId: number, token: string): Promise<Reminder[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/reminders`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch reminders for contact');
  }

  const data: RemindersResponse = await response.json();
  return data.reminders || [];
}

// Get a single reminder
export async function getReminder(reminderId: number, token: string): Promise<Reminder> {
  const response = await apiFetch(
    `${API_BASE_URL}/reminders/${reminderId}`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch reminder');
  }

  return response.json();
}

// Create a new reminder
export async function createReminder(
  contactId: number,
  reminderData: ReminderFormData,
  token: string
): Promise<Reminder> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/reminders`,
    {
      method: 'POST',
      headers: getAuthHeaders(token),
      body: JSON.stringify(reminderData),
    }
  );

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || 'Failed to create reminder');
  }

  const data = await response.json();
  return data.reminder;
}

// Update an existing reminder
export async function updateReminder(
  reminderId: number,
  reminderData: Partial<ReminderFormData>,
  token: string
): Promise<Reminder> {
  const response = await apiFetch(
    `${API_BASE_URL}/reminders/${reminderId}`,
    {
      method: 'PUT',
      headers: getAuthHeaders(token),
      body: JSON.stringify(reminderData),
    }
  );

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || 'Failed to update reminder');
  }

  const data = await response.json();
  return data.reminder;
}

// Complete a reminder (marks as done and reschedules if recurring)
export async function completeReminder(
  reminderId: number,
  token: string
): Promise<{ message: string; reminder?: Reminder }> {
  const response = await apiFetch(
    `${API_BASE_URL}/reminders/${reminderId}/complete`,
    {
      method: 'POST',
      headers: getAuthHeaders(token),
    }
  );

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || 'Failed to complete reminder');
  }

  return response.json();
}

// Delete a reminder
export async function deleteReminder(reminderId: number, token: string): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/reminders/${reminderId}`,
    {
      method: 'DELETE',
      headers: getAuthHeaders(token),
    }
  );

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || 'Failed to delete reminder');
  }
}
