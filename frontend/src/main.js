// src/main.js
import { createApp } from 'vue';
import App from './App.vue';
import router from './router'; // Import the router
import './assets/styles.css';  // Import global CSS

const app = createApp(App);

app.use(router); // Use the router in your app
app.mount('#app');

