// Custom hook for fetching and managing activities
import { useState, useEffect, useCallback } from 'react';
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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchActivities = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const token = getToken();
      if (!token) {
        throw new Error('No authentication token found');
      }

      if (contactId) {
        const data = await getContactActivities(contactId, token);
        setActivities(data.activities || []);
        setTotal(data.activities?.length || 0);
        setPage(1);
      } else {
        const data: ActivitiesResponse = await getActivities(params, token);
        setActivities(data.activities || []);
        setTotal(data.total || 0);
        setPage(data.page || 1);
      }
    } catch (err) {
      console.error('Error fetching activities:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch activities');
    } finally {
      setLoading(false);
    }
  }, [params.page, params.limit, params.includeContacts, contactId]);

  useEffect(() => {
    fetchActivities();
  }, [fetchActivities]);

  return {
    activities,
    total,
    page,
    loading,
    error,
    refetch: fetchActivities,
  };
}
