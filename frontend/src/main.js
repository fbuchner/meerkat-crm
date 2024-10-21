// src/main.js
import { createApp } from 'vue';
import App from './App.vue';
import router from './router'; // Import the router
import 'vuetify/styles' // Global Vuetify styles
import vuetify from './plugins/vuetify' // Import Vuetify


const app = createApp(App);

app.use(router); // Use the router in your app
app.use(vuetify) // Add Vuetify as a plugin
app.mount('#app');

