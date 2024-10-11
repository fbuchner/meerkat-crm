
import apiClient from '@/services/api';

const API_URL = '/activities';

export default {
  async getActivities(contactId) {
    try {
      const response = await apiClient.get(`/contacts/${contactId}/activities`);
      return response;
    } catch (error) {
      console.error('Error fetching activities:', error);
      throw error;
    }
  },
  async addActivity(activityData) {
    try {
      await apiClient.post(API_URL, activityData);
    } catch (error) {
      console.error('Error adding activity:', error);
      throw error;
    }
  },
  async updateActivity(activityId, activityData) {
    try {
      await apiClient.put(`${API_URL}/${activityId}`, activityData);
    } catch (error) {
      console.error('Error updating activity:', error);
      throw error;
    }
  },
  async deleteActivity(activityId) {
    try {
      await apiClient.delete(`${API_URL}/${activityId}`);
    } catch (error) {
      console.error('Error deleting activity:', error);
      throw error;
    }
  },
};