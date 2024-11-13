import { createRouter, createWebHistory } from 'vue-router';
import AddContact from '@/components/ContactAdd.vue'; // Import AddContact component
import ContactList from '@/components/ContactList.vue'; // Import ContactList component
import ContactView from '@/components/ContactView.vue'; // Import ContactView component
import NotesList from '@/components/NotesList.vue'; // Import NotesList component
import ActivitiesList from '@/components/ActivitiesList.vue'; // Import ActivitiesList component

const routes = [
  //{ path: '/', name: 'DashboardHome', component: () => import('@/views/DashboardHome.vue'), },
  { path: '/', redirect: '/contacts' },
  { path: '/add-contact', component: AddContact },
  { path: '/contacts', component: ContactList },
  { path: '/contacts/:ID', name: 'ContactView', component: ContactView, props: true }, // Route for viewing a contact
  { path: '/notes', component: NotesList },
  { path: '/activities', component: ActivitiesList },
  { path: '/:catchAll(.*)', redirect: '/', name: 'NotFound'},
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;