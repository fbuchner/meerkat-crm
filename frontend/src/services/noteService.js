import apiClient from '@/services/api';

const API_URL = '/notes';

export default {
  async getNotes(contactId) {
    try {
      const response = await apiClient.get(`/contacts/${contactId}/notes`);
      return response;
    } catch (error) {
      console.error('Error fetching notes:', error);
      throw error;
    }
  },
  async addNote(contactId, noteData) {
    console.log('noteData:', noteData);
    try {
      await apiClient.post(`/contacts/${contactId}/notes`, noteData);
    } catch (error) {
      console.error('Error adding note:', error);
      throw error;
    }
  },
  async updateNote(noteId, content) {
    try {
      await apiClient.put(`${API_URL}/${noteId}`, content );
    } catch (error) {
      console.error('Error updating note:', error);
      throw error;
    }
  },
  async deleteNote(noteId) {
    try {
      await apiClient.delete(`${API_URL}/${noteId}`);
    } catch (error) {
      console.error('Error deleting note:', error);
      throw error;
    }
  },
};