<template>
  <n-config-provider :theme="naiveTheme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <router-view />
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
// 根组件 — Naive UI 全局配置提供者。
// 通过 NConfigProvider 注入主题覆盖，所有子组件自动响应主题切换。
// NMessageProvider 提供 useMessage() 的全局上下文。
//
// TODO(App): light 主题时 naiveTheme 设为 null（第 21 行）— Naive UI 回退到默认亮色主题，
//           可能与 global.css 中定义的自定义 CSS 变量不一致，导致 light 模式视觉割裂。
//           应考虑为 light 主题也提供完整的 Naive UI theme overrides。

import { computed } from 'vue'
import { darkTheme, NConfigProvider, NMessageProvider } from 'naive-ui'
import { useTheme } from '@/composables/useTheme'
import { darkThemeOverrides, lightThemeOverrides } from '@/theme'

const { theme } = useTheme()

const naiveTheme = computed(() => (theme.value === 'dark' ? darkTheme : null))
const themeOverrides = computed(() =>
  theme.value === 'dark' ? darkThemeOverrides : lightThemeOverrides,
)
</script>
