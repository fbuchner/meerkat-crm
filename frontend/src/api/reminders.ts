// Reminder-related API calls
import { apiFetch, API_BASE_URL, getAuthHeaders, parseErrorResponse } from './client';

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

export interface ReminderCompletion {
  ID: number;
  reminder_id?: number;
  contact_id: number;
  message: string;
  completed_at: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export interface CompletionsResponse {
  completions: ReminderCompletion[];
}

// Get all reminders across all contacts
export async function getAllReminders(token: string): Promise<Reminder[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/reminders`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
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
    throw await parseErrorResponse(response);
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
    throw await parseErrorResponse(response);
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
    throw await parseErrorResponse(response);
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
    throw await parseErrorResponse(response);
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
    throw await parseErrorResponse(response);
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
    throw await parseErrorResponse(response);
  }

  return response.json();
}

// Skip a reminder (reschedules recurring reminders without recording completion)
export async function skipReminder(
  reminderId: number,
  token: string
): Promise<{ message: string; reminder?: Reminder }> {
  const response = await apiFetch(
    `${API_BASE_URL}/reminders/${reminderId}/complete?skip=true`,
    {
      method: 'POST',
      headers: getAuthHeaders(token),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
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
    throw await parseErrorResponse(response);
  }
}

// Get reminder completions for a specific contact
export async function getCompletionsForContact(contactId: number, token: string): Promise<ReminderCompletion[]> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts/${contactId}/reminder-completions`,
    { headers: getAuthHeaders(token) }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  const data: CompletionsResponse = await response.json();
  return data.completions || [];
}

// Delete a reminder completion
export async function deleteCompletion(completionId: number, token: string): Promise<void> {
  const response = await apiFetch(
    `${API_BASE_URL}/reminder-completions/${completionId}`,
    {
      method: 'DELETE',
      headers: getAuthHeaders(token),
    }
  );

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }
}
