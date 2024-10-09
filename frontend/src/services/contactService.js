// src/services/contactsService.js
import apiClient from './api';

export default {
  getContacts() {
    return apiClient.get('/contacts');
  },
  getContact(id) {
    return apiClient.get(`/contacts/${id}`);
  },
  createContact(contact) {
    return apiClient.post('/contacts', contact);
  },
  updateContact(id, contact) {
    return apiClient.put(`/contacts/${id}`, contact);
  },
  deleteContact(id) {
    return apiClient.delete(`/contacts/${id}`);
  },
};
