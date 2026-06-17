<template>
  <div v-if="chatStore.currentStep || chatStore.pipelineMetrics" class="pipeline-steps">
    <div v-if="chatStore.currentStep" class="step-current">
      <span class="step-dot"></span>{{ chatStore.currentStep }}
    </div>
    <div v-if="chatStore.pipelineMetrics" class="step-metrics">
      <span v-for="s in chatStore.pipelineMetrics.steps" :key="s.id" :class="['step-badge', s.success ? 'done' : 'failed']">{{ s.label }} {{ s.duration_ms }}ms</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useChatStore } from '@/stores/chat'
const chatStore = useChatStore()
</script>

<style scoped>
.pipeline-steps { margin-bottom: 12px; }
.step-current { display: flex; align-items: center; gap: 6px; padding: 6px 10px; background: var(--bg-overlay); border-radius: 6px; font-size: 12px; color: var(--text-secondary); }
.step-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--accent); animation: pulse 1s infinite; }
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
.step-metrics { display: flex; gap: 6px; flex-wrap: wrap; }
.step-badge { padding: 2px 8px; border-radius: 4px; font-size: 11px; }
.step-badge.done { background: rgba(94, 106, 210, 0.08); color: var(--accent); }
.step-badge.failed { background: rgba(248, 81, 73, 0.08); color: var(--tag-rejected-text); }
</style>
