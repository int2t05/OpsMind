<template>
  <div class="messages-area" ref="containerRef">
    <div v-if="messages.length === 0 && !loading" class="empty-chat">
      <p>欢迎使用智能问答</p>
      <p class="sub-text">选择一个知识库，输入您的问题开始对话</p>
    </div>

    <div
      v-for="msg in messages"
      :key="msg.id"
      :class="['message', msg.role === 'user' ? 'message--user' : 'message--assistant']"
    >
      <div class="message-bubble">
        <div class="message-content">
          {{ msg.content }}
          <span v-if="msg.isStreaming && isStreaming" class="streaming-cursor">▊</span>
        </div>
        <div v-if="msg.sources && msg.sources.length > 0 && !msg.isStreaming" class="sources">
          <div class="sources-title">参考来源：</div>
          <div v-for="(src, si) in msg.sources" :key="si" class="source-item">
            <span class="source-name">{{ src.doc_name }}</span>
            <span class="source-confidence">{{ (src.confidence * 100).toFixed(0) }}%</span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="loading && !isStreaming" class="loading-indicator">
      <span class="loading-dot"></span>
      <span class="loading-dot"></span>
      <span class="loading-dot"></span>
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * 聊天消息列表 — 从 Chat.vue 提取。
 *
 * 展示消息气泡、参考来源、流式光标和加载指示器。
 * 通过 props 接收消息列表，不直接依赖 store。
 */
import { ref } from 'vue'
import type { SourceItem } from '@/api/chat'

interface ChatMessage {
  id: string
  role: string
  content: string
  sources?: SourceItem[]
  isStreaming?: boolean
}

defineProps<{
  messages: ChatMessage[]
  loading: boolean
  isStreaming: boolean
}>()

const containerRef = ref<HTMLElement | null>(null)

/** 滚动到底部（外部通过 ref 调用或 watch 触发） */
function scrollToBottom() {
  if (containerRef.value) {
    containerRef.value.scrollTop = containerRef.value.scrollHeight
  }
}

defineExpose({ scrollToBottom })
</script>

<style scoped>
.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.empty-chat { text-align: center; padding: 64px 0; }
.empty-chat p { font-size: 16px; color: var(--text-primary); font-weight: 510; }
.sub-text { font-size: 14px !important; color: var(--text-secondary) !important; font-weight: 400 !important; margin-top: 8px; }

.message { margin-bottom: 20px; display: flex; }
.message--user { justify-content: flex-end; }
.message--assistant { justify-content: flex-start; }

.message-bubble {
  max-width: 75%;
  padding: 12px 18px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.6;
}

.message--user .message-bubble {
  background: var(--accent);
  color: #fff;
  border-bottom-right-radius: 4px;
}

.message--assistant .message-bubble {
  background: var(--bg-overlay);
  color: var(--text-primary);
  border: 1px solid var(--border-default);
  border-bottom-left-radius: 4px;
}

.message-content { white-space: pre-wrap; word-break: break-word; }

.streaming-cursor {
  display: inline-block;
  animation: blink 1s step-end infinite;
  color: var(--accent);
}

@keyframes blink { 50% { opacity: 0; } }

.sources { margin-top: 10px; padding-top: 10px; border-top: 1px solid var(--border-default); }
.sources-title { font-size: 11px; color: var(--text-secondary); margin-bottom: 6px; }

.source-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
  font-size: 12px;
}

.source-name { color: var(--text-secondary); }
.source-confidence { color: var(--accent); font-weight: 500; }

.loading-indicator {
  display: flex;
  gap: 6px;
  padding: 16px;
  justify-content: center;
}

.loading-dot {
  width: 8px;
  height: 8px;
  background: var(--accent);
  border-radius: 50%;
  animation: dotPulse 1.4s infinite ease-in-out both;
}

.loading-dot:nth-child(1) { animation-delay: -0.32s; }
.loading-dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes dotPulse {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}
</style>
