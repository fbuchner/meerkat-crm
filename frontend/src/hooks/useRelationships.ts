import { useState } from 'react';
import {
  getRelationships,
  createRelationship,
  updateRelationship,
  deleteRelationship,
  Relationship,
  RelationshipFormData
} from '../api/relationships';

export function useRelationships(contactId: string | undefined, token: string) {
  const [relationships, setRelationships] = useState<Relationship[]>([]);
  const [relationshipDialogOpen, setRelationshipDialogOpen] = useState(false);
  const [editingRelationship, setEditingRelationship] = useState<Relationship | null>(null);
  const [loading, setLoading] = useState(false);

  const refreshRelationships = async () => {
    if (!contactId) return;
    setLoading(true);
    try {
      const response = await getRelationships(contactId, token);
      setRelationships(response.relationships || []);
    } catch (err) {
      console.error('Error fetching relationships:', err);
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
      console.error('Error saving relationship:', err);
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
      console.error('Error deleting relationship:', err);
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
    refreshRelationships,
    handleSaveRelationship,
    handleEditRelationship,
    handleDeleteRelationship,
    handleAddRelationship,
    setRelationshipDialogOpen,
    setEditingRelationship,
  };
}
