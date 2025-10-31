import { render } from '@testing-library/react';
import App from './App';

test('renders app component', () => {
  const { container } = render(<App />);
  expect(container).toBeTruthy();
});
