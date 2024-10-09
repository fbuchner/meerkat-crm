import { createRouter, createWebHistory } from 'vue-router';
import AddContact from '@/components/AddContact.vue'; // Import AddContact component
import ContactList from '@/components/ContactList.vue'; // Example: if you have a contact list component

const routes = [
  { path: '/add-contact', component: AddContact }, // Route for adding a contact
  { path: '/contacts', component: ContactList },   // Example: Route for listing contacts
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;
