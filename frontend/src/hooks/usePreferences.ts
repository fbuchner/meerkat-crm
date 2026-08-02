import { useState, useCallback, useEffect } from 'react';
import {
  getPreferences,
  createPreference,
  updatePreference,
  deletePreference,
  Preference,
  PreferenceInput,
} from '../api/preferences';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

// usePreferences loads and manages the structured Preferences (§91.9) for one
// contact, keyed by the contact's VCardUID (entity_id).
export function usePreferences(entityId: string | undefined, notifier?: ErrorNotifier) {
  const [preferences, setPreferences] = useState<Preference[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!entityId) return;
    setLoading(true);
    setError(null);
    try {
      const response = await getPreferences({ entityId, limit: 200 });
      setPreferences(response.preferences || []);
    } catch (err) {
      setError(handleFetchError(err, 'fetching preferences'));
    } finally {
      setLoading(false);
    }
  }, [entityId]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleSave = useCallback(
    async (preference: Preference | null, input: PreferenceInput) => {
      try {
        if (preference) {
          await updatePreference(preference.id, input);
        } else {
          await createPreference(input);
        }
        await refresh();
      } catch (err) {
        handleError(err, { operation: 'saving preference' }, notifier);
        throw err;
      }
    },
    [refresh, notifier]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deletePreference(id);
        await refresh();
      } catch (err) {
        handleError(err, { operation: 'deleting preference' }, notifier);
        throw err;
      }
    },
    [refresh, notifier]
  );

  return { preferences, loading, error, refresh, handleSave, handleDelete };
}
