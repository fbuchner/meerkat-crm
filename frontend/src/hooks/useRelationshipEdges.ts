import { useState, useCallback, useMemo } from 'react';
import {
  getRelationshipEdges,
  createRelationshipEdge,
  updateRelationshipEdge,
  deleteRelationshipEdge,
  acceptRelationshipEdge,
  rejectRelationshipEdge,
  getOtherPartyId,
  RelationshipEdge,
  RelationshipEdgeInput,
} from '../api/relationshipEdges';
import { getContactsByUid, Contact } from '../api/contacts';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

export function useRelationshipEdges(
  viewedContactUid: string | undefined,
  notifier?: ErrorNotifier
) {
  const [edges, setEdges] = useState<RelationshipEdge[]>([]);
  const [contactsByUid, setContactsByUid] = useState<Map<string, Contact>>(new Map());
  const [relationshipDialogOpen, setRelationshipDialogOpen] = useState(false);
  const [editingEdge, setEditingEdge] = useState<RelationshipEdge | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const confirmedEdges = useMemo(() => edges.filter((e) => e.status === 'confirmed'), [edges]);
  const suggestedEdges = useMemo(() => edges.filter((e) => e.status === 'suggested'), [edges]);

  // Accepts an optional override uid so the very first load (before
  // viewedContactUid is available via state -- the caller's own record is
  // still resolving) can pass the freshly-fetched uid directly instead of
  // this callback closing over an undefined value from the render that
  // instantiated it.
  const refreshRelationshipEdges = useCallback(async (overrideUid?: string) => {
    const uid = overrideUid ?? viewedContactUid;
    if (!uid) return;
    setLoading(true);
    setError(null);
    try {
      const response = await getRelationshipEdges({ contactId: uid, limit: 100 });
      const fetched = response.relationship_edges || [];
      setEdges(fetched);
      const otherUids = fetched.map((e) => getOtherPartyId(e, uid));
      setContactsByUid(otherUids.length > 0 ? await getContactsByUid(otherUids) : new Map());
    } catch (err) {
      setError(handleFetchError(err, 'fetching relationship edges'));
    } finally {
      setLoading(false);
    }
  }, [viewedContactUid]);

  const handleSaveRelationshipEdge = async (input: RelationshipEdgeInput) => {
    try {
      if (editingEdge) {
        await updateRelationshipEdge(editingEdge.id, input);
      } else {
        await createRelationshipEdge(input);
      }
      await refreshRelationshipEdges();
      setRelationshipDialogOpen(false);
      setEditingEdge(null);
    } catch (err) {
      handleError(err, { operation: 'saving relationship' }, notifier);
      throw err;
    }
  };

  const handleEditRelationshipEdge = (edge: RelationshipEdge) => {
    setEditingEdge(edge);
    setRelationshipDialogOpen(true);
  };

  const handleDeleteRelationshipEdge = async (edgeId: string) => {
    try {
      await deleteRelationshipEdge(edgeId);
      await refreshRelationshipEdges();
    } catch (err) {
      handleError(err, { operation: 'deleting relationship' }, notifier);
      throw err;
    }
  };

  const handleAcceptSuggestion = async (edgeId: string) => {
    try {
      await acceptRelationshipEdge(edgeId);
      await refreshRelationshipEdges();
    } catch (err) {
      handleError(err, { operation: 'accepting suggested relationship' }, notifier);
      throw err;
    }
  };

  const handleRejectSuggestion = async (edgeId: string) => {
    try {
      await rejectRelationshipEdge(edgeId);
      await refreshRelationshipEdges();
    } catch (err) {
      handleError(err, { operation: 'rejecting suggested relationship' }, notifier);
      throw err;
    }
  };

  const handleAddRelationshipEdge = () => {
    setEditingEdge(null);
    setRelationshipDialogOpen(true);
  };

  return {
    edges,
    confirmedEdges,
    suggestedEdges,
    contactsByUid,
    relationshipDialogOpen,
    editingEdge,
    loading,
    error,
    refreshRelationshipEdges,
    handleSaveRelationshipEdge,
    handleEditRelationshipEdge,
    handleDeleteRelationshipEdge,
    handleAcceptSuggestion,
    handleRejectSuggestion,
    handleAddRelationshipEdge,
    setRelationshipDialogOpen,
    setEditingEdge,
  };
}
