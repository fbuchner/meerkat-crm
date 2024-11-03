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
  async getUnassignedNotes() {
    try {
      const response = await apiClient.get(`${API_URL}`);
      return response;
    } catch (error) {
      console.error('Error fetching notes:', error);
      throw error;
    }
  },
  async addNote(contactId, noteData) {
    try {
      // Return the response so it can be used by the calling function
      const response = await apiClient.post(`/contacts/${contactId}/notes`, noteData);
      return response; // Ensure the response is returned
    } catch (error) {
      console.error('Error adding note:', error);
      throw error; // Re-throw the error to handle it in the calling function
    }
  },
  async addUnassignedNote(noteData) {
    try {
      // Return the response so it can be used by the calling function
      const response = await apiClient.post(`${API_URL}`, noteData);
      return response; // Ensure the response is returned
    } catch (error) {
      console.error('Error adding note:', error);
      throw error; // Re-throw the error to handle it in the calling function
    }
  },
  async updateNote(noteId, noteData) {
    try {
      const response = await apiClient.put(`${API_URL}/${noteId}`, noteData);
      return response; // Return the response from the update operation
    } catch (error) {
      console.error('Error updating note:', error);
      throw error; // Re-throw the error for the calling function to handle
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