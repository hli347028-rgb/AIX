import { join } from "path";
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import postcssPluginPx2rem from "postcss-plugin-px2rem";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
    const env = loadEnv(mode, process.cwd(), '')
    const proxyTarget = env.VITE_PROXY_TARGET || 'https://aixai.pro'
    return {
    plugins: [vue({
        reactivityTransform: true,
    }),],
    resolve: {
        alias: {
            '@': join(__dirname, "src"),
        }
    },
    server: {
        host: '0.0.0.0',
        open: false,
        // 本地 nginx(8081) 把 / 反代到 9200，需与 nginx.conf 保持一致
        port: 9200,
        strictPort: true,
        proxy: {
          '/api': {
            target: proxyTarget,
            changeOrigin: true,
          },
          '/v1': {
            target: proxyTarget,
            changeOrigin: true,
          },
          // 供预览用注入钱包(devWallet.ts)转发只读 JSON-RPC，绕开公共节点的 CORS
          '/dev-rpc/eoeo': {
            target: env.VITE_RPC_URL || 'https://rpc1.eoeo.info',
            changeOrigin: true,
            rewrite: () => '/',
          },
          '/dev-rpc/bsc': {
            target: env.VITE_BSC_RPC_URL || 'https://bsc-dataseed.binance.org',
            changeOrigin: true,
            rewrite: () => '/',
          }
        }
    },
    build: {
        outDir: "dist",
        assetsDir: "static",
        assetsInlineLimit: 150000
    },
    css: {
        preprocessorOptions: {
            less: {
                charset: false,
                additionalData: '@import "./src/style/global.less";',
            }
        },
        postcss: {
            plugins: [
                postcssPluginPx2rem({
                    rootValue: 37.5,
                    exclude: /(node_module)/,
                    mediaQuery: false,
                }),
            ]
        },
    }
    }
})
