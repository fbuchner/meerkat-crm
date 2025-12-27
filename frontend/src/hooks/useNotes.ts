// Custom hook for fetching and managing notes
import { useState, useEffect, useCallback } from 'react';
import { getToken } from '../auth';
import { 
  getUnassignedNotes,
  getContactNotes,
  Note,
  GetNotesParams,
} from '../api/notes';
import { handleFetchError } from '../utils/errorHandler';

interface UseNotesResult {
  notes: Note[];
  total: number;
  page: number;
  limit: number;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useNotes(
	contactId?: string | number,
	params: GetNotesParams = {}
): UseNotesResult {
  const [notes, setNotes] = useState<Note[]>([]);
  const [total, setTotal] = useState(0);
  const [pageState, setPageState] = useState(params.page || 1);
  const [limit, setLimit] = useState(params.limit || 25);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchNotes = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const token = getToken();
      if (!token) {
        throw new Error('No authentication token found');
      }

      if (contactId) {
        const data = await getContactNotes(contactId, token);
        const normalized = Array.isArray(data) ? data : data.notes || [];
        setNotes(normalized);
        setTotal(normalized.length);
        setPageState(1);
        setLimit(normalized.length || params.limit || 25);
      } else {
        const data = await getUnassignedNotes(token, params);
        setNotes(data.notes || []);
        setTotal(data.total ?? data.notes?.length ?? 0);
        setPageState(data.page || params.page || 1);
        setLimit(data.limit || params.limit || 25);
      }
    } catch (err) {
      const message = handleFetchError(err, 'fetching notes');
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [contactId, params]);

  useEffect(() => {
    fetchNotes();
  }, [fetchNotes]);

  return {
    notes,
    total,
    page: pageState,
    limit,
    loading,
    error,
    refetch: fetchNotes,
  };
}
