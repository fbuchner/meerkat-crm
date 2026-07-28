import { test, expect } from 'vitest';
import { render } from '@testing-library/react';
import App from './App';
import { AppThemeProvider } from './AppThemeProvider';
import { DateFormatProvider } from './DateFormatProvider';
import { SnackbarProvider } from './context/SnackbarContext';

// Mirrors index.tsx's real provider composition -- App itself doesn't
// include these, they're only added at the root render call, so any
// component that reaches into their context (e.g. BrandLogo's
// useThemePreference) needs the same wrapping here to render at all.
test('renders app component', () => {
  const { container } = render(
    <AppThemeProvider>
      <DateFormatProvider>
        <SnackbarProvider>
          <App />
        </SnackbarProvider>
      </DateFormatProvider>
    </AppThemeProvider>
  );
  expect(container).toBeTruthy();
});
