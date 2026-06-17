// 主题管理 composable — 提供 light/dark 切换能力，持久化到 localStorage。
//
// 为什么用 composable 而非 store：
// 主题切换是纯 UI 行为，不涉及业务状态，composable 更轻量且不依赖 Pinia。
import { ref, computed } from 'vue'

type Theme = 'dark' | 'light'

const STORAGE_KEY = 'opsmind-theme'
// TODO(composable/theme): 顶层直接访问 localStorage/document，不适合 SSR 和单元测试环境。
// 可在 useTheme 内惰性初始化，并检查 typeof window !== 'undefined'。
const currentTheme = ref<Theme>((localStorage.getItem(STORAGE_KEY) as Theme) || 'dark')

function applyTheme(theme: Theme) {
  document.documentElement.setAttribute('data-theme', theme)
  localStorage.setItem(STORAGE_KEY, theme)
}

applyTheme(currentTheme.value)

export function useTheme() {
  const isDark = computed(() => currentTheme.value === 'dark')

  const toggleTheme = () => {
    currentTheme.value = currentTheme.value === 'dark' ? 'light' : 'dark'
    applyTheme(currentTheme.value)
  }

  return { theme: currentTheme, toggleTheme, isDark }
}
