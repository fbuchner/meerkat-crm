# Frontend Setup TODO List


Project: Meerkat CRM
Frontend: React + Material UI, React Router, JWT Auth
Backend: Go, REST API, JWT Auth, supports pagination for contacts

Recent Frontend Progress:

App uses JWT for all API calls except login/register; token is passed via Authorization header.
Contacts page displays a compact card: photo (fetched securely as blob), firstname, nickname (in quotes), lastname, and circles.
Circle tags are clickable; clicking filters the contact list by that circle.
Filter message shows “n out of m contacts in ‘circle’” with a reset button to clear the filter.
Circle dropdown is populated from /contacts/circles (array response).
Contacts API supports pagination; frontend requests contacts with page and size params, displays pagination controls, and updates counts accordingly.
All API calls use the JWT from login.
Code is modular, uses React hooks, and Material UI components.
Next Steps:

Continue building out Notes, Activities, and Reminders pages.
Add i18n support (suggested: react-i18next).
Configure environment variables for frontend.
Add frontend tests (Jest, React Testing Library).
Update documentation in README.md.
Open Issues/Considerations:

Ensure backend endpoints for notes, activities, reminders are ready and documented.
Confirm all error handling and edge cases for API calls.
Optimize image fetching and caching if needed.
Consider accessibility and mobile responsiveness for all pages.
How to Continue:

Pick up from ContactsPage.tsx for further UI/UX improvements or new features.
Use the established pattern for secure API calls and pagination.
Reference the TODO list for pending tasks.

TODO List:
- [x] **Build Contacts page**
  - Develop the Contacts page: list, search, add, and view contact details. Integrate Material UI components for forms and lists.
- [ ] **Build Notes & Activities pages**
  - Develop the Notes and Activities pages: timeline, note creation, and activity assignment. Use Material UI cards, lists, and forms.
- [ ] **Build Reminders page**
  - Develop the Reminders page: list reminders, create new reminders, and configure notifications. Use Material UI components for forms and lists.
- [x] **Set up i18n support**
  - ✅ Implemented internationalization (i18n) support with react-i18next
  - ✅ Configured English (EN) and German (DE) languages
  - ✅ Language selector in app header with persistent storage
  - ✅ All existing pages translated (Login, Register, Contacts, App navigation)
  - See I18N_README.md for detailed documentation
- [ ] **Configure environment variables**
  - Set up environment variables and configuration for development and production builds.
- [ ] **Add frontend tests**
  - Add unit and integration tests for key components and pages using Jest and React Testing Library.
- [ ] **Document frontend setup**
  - Update documentation in README.md to include frontend setup and usage instructions.

