import { createRouter, createWebHistory } from "vue-router";
import AddContact from "@/components/ContactAdd.vue";
import ContactList from "@/components/ContactList.vue";
import ContactView from "@/components/ContactView.vue";
import NotesList from "@/components/NotesList.vue";
import ActivitiesList from "@/components/ActivitiesList.vue";
import UserRegister from "@/components/UserRegister.vue";
import UserLogin from "@/components/UserLogin.vue";
import MainView from "@/components/MainView.vue";

const routes = [
  //{ path: '/', name: 'DashboardHome', component: () => import('@/views/DashboardHome.vue'), },
  { path: "/", component: MainView },
  { path: "/register", component: UserRegister },
  { path: "/login", component: UserLogin },
  { path: "/add-contact", component: AddContact },
  { path: "/contacts", component: ContactList },
  {
    path: "/contacts/:ID",
    name: "ContactView",
    component: ContactView,
    props: true,
  }, // Route for viewing a contact
  { path: "/notes", component: NotesList },
  { path: "/activities", component: ActivitiesList },
  { path: "/:catchAll(.*)", redirect: "/", name: "NotFound" },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;
