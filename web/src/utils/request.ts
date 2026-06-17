/**
 * Axios 实例封装
 *
 * 提供统一的 HTTP 客户端，配置：
 * - 请求拦截器：注入 Authorization: Bearer <token> + 全局 loading 计数器
 * - 响应拦截器：自动 token 刷新（401 时）、权限错误处理（403）、统一提取 data + loading 递减
 */

import axios, { type AxiosRequestConfig } from 'axios'
import { getToken, setToken as saveToken, removeToken, getRefreshToken, setRefreshToken as saveRefresh, removeRefreshToken } from './auth'
import { refreshToken } from '@/api/auth'
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
// TODO(web/request): loadingState 是模块级全局计数，SSR/多实例测试时会共享状态。
// 如果未来引入服务端渲染或并发组件测试，应改为 Pinia store 或可注入实例。

function incLoading() { loadingState.active++ }
function decLoading() { if (loadingState.active > 0) loadingState.active-- }

// 创建 Axios 实例，baseURL 为空（通过 Vite proxy 转发）
const raw = axios.create({
  timeout: 30000,
})
// TODO(web/request): baseURL 为空依赖 Vite proxy/Nginx 配置。
// 建议从 import.meta.env.VITE_API_BASE_URL 读取，方便测试、预发和生产环境切换。

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

// Token 刷新状态管理
let isRefreshing = false
let refreshSubscribers: Array<{ resolve: (token: string) => void; reject: (error: any) => void }> = []

function subscribeTokenRefresh(resolve: (token: string) => void, reject: (error: any) => void) {
  refreshSubscribers.push({ resolve, reject })
}

function onTokenRefreshed(newToken: string) {
  refreshSubscribers.forEach(({ resolve }) => resolve(newToken))
  refreshSubscribers = []
}

function onTokenRefreshFailed(error: any) {
  refreshSubscribers.forEach(({ reject }) => reject(error))
  refreshSubscribers = []
}

// 响应拦截器：统一错误处理 + Token 自动刷新 + loading 递减
raw.interceptors.response.use(
  (response) => {
    decLoading()
    return response.data
  },
  async (error) => {
    decLoading()
    const { response, config } = error

    if (response) {
      // 401 时尝试刷新 Token
      if (response.status === 401 && config && !config._retry) {
        const rt = getRefreshToken()
        if (rt && !isRefreshing) {
          isRefreshing = true
          config._retry = true
          try {
            const res = await refreshToken(rt)
            const newToken = res.data.access_token
            const newRefresh = res.data.refresh_token
            saveToken(newToken)
            if (newRefresh) saveRefresh(newRefresh)
            config.headers.Authorization = `Bearer ${newToken}`
            onTokenRefreshed(newToken)
            isRefreshing = false
            return raw(config)
          } catch (refreshErr) {
            isRefreshing = false
            onTokenRefreshFailed(refreshErr)
            removeToken()
            removeRefreshToken()
            if (router.currentRoute.value.path !== '/login') {
              router.push('/login')
            }
            return Promise.reject(error)
          }
        } else if (isRefreshing) {
          // 其他请求等待刷新完成
          return new Promise((resolve, reject) => {
            subscribeTokenRefresh(
              (token: string) => {
                config.headers.Authorization = `Bearer ${token}`
                resolve(raw(config))
              },
              (err) => reject(err)
            )
          })
        } else {
          removeToken()
          removeRefreshToken()
          if (router.currentRoute.value.path !== '/login') {
            router.push('/login')
          }
        }
      } else if (response.status === 403) {
        // 已登录但无权限 — 不去登录页，仅输出错误
        console.error('无权限访问该资源')
      } else {
        console.error(response.data?.message || '请求失败')
      }
    } else {
      console.error('网络错误')
    }

    return Promise.reject(error)
  },
)

export default request
