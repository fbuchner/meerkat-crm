import apiClient from '@/services/api';

const API_URL = '/contacts';

export default {
  async getContacts() {
    try {
      const response = await apiClient.get(API_URL);
      return response;
    } catch (error) {
      console.error('Error fetching contacts:', error);
      throw error;
    }
  },
  async getContact(contactId) {
    try {
      const response = await apiClient.get(`${API_URL}/${contactId}`);
      return response;
    } catch (error) {
      console.error('Error fetching contact:', error);
      throw error;
    }
  },
  async addContact(contactData) {
    try {
      const response = await apiClient.post(API_URL, contactData);
      return response;
    } catch (error) {
      console.error('Error creating contact:', error);
      throw error;
    }
  },
  async updateContact(contactId, contactData) {
    try {
      await apiClient.put(`${API_URL}/${contactId}`, contactData);
    } catch (error) {
      console.error('Error updating contact:', error);
      throw error;
    }
  },
  async deleteContact(contactId) {
    try {
      await apiClient.delete(`${API_URL}/${contactId}`);
    } catch (error) {
      console.error('Error deleting contact:', error);
      throw error;
    }
  },
};

