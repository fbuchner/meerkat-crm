import apiClient from '@/services/api';

const API_URL = '/contacts';

export default {
  async getContacts({ fields = null, includes = null, page = 1, limit = 25 } = {}) {
    try {
      // Build query parameters
      const params = {
        page,
        limit,
      };
  
      // Add fields if specified
      if (fields) {
        params.fields = fields.join(','); // Convert array of fields to a comma-separated string
      }

      // Add fields if specified
      if (includes) {
        params.includes = fields.join(','); // Convert array of fields to a comma-separated string
      }
  
      // Make the API request with query parameters
      const response = await apiClient.get(API_URL, { params });
      return response; 
    } catch (error) {
      console.error('Error fetching contacts:', error);
      throw error;
    }
  },
  async getCircles() {
    try {
      const response = await apiClient.get(`${API_URL}/circles`);
      return response;
    } catch (error) {
      console.error('Error fetching circles:', error);
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
  async getRelationships(contactId) {
    try {
      const response = await apiClient.get(`${API_URL}/${contactId}/relationships`);
      return response;
    } catch (error) {
      console.error('Error fetching relationships:', error);
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
  async deleteRelationship(contactId, relationshipId) {
    try {
      await apiClient.delete(`${API_URL}/${contactId}/relationships/${relationshipId}`);
    } catch (error) {
      console.error('Error deleting relationship:', error);
      throw error;
    }
  },
  async addPhotoToContact(contactId, photoFile) {
    try {
      // Prepare FormData
      const formData = new FormData();
      formData.append('photo', photoFile);

      // Send the POST request to upload the photo
      const response = await apiClient.post(
        `${API_URL}/${contactId}/profile_picture`,
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        }
      );
      return response.data;
    } catch (error) {
      console.error('Error uploading profile picture:', error);
      throw error;
    }
  },
  async addRelationship(contactId, relationshipData) {
    try {
      const response = await apiClient.post(`${API_URL}/${contactId}/relationships`, relationshipData);
      return response;
    } catch (error) {
      console.error('Error adding relationship:', error);
      throw error;
    }
  },

};

