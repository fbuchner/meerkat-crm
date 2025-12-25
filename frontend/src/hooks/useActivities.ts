// Custom hook for fetching and managing activities
import { useState, useEffect, useCallback, useRef } from 'react';
import { getToken } from '../auth';
import { 
  getActivities, 
  getContactActivities,
  GetActivitiesParams, 
  ActivitiesResponse,
  Activity 
} from '../api/activities';

interface UseActivitiesResult {
  activities: Activity[];
  total: number;
  page: number;
  limit: number;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useActivities(
  params: GetActivitiesParams = {},
  contactId?: string | number
): UseActivitiesResult {
  const [activities, setActivities] = useState<Activity[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(params.page || 1);
  const [limit, setLimit] = useState(params.limit || 25);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const requestRef = useRef(0);

  const fetchActivities = useCallback(async () => {
    const requestId = requestRef.current + 1;
    requestRef.current = requestId;
    setLoading(true);
    setError(null);

    try {
      const token = getToken();
      if (!token) {
        throw new Error('No authentication token found');
      }

      if (contactId) {
        const data = await getContactActivities(contactId, token);
        if (requestRef.current !== requestId) {
          return;
        }
        setActivities(data.activities || []);
        setTotal(data.activities?.length || 0);
        setPage(1);
        setLimit(params.limit || data.activities?.length || 25);
      } else {
        const data: ActivitiesResponse = await getActivities(params, token);
        if (requestRef.current !== requestId) {
          return;
        }
        setActivities(data.activities || []);
        setTotal(data.total || 0);
        setPage(data.page || 1);
        setLimit(data.limit || params.limit || 25);
      }
    } catch (err) {
      if (requestRef.current !== requestId) {
        return;
      }
      console.error('Error fetching activities:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch activities');
    } finally {
      if (requestRef.current === requestId) {
        setLoading(false);
      }
    }
  }, [params, contactId]);

  useEffect(() => {
    fetchActivities();
  }, [fetchActivities]);

  return {
    activities,
    total,
    page,
    limit,
    loading,
    error,
    refetch: fetchActivities,
  };
}
