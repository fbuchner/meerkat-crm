import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import App from './App';
import ErrorBoundary from './components/ErrorBoundary';
import * as serviceWorkerRegistration from './serviceWorkerRegistration';
import reportWebVitals from './reportWebVitals';
import './i18n/config';
import { AppThemeProvider } from './AppThemeProvider';
import { DateFormatProvider } from './DateFormatProvider';
import { SnackbarProvider } from './context/SnackbarContext';

const logError = (error: Error, errorInfo: React.ErrorInfo) => {
  console.error('Application Error:', error);
  console.error('Error Info:', errorInfo);
  
};

// A tab left open across a release can request page files that no longer exist.
// Reload to pick up the current ones, but only once so a real outage can't loop.
window.addEventListener('vite:preloadError', () => {
  const lastReload = Number(sessionStorage.getItem('staleChunkReloadAt'));
  if (Date.now() - lastReload > 10000) {
    sessionStorage.setItem('staleChunkReloadAt', String(Date.now()));
    window.location.reload();
  }
});

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

root.render(
  <React.StrictMode>
    <AppThemeProvider>
      <DateFormatProvider>
        <SnackbarProvider>
          <ErrorBoundary
            name="Application"
            onError={logError}
            showDetails={import.meta.env.MODE === 'development'}
          >
            <App />
          </ErrorBoundary>
        </SnackbarProvider>
      </DateFormatProvider>
    </AppThemeProvider>
  </React.StrictMode>
);

serviceWorkerRegistration.unregister();

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint.
reportWebVitals();
