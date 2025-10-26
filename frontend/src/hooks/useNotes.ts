// Custom hook for fetching and managing notes
import { useState, useEffect, useCallback } from 'react';
import { getToken } from '../auth';
import { 
  getUnassignedNotes,
  getContactNotes,
  Note,
  NotesResponse 
} from '../api/notes';

interface UseNotesResult {
  notes: Note[];
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useNotes(contactId?: string | number): UseNotesResult {
  const [notes, setNotes] = useState<Note[]>([]);
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

      let data: Note[] | NotesResponse;
      if (contactId) {
        data = await getContactNotes(contactId, token);
        setNotes(Array.isArray(data) ? data : data.notes || []);
      } else {
        data = await getUnassignedNotes(token);
        setNotes(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      console.error('Error fetching notes:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch notes');
    } finally {
      setLoading(false);
    }
  }, [contactId]);

  useEffect(() => {
    fetchNotes();
  }, [fetchNotes]);

  return {
    notes,
    loading,
    error,
    refetch: fetchNotes,
  };
}
