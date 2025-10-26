// Custom hook for fetching and managing contacts
import { useState, useEffect, useCallback } from 'react';
import { getToken } from '../auth';
import { 
  getContacts, 
  GetContactsParams, 
  ContactsResponse,
  Contact 
} from '../api/contacts';

interface UseContactsResult {
  contacts: Contact[];
  total: number;
  page: number;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useContacts(params: GetContactsParams = {}): UseContactsResult {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(params.page || 1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchContacts = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const token = getToken();
      if (!token) {
        throw new Error('No authentication token found');
      }

      const data: ContactsResponse = await getContacts(params, token);
      setContacts(data.contacts || []);
      setTotal(data.total || 0);
      setPage(data.page || 1);
    } catch (err) {
      console.error('Error fetching contacts:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch contacts');
    } finally {
      setLoading(false);
    }
  }, [params]);

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
