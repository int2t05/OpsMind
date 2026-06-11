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
