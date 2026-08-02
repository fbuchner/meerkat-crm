import { useState, useCallback, useEffect } from 'react';
import {
  listTags,
  createTag,
  updateTag,
  deleteTag,
  Tag,
} from '../api/tags';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

export function useTags(notifier?: ErrorNotifier) {
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await listTags({ limit: 100 });
      setTags(response.tags || []);
    } catch (err) {
      setError(handleFetchError(err, 'fetching tags'));
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
        await createTag(name);
      } catch (err) {
        handleError(err, { operation: 'creating tag' }, notifier);
        throw err;
      }
    },
    [notifier]
  );

  const handleUpdate = useCallback(
    async (id: string, name: string) => {
      try {
        await updateTag(id, name);
      } catch (err) {
        handleError(err, { operation: 'updating tag' }, notifier);
        throw err;
      }
    },
    [notifier]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteTag(id);
        await refresh();
      } catch (err) {
        handleError(err, { operation: 'deleting tag' }, notifier);
        throw err;
      }
    },
    [refresh, notifier]
  );

  return { tags, loading, error, refresh, handleCreate, handleUpdate, handleDelete };
}
