// src/main.js
import { createApp } from 'vue';
import { createI18n } from 'vue-i18n';
import { loadLocaleMessages } from './locales/index';
import App from './App.vue';
import router from './router';
import 'vuetify/styles';
import vuetify from './plugins/vuetify';

export const i18n = createI18n({
    legacy: false, 
    locale: 'en',  // Default language
    fallbackLocale: 'en', 
    messages: {}
  });

const app = createApp(App);

(async () => {
  const savedLanguage = localStorage.getItem('preferredLanguage') || 'en';
  await loadLocaleMessages(i18n, savedLanguage); 
  app.use(i18n);
  app.use(router);
  app.use(vuetify);
  app.mount('#app'); 
})();




