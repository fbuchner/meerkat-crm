// Custom hook for fetching suggested duplicate contacts
import { useState, useEffect, useCallback, useRef } from 'react';
import { getDuplicateContacts, DuplicatePair } from '../api/contacts';
import { handleFetchError } from '../utils/errorHandler';

interface UseDuplicateContactsResult {
  data: DuplicatePair[];
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useDuplicateContacts(): UseDuplicateContactsResult {
  const [data, setData] = useState<DuplicatePair[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const requestRef = useRef(0);

  const fetchDuplicates = useCallback(async () => {
    const requestId = requestRef.current + 1;
    requestRef.current = requestId;
    setLoading(true);
    setError(null);

    try {
      const duplicates = await getDuplicateContacts();
      if (requestRef.current !== requestId) {
        return;
      }
      setData(duplicates);
    } catch (err) {
      if (requestRef.current !== requestId) {
        return;
      }
      setError(handleFetchError(err, 'fetching duplicate contacts'));
    } finally {
      if (requestRef.current === requestId) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    fetchDuplicates();
  }, [fetchDuplicates]);

  return {
    data,
    loading,
    error,
    refetch: fetchDuplicates,
  };
}
