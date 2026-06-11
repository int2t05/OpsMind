// Vue 应用入口 — 注册 Naive UI、Pinia、Router、全局样式。
// Naive UI 通过 NConfigProvider 提供全局主题，主题切换由 useTheme composable 驱动。

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles/global.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')
