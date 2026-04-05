import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import enTranslations from './locales/en.json';
import deTranslations from './locales/de.json';
import itTranslations from './locales/it.json';

// Suppress i18next's promotional console message (hardcoded since v23)
const noop = () => {};
const origLog = console.log;
console.log = noop;
i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: {
        translation: enTranslations
      },
      de: {
        translation: deTranslations
      },
      it: {
        translation: itTranslations
      }
    },
    fallbackLng: 'en',
    load: 'languageOnly',
    debug: false,
    interpolation: {
      escapeValue: false
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage']
    }
  }).then(() => {
    console.log = origLog;
  });

export default i18n;
