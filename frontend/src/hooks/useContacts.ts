// Custom hook for fetching and managing contacts
import { useState, useEffect, useCallback } from 'react';
import { isAuthenticated } from '../auth';
import {
  getContacts,
  GetContactsParams,
  ContactsResponse,
  Contact
} from '../api/contacts';
import { handleFetchError } from '../utils/errorHandler';

interface UseContactsResult {
  contacts: Contact[];
  total: number;
  page: number;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useContacts(params: GetContactsParams = {}): UseContactsResult {
  // Destructure params to use primitive values as dependencies
  // This prevents re-fetches when callers pass new object references with identical values
  const { page: paramPage, limit, search, circle, sort, order, includeArchived, archived } = params;

  const [contacts, setContacts] = useState<Contact[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(paramPage || 1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchContacts = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      if (!isAuthenticated()) {
        throw new Error('No authentication token found');
      }

      const fetchParams: GetContactsParams = {
        page: paramPage,
        limit,
        search,
        circle,
        sort,
        order,
        includeArchived,
        archived,
      };
      const data: ContactsResponse = await getContacts(fetchParams);
      setContacts(data.contacts || []);
      setTotal(data.total || 0);
      setPage(data.page || 1);
    } catch (err) {
      const message = handleFetchError(err, 'fetching contacts');
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [paramPage, limit, search, circle, sort, order, includeArchived, archived]);

  useEffect(() => {
    fetchContacts();
  }, [fetchContacts]);

  return {
    contacts,
    total,
    page,
    loading,
    error,
    refetch: fetchContacts,
  };
}
