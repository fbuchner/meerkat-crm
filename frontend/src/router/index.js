import { createRouter, createWebHistory } from 'vue-router';
import AddContact from '@/components/ContactAdd.vue'; // Import AddContact component
import ContactList from '@/components/ContactList.vue'; // Import ContactList component
import ContactView from '@/components/ContactView.vue'; // Import ContactView component

const routes = [
  { path: '/add-contact', component: AddContact },
  { path: '/contacts', component: ContactList },
  { path: '/contacts/:ID', name: 'ContactView', component: ContactView, props: true }, // Route for viewing a contact
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;