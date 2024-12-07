import en from './en.json';
import de from './de.json';

export default {
  en,
  de,
};

// Dynamic loading of translation files
export async function loadLocaleMessages(i18n, locale) {
  try {
    const messages = await import(`./${locale}.json`);
    i18n.global.setLocaleMessage(locale, messages.default);
    i18n.global.locale = locale;
  } catch (error) {
    console.error(`Error loading locale messages for ${locale}:`, error);
    i18n.global.locale = i18n.global.fallbackLocale;
  }
}