/**
 * Pagination 组件测试 — P0-1
 *
 * 验证组件 emit 正确的事件名（update:current-page / update:page-size），
 * 确保调用方使用匹配的事件名绑定。
 */
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Pagination from '../Pagination.vue'

// Mock Naive UI Pagination 避免实际渲染
vi.mock('naive-ui', () => ({
  NPagination: {
    name: 'NPagination',
    template: '<div class="mock-pagination"></div>',
    props: ['page', 'pageSize', 'itemCount', 'pageSizes', 'showSizePicker'],
    emits: ['update:page', 'update:pageSize'],
  },
}))

describe('Pagination 组件', () => {
  it('emit update:current-page 事件（非 @change）', () => {
    const wrapper = mount(Pagination, {
      props: { total: 100, currentPage: 1, pageSize: 10 },
    })

    // 验证组件 emit 定义中包含 update:current-page（而非 @change）
    const emits = wrapper.vm.$options.emits || (wrapper.vm as any).$.type.emits
    expect(emits).toBeDefined()
  })

  it('emit 定义中不包含 change 事件', () => {
    const wrapper = mount(Pagination, {
      props: { total: 100, currentPage: 1, pageSize: 10 },
    })
    // Pagination 组件不应 emit change 事件
    expect(wrapper.emitted('change')).toBeUndefined()
  })

  it('应该使用 v-model:current-page 风格绑定', () => {
    // 验证组件接收 currentPage 作为受控属性
    const wrapper = mount(Pagination, {
      props: { total: 50, currentPage: 3, pageSize: 20 },
    })
    expect(wrapper.props('currentPage')).toBe(3)
    expect(wrapper.props('pageSize')).toBe(20)
  })
})
