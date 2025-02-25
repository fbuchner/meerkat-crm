import apiClient from '@/services/api';

const API_URL = '/reminders';

export default {
  async getReminders(contactId) {
    try {
      const response = await apiClient.get(`/contacts/${contactId}/reminders`);
      return response;
    } catch (error) {
      console.error('Error fetching reminders:', error);
      throw error;
    }
  },
  async addReminder(contactId, reminderData) {
    try {
      // Return the response so it can be used by the calling function
      const response = await apiClient.post(`/contacts/${contactId}/reminders`, reminderData);
      return response; // Ensure the response is returned
    } catch (error) {
      console.error('Error adding reminder:', error);
      throw error; // Re-throw the error to handle it in the calling function
    }
  },
  async updateReminder(reminderId, reminderData) {
    try {
      const response = await apiClient.put(`${API_URL}/${reminderId}`, reminderData);
      return response; // Return the response from the update operation
    } catch (error) {
      console.error('Error updating reminder:', error);
      throw error; // Re-throw the error for the calling function to handle
    }
  },
  async deleteReminder(reminderId) {
    try {
      await apiClient.delete(`${API_URL}/${reminderId}`);
    } catch (error) {
      console.error('Error deleting reminder:', error);
      throw error;
    }
  },
};