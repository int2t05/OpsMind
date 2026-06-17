<template>
  <div class="advanced-panel">
    <div class="advanced-row">
      <label class="advanced-label">Top K</label>
      <input v-model.number="chatStore.ragOptions.top_k" type="number" min="1" max="20" class="advanced-input" />
    </div>
    <div class="advanced-toggles">
      <label class="advanced-toggle">
        <input type="checkbox" v-model="chatStore.ragOptions.query_rewrite" /><span>查询改写</span>
      </label>
      <label class="advanced-toggle">
        <input type="checkbox" v-model="chatStore.ragOptions.multi_route" /><span>多路检索</span>
      </label>
      <label class="advanced-toggle">
        <input type="checkbox" v-model="chatStore.ragOptions.hybrid" /><span>混合检索</span>
      </label>
      <label class="advanced-toggle">
        <input type="checkbox" v-model="chatStore.ragOptions.rerank" /><span>重排序</span>
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useChatStore } from '@/stores/chat'
const chatStore = useChatStore()
</script>

<style scoped>
.advanced-panel { padding: 12px 14px; background: var(--bg-overlay); border: 1px solid var(--border-default); border-radius: 8px; margin-bottom: 12px; }
.advanced-row { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
.advanced-label { font-size: 12px; color: var(--text-secondary); flex-shrink: 0; }
.advanced-input { width: 64px; padding: 4px 8px; background: var(--bg-base); border: 1px solid var(--border-default); border-radius: 4px; color: var(--text-primary); font-size: 13px; }
.advanced-toggles { display: flex; flex-wrap: wrap; gap: 12px; }
.advanced-toggle { display: flex; align-items: center; gap: 4px; font-size: 12px; color: var(--text-secondary); cursor: pointer; }
</style>
