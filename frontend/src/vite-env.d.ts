/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly REACT_APP_API_URL?: string;
  readonly REACT_APP_REQUEST_TIMEOUT?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
