// Custom hook for fetching and managing activities
import { useState, useEffect, useCallback, useRef } from 'react';
import { isAuthenticated } from '../auth';
import {
  getActivities,
  getContactActivities,
  GetActivitiesParams,
  ActivitiesResponse,
  Activity
} from '../api/activities';
import { handleFetchError } from '../utils/errorHandler';

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
  // Destructure params to use primitive values as dependencies
  // This prevents re-fetches when callers pass new object references with identical values
  const { page: paramPage, limit: paramLimit, includeContacts, search, fromDate, toDate } = params;

  const [activities, setActivities] = useState<Activity[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(paramPage || 1);
  const [limit, setLimit] = useState(paramLimit || 25);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const requestRef = useRef(0);

  const fetchActivities = useCallback(async () => {
    const requestId = requestRef.current + 1;
    requestRef.current = requestId;
    setLoading(true);
    setError(null);

    try {
      if (!isAuthenticated()) {
        throw new Error('No authentication token found');
      }

      if (contactId) {
        const data = await getContactActivities(contactId);
        if (requestRef.current !== requestId) {
          return;
        }
        setActivities(data.activities || []);
        setTotal(data.activities?.length || 0);
        setPage(1);
        setLimit(paramLimit || data.activities?.length || 25);
      } else {
        const fetchParams: GetActivitiesParams = { page: paramPage, limit: paramLimit, includeContacts, search, fromDate, toDate };
        const data: ActivitiesResponse = await getActivities(fetchParams);
        if (requestRef.current !== requestId) {
          return;
        }
        setActivities(data.activities || []);
        setTotal(data.total || 0);
        setPage(data.page || 1);
        setLimit(data.limit || paramLimit || 25);
      }
    } catch (err) {
      if (requestRef.current !== requestId) {
        return;
      }
      const message = handleFetchError(err, 'fetching activities');
      setError(message);
    } finally {
      if (requestRef.current === requestId) {
        setLoading(false);
      }
    }
  }, [contactId, paramPage, paramLimit, includeContacts, search, fromDate, toDate]);

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
