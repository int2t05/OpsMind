<template>
  <div class="dashboard-page">
    <div class="page-header">
      <h1 class="page-title">数据看板</h1>
      <span class="page-subtitle">运维运营概览</span>
    </div>

    <!-- 加载中 -->
    <n-spin v-if="loading" size="medium" class="loading-state" />

    <!-- 统计卡片 -->
    <n-grid v-else :cols="7" :x-gap="14" :y-gap="14" responsive="screen" class="stats-grid">
      <n-gi v-for="card in statCards" :key="card.label" :span="1">
        <n-card :bordered="true" size="small" class="stat-card">
          <n-statistic :label="card.label" :value="card.value" />
        </n-card>
      </n-gi>
    </n-grid>

    <!-- 加载错误 -->
    <n-alert v-if="error" type="error" class="error-alert" closable @close="error = ''">
      {{ error }} <n-button size="tiny" @click="fetchStats">重试</n-button>
    </n-alert>

    <!-- 趋势图区域 -->
    <n-card :bordered="true" size="small" class="trends-section">
      <template #header>
        <div class="trends-header">
          <span class="section-title">趋势数据</span>
          <n-space>
            <n-date-picker v-model:value="trendDateRange" type="daterange"
              :default-value="trendDateRange" size="small" clearable @update:value="fetchTrends" />
          </n-space>
        </div>
      </template>

      <n-spin v-if="trendsLoading" size="small" class="loading-state" />

      <n-empty v-else-if="trendPoints.length === 0" description="暂无趋势数据" class="empty-state" />

      <div v-else class="trend-chart">
        <div class="chart-bars">
          <div v-for="(point, i) in trendPoints" :key="i" class="chart-column">
            <div class="bar-group">
              <div class="bar bar--ticket"
                :style="{ height: barHeight(point.ticket_count, maxTrendValue) + 'px' }"
                :title="'申告: ' + point.ticket_count" />
              <div class="bar bar--chat"
                :style="{ height: barHeight(point.chat_count, maxTrendValue) + 'px' }"
                :title="'问答: ' + point.chat_count" />
            </div>
            <div class="chart-label">{{ formatDate(point.date) }}</div>
          </div>
        </div>
        <div class="chart-legend">
          <n-space>
            <n-tag :bordered="false" size="tiny" type="primary">申告</n-tag>
            <n-tag :bordered="false" size="tiny" :color="{ color: 'rgba(94,106,210,0.35)', textColor: '#d0d6e0' }">问答</n-tag>
          </n-space>
        </div>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
// TODO(admin/Dashboard): 使用 (res as any) 强制类型转换 — 等 API 泛型补全后移除。
import { ref, onMounted, computed } from 'vue'
import {
  NCard, NGrid, NGi, NStatistic, NButton, NDatePicker, NSpace,
  NSpin, NEmpty, NAlert, NTag,
} from 'naive-ui'
import { getStats, getTrends, type StatsData, type TrendDataPoint } from '@/api/dashboard'

const loading = ref(true)
const error = ref('')
const stats = ref<StatsData>({
  today_tickets: 0, pending_tickets: 0, processing_tickets: 0,
  resolved_tickets: 0, today_chats: 0, avg_confidence: 0, knowledge_count: 0,
})

const statCards = computed(() => [
  { label: '今日申告', value: stats.value.today_tickets },
  { label: '待处理', value: stats.value.pending_tickets },
  { label: '处理中', value: stats.value.processing_tickets },
  { label: '已解决', value: stats.value.resolved_tickets },
  { label: '今日问答', value: stats.value.today_chats },
  { label: '平均置信度', value: `${(stats.value.avg_confidence * 100).toFixed(0)}%` },
  { label: '知识条目数', value: stats.value.knowledge_count },
])

// 趋势数据
const trendsLoading = ref(false)
const trendPoints = ref<TrendDataPoint[]>([])
const trendDateRange = ref<[number, number]>()

onMounted(() => {
  const end = new Date()
  const start = new Date(end)
  start.setDate(start.getDate() - 6)
  trendDateRange.value = [start.getTime(), end.getTime()]
  fetchStats()
  fetchTrends()
})

const maxTrendValue = computed(() => {
  let max = 1
  for (const p of trendPoints.value) max = Math.max(max, p.ticket_count, p.chat_count)
  return max
})

async function fetchStats() {
  loading.value = true; error.value = ''
  try {
    const res = await getStats()
    const data = (res as any).data || res
    if (data) stats.value = data
  } catch (e: any) {
    error.value = e?.response?.data?.message || e?.message || '网络错误'
  } finally { loading.value = false }
}

async function fetchTrends() {
  if (!trendDateRange.value) return
  trendsLoading.value = true
  const [start, end] = trendDateRange.value
  try {
    const res = await getTrends({
      start_date: new Date(start).toISOString().slice(0, 10),
      end_date: new Date(end).toISOString().slice(0, 10),
      granularity: 'day',
    })
    const data = (res as any).data || res
    trendPoints.value = data?.data_points || []
  } catch { trendPoints.value = [] }
  finally { trendsLoading.value = false }
}

function barHeight(count: number, max: number): number {
  if (max <= 0) return 0
  return Math.max(4, Math.round((count / max) * 120))
}

function formatDate(dateStr: string): string {
  const parts = dateStr.split('-')
  return parts.length === 3 ? `${parts[1]}/${parts[2]}` : dateStr
}
</script>

<style scoped>
.dashboard-page { max-width: 1100px; }
.page-header { margin-bottom: 28px; }
.page-title { font-size: 22px; font-weight: var(--font-weight-emphasis); }
.page-subtitle { font-size: 13px; color: var(--text-secondary); margin-left: 10px; }

.stats-grid { margin-bottom: 32px; }
.stat-card { text-align: center; }

.loading-state, .empty-state { text-align: center; padding: 48px 24px; }
.error-alert { margin-bottom: 16px; }

.trends-section { margin-bottom: 32px; }
.trends-header { display: flex; justify-content: space-between; align-items: center; width: 100%; }
.section-title { font-size: 16px; font-weight: var(--font-weight-emphasis); }

.trend-chart { padding-top: 8px; }
.chart-bars { display: flex; align-items: flex-end; gap: 6px; height: 140px; padding: 4px 0; border-bottom: 1px solid var(--border-subtle); }
.chart-column { flex: 1; display: flex; flex-direction: column; align-items: center; }
.bar-group { display: flex; align-items: flex-end; gap: 3px; height: 120px; }
.bar { width: 12px; border-radius: 3px 3px 0 0; transition: height 0.3s ease; min-height: 4px; }
.bar--ticket { background: var(--accent); }
.bar--chat { background: rgba(94, 106, 210, 0.3); }
.chart-label { font-size: 10px; color: var(--text-secondary); margin-top: 6px; white-space: nowrap; }
.chart-legend { padding-top: 12px; }
</style>
