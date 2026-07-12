import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    proxy: {
      '/register': 'http://localhost:8080',
      '/login': 'http://localhost:8080',
      '/refresh': 'http://localhost:8080',
      '/chats': 'http://localhost:8080',
      '/users': 'http://localhost:8080',
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
})
