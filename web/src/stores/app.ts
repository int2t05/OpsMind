import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  // State
  const sidebarCollapsed = ref(false)
  const unreadMessageCount = ref(0)

  // Actions
  const toggleSidebar = () => {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  const setUnreadMessageCount = (count: number) => {
    unreadMessageCount.value = count
  }

  return {
    sidebarCollapsed,
    unreadMessageCount,
    toggleSidebar,
    setUnreadMessageCount,
  }
})
