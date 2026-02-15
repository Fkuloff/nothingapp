import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const useProxy = env.VITE_USE_PROXY === 'true'
  const proxyTarget = env.VITE_PROXY_TARGET || env.VITE_API_BASE_URL || 'http://localhost:8080'

  const proxy = useProxy
    ? {
        '/api': {
          target: proxyTarget,
          changeOrigin: true,
        },
        '/static': {
          target: proxyTarget,
          changeOrigin: true,
        },
        '/uploads': {
          target: proxyTarget,
          changeOrigin: true,
        },
        '/ws': {
          target: proxyTarget.replace(/^http/, 'ws'),
          changeOrigin: true,
          ws: true,
        },
      }
    : undefined

  return {
    plugins: [react()],
    server: useProxy ? { proxy } : undefined,
  }
})
