/**
 * 菜单相关类型定义
 *
 * 替代 auth.ts 中 `menus: any[]` 的类型丢失问题。
 */

export interface MenuItem {
  id: number
  name: string
  path: string
  icon: string
  children?: MenuItem[]
}
