import { useState, useCallback, useEffect } from 'react';
import {
  getLifeEvents,
  createLifeEvent,
  updateLifeEvent,
  deleteLifeEvent,
  LifeEvent,
  PartialDate,
  GetLifeEventsParams,
} from '../api/lifeEvents';
import { getContactsByUid, Contact } from '../api/contacts';
import { handleFetchError } from '../utils/errorHandler';

export function useLifeEvents(entityId: string | undefined) {
  const [events, setEvents] = useState<LifeEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [contactsByUid, setContactsByUid] = useState<Map<string, Contact>>(new Map());

  const refresh = useCallback(
    async (overrideEntityId?: string) => {
      const uid = overrideEntityId ?? entityId;
      if (!uid) return;
      setLoading(true);
      setError(null);
      try {
        const params: GetLifeEventsParams = { entity_id: uid, limit: 50 };
        const response = await getLifeEvents(params);
        const fetched = response.life_events || [];
        setEvents(fetched);
        setTotal(response.total ?? fetched.length);

        const relatedUids: string[] = [];
        for (const e of fetched) {
          if (e.related_entity_ids) {
            for (const rid of e.related_entity_ids) {
              if (rid !== uid && !relatedUids.includes(rid)) {
                relatedUids.push(rid);
              }
            }
          }
        }
        setContactsByUid(
          relatedUids.length > 0 ? await getContactsByUid(relatedUids) : new Map()
        );
      } catch (err) {
        setError(handleFetchError(err, 'fetching life events'));
      } finally {
        setLoading(false);
      }
    },
    [entityId]
  );

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleCreate = useCallback(
    async (data: {
      entity_id: string;
      type: string;
      date?: PartialDate;
      description?: string;
      source?: string;
      related_entity_ids?: string[];
      remind?: boolean;
    }) => {
      await createLifeEvent(data);
      await refresh();
    },
    [refresh]
  );

  const handleUpdate = useCallback(
    async (
      id: string,
      data: {
        entity_id: string;
        type: string;
        date?: PartialDate;
        description?: string;
        source?: string;
        related_entity_ids?: string[];
        remind?: boolean;
      }
    ) => {
      await updateLifeEvent(id, data);
      await refresh();
    },
    [refresh]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      await deleteLifeEvent(id);
      await refresh();
    },
    [refresh]
  );

  return {
    events,
    total,
    contactsByUid,
    loading,
    error,
    refresh,
    handleCreate,
    handleUpdate,
    handleDelete,
  };
}
