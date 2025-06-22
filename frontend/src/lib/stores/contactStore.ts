import { writable } from 'svelte/store';
import type { Contact, ContactsResponse } from '$lib/services/contactService';

export const contactsStore = writable<{
  contacts: Contact[];
  total: number;
  page: number;
  limit: number;
  loading: boolean;
  error: string | null;
}>({
  contacts: [],
  total: 0,
  page: 1,
  limit: 25,
  loading: false,
  error: null
});

export const selectedContact = writable<Contact | null>(null);

export const contactFilters = writable({
  search: '',
  circle: '',
  page: 1,
  limit: 25
});

export const circlesStore = writable<string[]>([]);
