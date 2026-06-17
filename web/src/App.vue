<template>
  <n-config-provider :theme="naiveTheme" :theme-overrides="themeOverrides">
    <router-view />
  </n-config-provider>
</template>

<script setup lang="ts">
// 根组件 — Naive UI 全局配置提供者。
// 通过 NConfigProvider 注入主题覆盖，所有子组件自动响应主题切换。
// light 主题使用 Naive UI 内置 lightTheme + 自定义 themeOverrides，
// 确保与 global.css 的自定义 CSS 变量视觉一致。

import { computed } from 'vue'
import { darkTheme, lightTheme, NConfigProvider } from 'naive-ui'
import { useTheme } from '@/composables/useTheme'
import { darkThemeOverrides, lightThemeOverrides } from '@/theme'

const { theme } = useTheme()

const naiveTheme = computed(() => (theme.value === 'dark' ? darkTheme : lightTheme))
const themeOverrides = computed(() =>
  theme.value === 'dark' ? darkThemeOverrides : lightThemeOverrides,
)
</script>
