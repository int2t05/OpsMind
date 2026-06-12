// Vue 应用入口 — 注册 Naive UI、Pinia、Router、全局样式。
// Naive UI 通过 NConfigProvider 提供全局主题，主题切换由 useTheme composable 驱动。
//
// TODO(main): 缺少 app.config.errorHandler — Vue 渲染错误会静默消失，应添加全局错误处理器
//            捕获 render/component 错误并输出到 console.error 或上报到监控系统。

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles/global.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')
