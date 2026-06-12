/**
 * 全局 Loading 指示器
 *
 * 与 request.ts 共享 loadingState 变量，拦截器自动递增/递减。
 * 在布局组件中使用此 composable 获取 loading 状态。
 *
 * 使用方式：
 *   const { isLoading } = useLoading()
 *   // 模板中绑定 v-if="isLoading" 或进度条
 */
import { computed } from 'vue'
import { loadingState } from '@/utils/request'

export function useLoading() {
  const isLoading = computed(() => loadingState.active > 0)

  return { isLoading }
}
