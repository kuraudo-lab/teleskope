import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
export default defineConfig({
  plugins: [vue()],
  define: { 'process.env.NODE_ENV': JSON.stringify('production') },
  build: {
    outDir: '../internal/report/assets', emptyOutDir: true,
    lib: { entry: 'src/main.ts', name: 'TeleskopeUI', formats: ['iife'], fileName: () => 'ui.js', cssFileName: 'ui' },
    cssCodeSplit: false,
  },
})
