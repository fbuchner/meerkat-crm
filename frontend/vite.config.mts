import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
  const port = Number(loadEnv(mode, process.cwd(), '').PORT) || 7300;

  return {
    plugins: [react()],
    // The Docker files and .env set these names as build-time variables.
    envPrefix: 'REACT_APP_',
    server: { port, strictPort: true, host: true },
    build: {
      // The Dockerfiles copy the bundle from build/.
      outDir: 'build',
      // nginx caches content-hashed files under /static/ as immutable.
      assetsDir: 'static',
    },
  };
});
