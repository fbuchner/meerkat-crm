// Extends Vitest's expect with jest-dom matchers (toBeInTheDocument, etc.)
import '@testing-library/jest-dom/vitest';

// jsdom does not implement scrolling; App calls window.scrollTo on route changes
window.scrollTo = () => {};
