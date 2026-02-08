// Custom hook for fetching and managing notes
import { useState, useEffect, useCallback } from 'react';
import { isAuthenticated } from '../auth';
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
  // Destructure params to use primitive values as dependencies
  // This prevents re-fetches when callers pass new object references with identical values
  const { page: paramPage, limit: paramLimit, search, fromDate, toDate } = params;

  const [notes, setNotes] = useState<Note[]>([]);
  const [total, setTotal] = useState(0);
  const [pageState, setPageState] = useState(paramPage || 1);
  const [limit, setLimit] = useState(paramLimit || 25);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchNotes = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      if (!isAuthenticated()) {
        throw new Error('No authentication token found');
      }

      if (contactId) {
        const data = await getContactNotes(contactId);
        const normalized = Array.isArray(data) ? data : data.notes || [];
        setNotes(normalized);
        setTotal(normalized.length);
        setPageState(1);
        setLimit(normalized.length || paramLimit || 25);
      } else {
        const fetchParams: GetNotesParams = { page: paramPage, limit: paramLimit, search, fromDate, toDate };
        const data = await getUnassignedNotes(fetchParams);
        setNotes(data.notes || []);
        setTotal(data.total ?? data.notes?.length ?? 0);
        setPageState(data.page || paramPage || 1);
        setLimit(data.limit || paramLimit || 25);
      }
    } catch (err) {
      const message = handleFetchError(err, 'fetching notes');
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [contactId, paramPage, paramLimit, search, fromDate, toDate]);

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
