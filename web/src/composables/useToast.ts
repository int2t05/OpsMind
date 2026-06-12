/**
 * 共享 Toast 通知 composable
 *
 * 统一项目中 7 处重复的 toast 实现（ref + showToast + setTimeout 自动消失）。
 * 替代各页面独立定义的 toast 逻辑，自动在组件卸载时清理定时器。
 *
 * 使用方式：
 *   const toast = useToast()
 *   toast.showToast('保存成功', 'success')
 *   // 模板中使用 toast.visible / toast.message / toast.type
 */
import { ref, onUnmounted } from 'vue'

export type ToastType = 'success' | 'error' | 'info'

export function useToast(defaultDuration = 2000) {
  const visible = ref(false)
  const message = ref('')
  const type = ref<ToastType>('info')
  let timer: ReturnType<typeof setTimeout> | null = null

  function showToast(msg: string, toastType: ToastType = 'info') {
    visible.value = true
    message.value = msg
    type.value = toastType

    // 清除之前的定时器（防止多次调用堆积）
    if (timer) clearTimeout(timer)

    timer = setTimeout(() => {
      visible.value = false
    }, defaultDuration)
  }

  function clearToast() {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }

  // 组件卸载时自动清理定时器，防止内存泄漏
  onUnmounted(() => {
    clearToast()
  })

  return { visible, message, type, showToast, clearToast }
}
