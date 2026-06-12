/**
 * Axios 实例封装
 *
 * 创建统一的 HTTP 客户端，配置：
 * - 请求拦截器：注入 Authorization: Bearer <token> + 全局 loading 计数器
 * - 响应拦截器：处理 401（跳转登录）、403（提示无权限）、统一提取 data + loading 递减
 */

import axios, { type AxiosRequestConfig } from 'axios'
import { getToken, removeToken } from './auth'
import router from '@/router'

// 响应拦截器已将 AxiosResponse 的 data 提取，因此返回类型应为 T 而非 AxiosResponse<T>。
// 通过类型断言覆盖 axios.create 的返回类型。
interface InterceptedAxiosInstance {
  request<T = unknown>(config: AxiosRequestConfig): Promise<T>
  get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
}

// 全局 loading 计数器 — 模块级共享变量，避免循环依赖
// 由 useLoading composable 在组件中使用，拦截器直接操作此变量
export const loadingState = { active: 0 }

function incLoading() { loadingState.active++ }
function decLoading() { if (loadingState.active > 0) loadingState.active-- }

// 创建 Axios 实例，baseURL 为空（通过 Vite proxy 转发）
const raw = axios.create({
  timeout: 30000,
})

// 类型断言：拦截器已提取 response.data，返回类型简化为 T
const request = raw as unknown as InterceptedAxiosInstance

// 请求拦截器：注入 token + 全局 loading
raw.interceptors.request.use(
  (config) => {
    const token = getToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    // 增加全局 loading 计数
    incLoading()
    return config
  },
  (error) => {
    decLoading()
    return Promise.reject(error)
  },
)

// 响应拦截器：统一错误处理 + loading 递减
raw.interceptors.response.use(
  (response) => {
    decLoading()
    // 统一提取 data 字段
    return response.data
  },
  (error) => {
    decLoading()
    const { response } = error

    if (response) {
      switch (response.status) {
        case 401:
          // 未登录或令牌过期，清除 token 并跳转登录页
          // 防止无限循环：若当前已在登录页则不重复跳转
          removeToken()
          if (router.currentRoute.value.path !== '/login') {
            router.push('/login')
          }
          break
        case 403:
          // 无权限 — 输出错误并跳转登录页（角色不匹配时后端返回 403）
          console.error('无权限访问该资源')
          if (router.currentRoute.value.path !== '/login') {
            router.push('/login')
          }
          break
        default:
          console.error(response.data?.message || '请求失败')
      }
    } else {
      console.error('网络错误')
    }

    return Promise.reject(error)
  },
)

export default request
