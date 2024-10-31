// src/main.js
import { createApp } from 'vue';
import App from './App.vue';
import router from './router'; // Import the router
import 'vuetify/styles' // Ensure you are using CSS styles
import vuetify from './plugins/vuetify' // Import Vuetify


createApp(App)
    .use(router) // Use the router in your app
    .use(vuetify) // Add Vuetify as a plugin
    .mount('#app');

