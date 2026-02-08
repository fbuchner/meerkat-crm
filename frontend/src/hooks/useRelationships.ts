import { useState, useCallback } from 'react';
import {
  getRelationships,
  getIncomingRelationships,
  createRelationship,
  updateRelationship,
  deleteRelationship,
  Relationship,
  IncomingRelationship,
  RelationshipFormData
} from '../api/relationships';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

export function useRelationships(
  contactId: string | undefined,
  notifier?: ErrorNotifier
) {
  const [relationships, setRelationships] = useState<Relationship[]>([]);
  const [incomingRelationships, setIncomingRelationships] = useState<IncomingRelationship[]>([]);
  const [relationshipDialogOpen, setRelationshipDialogOpen] = useState(false);
  const [editingRelationship, setEditingRelationship] = useState<Relationship | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshRelationships = useCallback(async () => {
    if (!contactId) return;
    setLoading(true);
    setError(null);
    try {
      const [relationshipsResponse, incomingResponse] = await Promise.all([
        getRelationships(contactId),
        getIncomingRelationships(contactId)
      ]);
      setRelationships(relationshipsResponse.relationships || []);
      setIncomingRelationships(incomingResponse.incoming_relationships || []);
    } catch (err) {
      const message = handleFetchError(err, 'fetching relationships');
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [contactId]);

  const handleSaveRelationship = async (data: RelationshipFormData) => {
    if (!contactId) return;

    try {
      if (editingRelationship) {
        await updateRelationship(contactId, editingRelationship.ID, data);
      } else {
        await createRelationship(contactId, data);
      }
      await refreshRelationships();
      setRelationshipDialogOpen(false);
      setEditingRelationship(null);
    } catch (err) {
      handleError(err, { operation: 'saving relationship' }, notifier);
      throw err;
    }
  };

  const handleEditRelationship = (relationship: Relationship) => {
    setEditingRelationship(relationship);
    setRelationshipDialogOpen(true);
  };

  const handleDeleteRelationship = async (relationshipId: number) => {
    if (!contactId) return;

    try {
      await deleteRelationship(contactId, relationshipId);
      await refreshRelationships();
    } catch (err) {
      handleError(err, { operation: 'deleting relationship' }, notifier);
      throw err;
    }
  };

  const handleAddRelationship = () => {
    setEditingRelationship(null);
    setRelationshipDialogOpen(true);
  };

  return {
    relationships,
    incomingRelationships,
    relationshipDialogOpen,
    editingRelationship,
    loading,
    error,
    refreshRelationships,
    handleSaveRelationship,
    handleEditRelationship,
    handleDeleteRelationship,
    handleAddRelationship,
    setRelationshipDialogOpen,
    setEditingRelationship,
  };
}
