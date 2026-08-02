import { useState, useCallback, useEffect, useMemo } from 'react';
import {
  listTags,
  createTag,
  updateTag,
  deleteTag,
  Tag,
  ContactTag,
} from '../api/tags';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

export function useTags(notifier?: ErrorNotifier) {
  const [tags, setTags] = useState<Tag[]>([]);
  const [contacts, setContacts] = useState<ContactTag[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await listTags({ limit: 200, include_contacts: true });
      setTags(response.tags || []);
      setContacts(response.contacts || []);
    } catch (err) {
      setError(handleFetchError(err, 'fetching tags'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const tagById = useMemo(() => {
    const m = new Map<string, Tag>();
    for (const t of tags) m.set(t.id, t);
    return m;
  }, [tags]);

  // tagNamesByUid maps a contact VCardUID to the list of Tag names it has.
  const tagNamesByUid = useMemo(() => {
    const m = new Map<string, string[]>();
    for (const ct of contacts) {
      const tag = tagById.get(ct.tag_id);
      if (!tag) continue;
      const names = m.get(ct.contact_vcard_uid) || [];
      names.push(tag.name);
      m.set(ct.contact_vcard_uid, names);
    }
    return m;
  }, [contacts, tagById]);

  const handleCreate = useCallback(
    async (name: string) => {
      try {
        const resp = await createTag(name);
        return resp.tag;
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

  return { tags, contacts, tagNamesByUid, loading, error, refresh, handleCreate, handleUpdate, handleDelete };
}
