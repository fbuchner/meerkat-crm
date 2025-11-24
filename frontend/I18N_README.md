# Internationalization (i18n) Setup

This document describes the i18n implementation in the meerkat CRM frontend application.

## Overview

The application uses **react-i18next** for internationalization support, currently configured for:
- 🇬🇧 English (en) - Default language
- 🇩🇪 German (de)

## Installation

The following packages are installed:
```bash
npm install i18next react-i18next i18next-browser-languagedetector
```

## Configuration

### Main Configuration File
Location: `/src/i18n/config.ts`

The configuration includes:
- Language detector (automatically detects user's browser language)
- Language fallback to English
- Local storage for language persistence
- Translation resources for each language

### Translation Files
Location: `/src/i18n/locales/`

- `en.json` - English translations
- `de.json` - German translations

## Usage in Components

### Import the Hook
```tsx
import { useTranslation } from 'react-i18next';
```

### Use in Component
```tsx
function MyComponent() {
  const { t, i18n } = useTranslation();
  
  return (
    <div>
      <h1>{t('app.title')}</h1>
      <p>{t('dashboard.welcome')}</p>
    </div>
  );
}
```

### Change Language Programmatically
```tsx
i18n.changeLanguage('de'); // Switch to German
i18n.changeLanguage('en'); // Switch to English
```

### Interpolation (Dynamic Values)
```tsx
// In translation file:
{
  "contacts.filteredMessage": "{{count}} out of {{total}} contacts in '{{circle}}'"
}

// In component:
t('contacts.filteredMessage', { count: 5, total: 10, circle: 'Family' })
// Output: "5 out of 10 contacts in 'Family'"
```

## Translation Structure

The translation keys are organized by feature/page:

```json
{
  "app": {
    "title": "App-wide strings",
    "logout": "Common UI elements"
  },
  "nav": {
    "dashboard": "Navigation items",
    "contacts": "..."
  },
  "login": {
    "title": "Login page strings",
    "email": "..."
  },
  "contacts": {
    "search": "Contacts page strings",
    "filterByCircle": "..."
  }
}
```

## Language Selector

The language selector is implemented in the app header (`App.tsx`):
- Displays current language
- Allows switching between EN and DE
- Persists selection in localStorage
- Updates all components automatically

## Adding a New Language

1. Create a new translation file: `/src/i18n/locales/{language_code}.json`
2. Copy the structure from `en.json` and translate all strings
3. Import the translations in `/src/i18n/config.ts`:
   ```tsx
   import frTranslations from './locales/fr.json';
   ```
4. Add to the resources object:
   ```tsx
   resources: {
     en: { translation: enTranslations },
     de: { translation: deTranslations },
     fr: { translation: frTranslations }
   }
   ```
5. Add the language option to the selector in `App.tsx`:
   ```tsx
   <MenuItem value={'fr'}>FR</MenuItem>
   ```

## Adding New Translation Keys

1. Add the key to all language files (`en.json`, `de.json`, etc.)
2. Use the key in your component with `t('your.new.key')`

Example:
```json
// en.json
{
  "contacts": {
    "addContact": "Add Contact"
  }
}

// de.json
{
  "contacts": {
    "addContact": "Kontakt hinzufügen"
  }
}
```

```tsx
// In component
<Button>{t('contacts.addContact')}</Button>
```

## Best Practices

1. **Namespace your keys**: Use dot notation to organize translations by feature (e.g., `contacts.search`, `login.title`)
2. **Keep keys descriptive**: Use clear, descriptive key names
3. **Maintain parity**: Ensure all language files have the same structure
4. **Use interpolation**: For dynamic content, use interpolation instead of string concatenation
5. **Test both languages**: Always test your changes in all supported languages

## Current Implementation Status

✅ App.tsx - Fully translated
✅ LoginPage.tsx - Fully translated
✅ RegisterPage.tsx - Fully translated  
✅ ContactsPage.tsx - Fully translated
⏸️ Notes page - Pending (placeholder text translated)
⏸️ Activities page - Pending (placeholder text translated)
⏸️ Reminders page - Pending (placeholder text translated)

## Language Detection Order

The application detects the user's language in the following order:
1. Previously saved language in localStorage
2. Browser's default language
3. Fallback to English (en)

## Files Modified

- `/src/index.tsx` - Import i18n configuration
- `/src/i18n/config.ts` - i18n configuration
- `/src/i18n/locales/en.json` - English translations
- `/src/i18n/locales/de.json` - German translations
- `/src/App.tsx` - Language selector and navigation translations
- `/src/LoginPage.tsx` - Login form translations
- `/src/RegisterPage.tsx` - Registration form translations
- `/src/ContactsPage.tsx` - Contacts page translations
