import { useState } from 'react';
import {
  getRelationships,
  createRelationship,
  updateRelationship,
  deleteRelationship,
  Relationship,
  RelationshipFormData
} from '../api/relationships';
import { handleFetchError, handleError, ErrorNotifier } from '../utils/errorHandler';

export function useRelationships(
  contactId: string | undefined,
  token: string,
  notifier?: ErrorNotifier
) {
  const [relationships, setRelationships] = useState<Relationship[]>([]);
  const [relationshipDialogOpen, setRelationshipDialogOpen] = useState(false);
  const [editingRelationship, setEditingRelationship] = useState<Relationship | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshRelationships = async () => {
    if (!contactId) return;
    setLoading(true);
    setError(null);
    try {
      const response = await getRelationships(contactId, token);
      setRelationships(response.relationships || []);
    } catch (err) {
      const message = handleFetchError(err, 'fetching relationships');
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveRelationship = async (data: RelationshipFormData) => {
    if (!contactId) return;

    try {
      if (editingRelationship) {
        await updateRelationship(contactId, editingRelationship.ID, data, token);
      } else {
        await createRelationship(contactId, data, token);
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
      await deleteRelationship(contactId, relationshipId, token);
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
