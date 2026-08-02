import { useState, useCallback, useEffect, useMemo } from 'react';
import {
  listCircles,
  createCircle,
  updateCircle,
  deleteCircle,
  Circle,
  CircleMember,
} from '../api/circles';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

export function useCircles(notifier?: ErrorNotifier) {
  const [circles, setCircles] = useState<Circle[]>([]);
  const [members, setMembers] = useState<CircleMember[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await listCircles({ limit: 200, include_members: true });
      setCircles(response.circles || []);
      setMembers(response.members || []);
    } catch (err) {
      setError(handleFetchError(err, 'fetching circles'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const circleById = useMemo(() => {
    const m = new Map<string, Circle>();
    for (const c of circles) m.set(c.id, c);
    return m;
  }, [circles]);

  // circleNamesByUid maps a contact VCardUID to the list of Circles it belongs to.
  const circleNamesByUid = useMemo(() => {
    const m = new Map<string, string[]>();
    for (const member of members) {
      const circle = circleById.get(member.circle_id);
      if (!circle) continue;
      const names = m.get(member.member_vcard_uid) || [];
      names.push(circle.name);
      m.set(member.member_vcard_uid, names);
    }
    return m;
  }, [members, circleById]);

  const handleCreate = useCallback(
    async (name: string) => {
      try {
        const resp = await createCircle(name);
        return resp.circle;
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

  return { circles, members, circleNamesByUid, loading, error, refresh, handleCreate, handleUpdate, handleDelete };
}
