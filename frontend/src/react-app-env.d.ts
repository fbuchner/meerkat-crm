/// <reference types="react-scripts" />

// react-scripts provides ambient types for images, SVGs and CSS Modules, but not
// for plain side-effect stylesheet imports (e.g. `import './App.css'`). Under
// `moduleResolution: "bundler"` TypeScript requires a declaration for these,
// otherwise it reports TS2882.
declare module '*.css';
declare module '*.scss';
declare module '*.sass';
