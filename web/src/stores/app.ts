import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getUnreadCount } from '@/api/message'

export const useAppStore = defineStore('app', () => {
  const sidebarCollapsed = ref(false)
  const unreadMessageCount = ref(0)

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  async function fetchUnreadCount() {
    try {
      const res = await getUnreadCount()
      const data = (res as any).data || res
      unreadMessageCount.value = data?.count ?? data ?? 0
    } catch {
      // 静默失败，保留上次计数值
    }
  }

  function setUnreadCount(count: number) {
    unreadMessageCount.value = count
  }

  function decrementUnread() {
    if (unreadMessageCount.value > 0) unreadMessageCount.value--
  }

  return {
    sidebarCollapsed,
    unreadMessageCount,
    toggleSidebar,
    fetchUnreadCount,
    setUnreadCount,
    decrementUnread,
  }
})
