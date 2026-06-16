import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      // 搜索流使用特殊配置
      '/api/search/stream': {
        target: 'http://localhost:6464',
        changeOrigin: true,
        timeout: 0,
        proxyTimeout: 0,
      },
      // 其他API
      '/api': {
        target: 'http://localhost:6464',
        changeOrigin: true,
      },
    },
  },
})
