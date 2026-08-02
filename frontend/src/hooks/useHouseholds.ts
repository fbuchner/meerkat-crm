import { useState, useCallback, useEffect } from 'react';
import {
  listHouseholds,
  createHousehold,
  updateHousehold,
  deleteHousehold,
  addHouseholdMember,
  removeHouseholdMember,
  suggestHouseholdRelationships,
  Household,
  HouseholdMember,
  HouseholdInput,
} from '../api/households';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

export function useHouseholds(notifier?: ErrorNotifier) {
  const [households, setHouseholds] = useState<Household[]>([]);
  const [members, setMembers] = useState<HouseholdMember[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await listHouseholds({ limit: 200, include_members: true });
      setHouseholds(response.households || []);
      setMembers(response.members || []);
    } catch (err) {
      setError(handleFetchError(err, 'fetching households'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleCreate = useCallback(
    async (input: HouseholdInput) => {
      try {
        const created = await createHousehold(input);
        await refresh();
        return created;
      } catch (err) {
        handleError(err, { operation: 'creating household' }, notifier);
        throw err;
      }
    },
    [refresh, notifier]
  );

  const handleUpdate = useCallback(
    async (id: string, input: HouseholdInput) => {
      try {
        await updateHousehold(id, input);
        await refresh();
      } catch (err) {
        handleError(err, { operation: 'updating household' }, notifier);
        throw err;
      }
    },
    [refresh, notifier]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteHousehold(id);
        await refresh();
      } catch (err) {
        handleError(err, { operation: 'deleting household' }, notifier);
        throw err;
      }
    },
    [refresh, notifier]
  );

  const handleAddMember = useCallback(
    async (householdId: string, memberVCardUid: string, role?: string) => {
      try {
        await addHouseholdMember(householdId, { member_vcard_uid: memberVCardUid, role: role || undefined });
        await refresh();
      } catch (err) {
        handleError(err, { operation: 'adding household member' }, notifier);
        throw err;
      }
    },
    [refresh, notifier]
  );

  const handleRemoveMember = useCallback(
    async (householdId: string, memberVCardUid: string) => {
      try {
        await removeHouseholdMember(householdId, memberVCardUid);
        await refresh();
      } catch (err) {
        handleError(err, { operation: 'removing household member' }, notifier);
        throw err;
      }
    },
    [refresh, notifier]
  );

  // Runs the suggestion trigger. Returns the number of newly-created
  // suggested edges so the caller can surface "N new suggestions" to the
  // user (re-runs typically return 0).
  const handleSuggestRelationships = useCallback(
    async (householdId: string): Promise<number> => {
      try {
        const response = await suggestHouseholdRelationships(householdId);
        return response.total;
      } catch (err) {
        handleError(err, { operation: 'generating relationship suggestions' }, notifier);
        throw err;
      }
    },
    [notifier]
  );

  return {
    households,
    members,
    loading,
    error,
    refresh,
    handleCreate,
    handleUpdate,
    handleDelete,
    handleAddMember,
    handleRemoveMember,
    handleSuggestRelationships,
  };
}
