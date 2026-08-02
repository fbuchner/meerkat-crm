import { useState, useCallback, useEffect } from 'react';
import {
  listCircles,
  createCircle,
  updateCircle,
  deleteCircle,
  Circle,
} from '../api/circles';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

export function useCircles(notifier?: ErrorNotifier) {
  const [circles, setCircles] = useState<Circle[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await listCircles({ limit: 100 });
      setCircles(response.circles || []);
    } catch (err) {
      setError(handleFetchError(err, 'fetching circles'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleCreate = useCallback(
    async (name: string) => {
      try {
        await createCircle(name);
      } catch (err) {
        handleError(err, { operation: 'creating circle' }, notifier);
        throw err;
      }
    },
    [notifier]
  );

  const handleUpdate = useCallback(
    async (id: string, name: string) => {
      try {
        await updateCircle(id, name);
      } catch (err) {
        handleError(err, { operation: 'updating circle' }, notifier);
        throw err;
      }
    },
    [notifier]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteCircle(id);
        await refresh();
      } catch (err) {
        handleError(err, { operation: 'deleting circle' }, notifier);
        throw err;
      }
    },
    [refresh, notifier]
  );

  return { circles, loading, error, refresh, handleCreate, handleUpdate, handleDelete };
}
