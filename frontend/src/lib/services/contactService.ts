import api from './api';

const API_URL = '/contacts';

export interface ContactParams {
  fields?: string[];
  includes?: string[];
  search?: string;
  circle?: string;
  page?: number;
  limit?: number;
}

export interface Contact {
  ID: number;
  firstname: string;
  lastname: string;
  nickname?: string;
  gender?: string;
  email?: string;
  phone?: string;
  birthday?: string;
  photo?: string;
  photo_thumbnail?: string;
  address?: string;
  how_we_met?: string;
  food_preference?: string;
  work_information?: string;
  contact_information?: string;
  circles?: string[];
  relationships?: any[];
  activities?: any[];
  notes?: any[];
  reminders?: any[];
  CreatedAt?: string;
  UpdatedAt?: string;
}

export interface ContactsResponse {
  contacts: Contact[];
  total: number;
  page: number;
  limit: number;
}

export const contactService = {
  async getContacts({
    fields = [],
    includes = [],
    search = '',
    circle = '',
    page = 1,
    limit = 25,
  }: ContactParams = {}): Promise<ContactsResponse> {
    try {
      // Build query parameters
      const params: Record<string, string> = {
        page: page.toString(),
        limit: limit.toString(),
      };

      // Add fields if specified
      if (fields && fields.length > 0) {
        params.fields = fields.join(',');
      }

      // Add includes if specified
      if (includes && includes.length > 0) {
        params.includes = includes.join(',');
      }

      // Add search term if specified
      if (search) {
        params.search = search.trim();
      }

      // Add circle filter if specified
      if (circle) {
        params.circle = circle.trim();
      }

      // Create the query string
      const queryString = new URLSearchParams(params).toString();

      // Make the API request with query parameters
      const response = await api.get(`${API_URL}?${queryString}`);
      return response;
    } catch (error) {
      console.error('Error fetching contacts:', error);
      throw error;
    }
  },

  async getCircles() {
    try {
      const response = await api.get(`${API_URL}/circles`);
      return response;
    } catch (error) {
      console.error('Error fetching circles:', error);
      throw error;
    }
  },
  
  async getContact(contactId: number | string) {
    try {
      const response = await api.get(`${API_URL}/${contactId}`);
      return response;
    } catch (error) {
      console.error('Error fetching contact:', error);
      throw error;
    }
  },
  
  async addContact(contactData: Partial<Contact>) {
    try {
      const response = await api.post(API_URL, contactData);
      return response;
    } catch (error) {
      console.error('Error creating contact:', error);
      throw error;
    }
  },
  
  async updateContact(contactId: number | string, contactData: Partial<Contact>) {
    try {
      return await api.put(`${API_URL}/${contactId}`, contactData);
    } catch (error) {
      console.error('Error updating contact:', error);
      throw error;
    }
  },
  
  async deleteContact(contactId: number | string) {
    try {
      return await api.delete(`${API_URL}/${contactId}`);
    } catch (error) {
      console.error('Error deleting contact:', error);
      throw error;
    }
  },
  
  async addPhotoToContact(contactId: number | string, photoFile: File) {
    try {
      // Prepare FormData
      const formData = new FormData();
      formData.append('photo', photoFile);

      // Send the POST request to upload the photo
      const response = await fetch(`${API_URL}/${contactId}/profile_picture`, {
        method: 'POST',
        body: formData,
      });
      
      if (!response.ok) {
        throw new Error('Failed to upload photo');
      }
      
      return await response.json();
    } catch (error) {
      console.error('Error uploading profile picture:', error);
      throw error;
    }
  },
};
