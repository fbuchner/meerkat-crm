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
            showDetails={process.env.NODE_ENV === 'development'}
          >
            <App />
          </ErrorBoundary>
        </SnackbarProvider>
      </DateFormatProvider>
    </AppThemeProvider>
  </React.StrictMode>
);

// If you want your app to work offline and load faster, you can change
// unregister() to register() below. Note this comes with some pitfalls.
// Learn more about service workers: https://cra.link/PWA
serviceWorkerRegistration.unregister();

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
