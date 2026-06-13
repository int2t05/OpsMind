<template>
  <div class="chat-page">
    <div class="chat-container">
      <!-- 知识库选择 -->
      <div class="kb-selector" v-if="knowledgeBases.length > 0">
    <!-- 知识库错误提示 -->
    <div v-if="kbError" class="kb-error-banner">{{ kbError }}</div>

        <label class="kb-label">选择知识库</label>
        <select v-model="selectedKB" class="kb-select">
          <option v-for="kb in knowledgeBases" :key="kb.id" :value="kb.id">
            {{ kb.name }}
          </option>
        </select>

        <!-- v2: RAG 高级设置 -->
        <button class="btn-advanced" :class="{ active: showAdvanced }" @click="showAdvanced = !showAdvanced">
          ⚙ 高级
        </button>
      </div>

      <!-- v2: RAG 高级设置面板 -->
    <!-- 知识库为空提示 -->
    <div v-if="knowledgeBases.length === 0 && !kbError" class="kb-empty-hint">暂无可用知识库</div>

      <div v-if="showAdvanced" class="advanced-panel">
        <div class="advanced-row">
          <label class="advanced-label">Top K</label>
          <input v-model.number="chatStore.ragOptions.top_k" type="number" min="1" max="20" class="advanced-input" />
        </div>
        <div class="advanced-toggles">
          <label class="advanced-toggle">
            <input type="checkbox" v-model="chatStore.ragOptions.query_rewrite" />
            <span>查询改写</span>
          </label>
          <label class="advanced-toggle">
            <input type="checkbox" v-model="chatStore.ragOptions.multi_route" />
            <span>多路检索</span>
          </label>
          <label class="advanced-toggle">
            <input type="checkbox" v-model="chatStore.ragOptions.hybrid" />
            <span>混合检索</span>
          </label>
          <label class="advanced-toggle">
            <input type="checkbox" v-model="chatStore.ragOptions.rerank" />
            <span>重排序</span>
          </label>
        </div>
      </div>

      <!-- v2: RAG 管道步骤指示器 -->
      <div v-if="chatStore.currentStep || chatStore.pipelineMetrics" class="pipeline-steps">
        <div v-if="chatStore.currentStep" class="step-current">
          <span class="step-dot"></span>
          {{ chatStore.currentStep }}
        </div>
        <div v-if="chatStore.pipelineMetrics" class="step-metrics">
          <span v-for="s in chatStore.pipelineMetrics.steps" :key="s.id" :class="['step-badge', s.success ? 'done' : 'failed']">
            {{ s.label }} {{ s.duration_ms }}ms
          </span>
        </div>
      </div>

      <!-- 消息列表（子组件） -->
      <ChatMessageList
        ref="msgListRef"
        :messages="chatStore.messages"
        :loading="chatStore.loading"
        :is-streaming="chatStore.streaming"
      />

      <!-- 输入区域 -->
      <div class="input-area">
        <textarea
          v-model="question"
          class="chat-input"
          rows="3"
          placeholder="输入您的问题..."
          :disabled="chatStore.loading || !selectedKB"
          @keydown.enter.exact.prevent="handleSend"
        ></textarea>
        <div class="input-actions">
          <button
            class="btn-send"
            :disabled="!question.trim() || chatStore.loading || !selectedKB"
            @click="handleSend"
          >
            {{ chatStore.loading ? '思考中...' : '发送' }}
          </button>
        </div>
      </div>

      <!-- 低置信度引导 -->
      <div v-if="chatStore.currentSession?.can_submit_ticket" class="ticket-cta">
        <p>暂未找到足够匹配的知识，建议提交申告由运维人员人工处理</p>
        <router-link
          :to="{
            path: '/portal/tickets/submit',
            query: { chat_context: JSON.stringify({ question: chatStore.currentSession.question, answer: chatStore.currentSession.answer }) }
          }"
          class="btn-submit-ticket"
        >
          提交申告
        </router-link>
      </div>

      <!-- 反馈区域 -->
      <div v-if="chatStore.currentSession && !chatStore.currentSession.can_submit_ticket && !chatStore.currentSession.feedback" class="feedback-area">
        <span class="feedback-label">这个回答对您有帮助吗？</span>
        <button class="btn-feedback" @click="handleFeedback(1)">已解决</button>
        <button class="btn-feedback btn-feedback--no" @click="handleFeedback(2)">未解决</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// TODO(portal/Chat): 组件超过 560 行 — 应拆分为消息渲染子组件、高级设置面板子组件等。
import { ref, onMounted, nextTick } from 'vue'
import { useChatStore } from '@/stores/chat'
import { listKnowledgeBasesForPortal } from '@/api/knowledge'
import ChatMessageList from './ChatMessageList.vue'

const chatStore = useChatStore()
const question = ref('')
const selectedKB = ref<number | null>(null)
const knowledgeBases = ref<Array<{ id: number; name: string }>>([])
const msgListRef = ref<InstanceType<typeof ChatMessageList> | null>(null)
const showAdvanced = ref(false)
const kbError = ref('')

onMounted(async () => {
  try {
    const res = await listKnowledgeBasesForPortal()
    const data = (res as any).data || res
    const items = data?.items || data
    knowledgeBases.value = Array.isArray(items) ? items : []
    if (knowledgeBases.value.length > 0) {
      selectedKB.value = knowledgeBases.value[0].id
    } else {
      kbError.value = '暂无可用知识库，请联系管理员创建知识库'
    }
  } catch (err) {
    kbError.value = '加载知识库列表失败，请检查网络连接'
    console.error('加载知识库列表失败', err)
  }
})

async function handleSend() {
  const q = question.value.trim()
  if (!q || !selectedKB.value) return
  // 只靠 input min/max 不能阻止手动输入或脚本修改。

  question.value = ''
  await chatStore.sendQuestion(q, selectedKB.value)
  await nextTick()
  scrollToBottom()
}

async function handleFeedback(value: number) {
  // 目前用户可以重复点击，且失败时没有提示。
  await chatStore.submitFeedback(value)
}

function scrollToBottom() {
  if (msgListRef.value) {
    msgListRef.value.scrollToBottom()
  }
}
</script>

<style scoped>
.chat-page {
  max-width: 800px;
  margin: 0 auto;
}

.chat-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 120px);
}

/* 知识库选择器 */
.kb-selector {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-default);
  margin-bottom: 16px;
}

.kb-label {
  font-size: 13px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.kb-select {
  padding: 6px 12px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  font-family: inherit;
  flex: 1;
}

/* v2: 高级设置按钮 */
.btn-advanced {
  padding: 6px 12px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  font-family: inherit;
  flex-shrink: 0;
}
.btn-advanced:hover { border-color: var(--accent); color: var(--text-primary); }
.btn-advanced.active { border-color: var(--accent); color: var(--accent); }

/* v2: 高级设置面板 */
.advanced-panel {
  padding: 12px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  margin-bottom: 12px;
}

.advanced-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.advanced-label {
  font-size: 12px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.advanced-input {
  width: 64px;
  padding: 4px 8px;
  background: var(--bg-base);
  border: 1px solid var(--border-default);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 13px;
  font-family: inherit;
  text-align: center;
}

.advanced-toggles {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.advanced-toggle {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
}

/* v2: 管道步骤指示器 */
.pipeline-steps {
  padding: 8px 14px;
  margin-bottom: 12px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 6px;
}

.step-current {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--accent);
  margin-bottom: 6px;
}

.step-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.step-metrics {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.step-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
}

.step-badge.done {
  background: rgba(46, 160, 67, 0.12);
  color: var(--color-success);
}

.step-badge.failed {
  background: rgba(248, 81, 73, 0.12);
  color: var(--tag-rejected-text);
}

/* 消息区域 */
.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.empty-chat {
  text-align: center;
  padding: 64px 24px;
  color: var(--text-secondary);
  font-size: 16px;
}

.sub-text {
  font-size: 13px;
  margin-top: 8px;
  opacity: 0.6;
}

.message {
  margin-bottom: 20px;
  display: flex;
}

.message--user {
  justify-content: flex-end;
}

.message--assistant {
  justify-content: flex-start;
}

.message-bubble {
  max-width: 75%;
  padding: 12px 16px;
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
  border-bottom-left-radius: 4px;
  border: 1px solid var(--border-default);
}

/* 来源 */
.sources {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border-default);
}

.sources-title {
  font-size: 11px;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.source-item {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  padding: 3px 0;
}

.source-name {
  color: var(--accent);
}

.source-confidence {
  color: var(--text-secondary);
  font-size: 11px;
}

/* 加载指示器 */
.loading-indicator {
  display: flex;
  gap: 6px;
  padding: 12px 16px;
}

.loading-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-secondary);
  animation: bounce 1.4s infinite ease-in-out both;
}

.loading-dot:nth-child(1) { animation-delay: -0.32s; }
.loading-dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

/* 输入区域 */
.input-area {
  border-top: 1px solid var(--border-default);
  padding-top: 12px;
  margin-top: 8px;
}

.chat-input {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 14px;
  font-family: inherit;
  resize: none;
}

.chat-input:focus {
  outline: none;
  border-color: var(--accent);
}

.input-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.btn-send {
  padding: 8px 28px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
}

.btn-send:hover { background: var(--accent-hover); }
.btn-send:disabled { opacity: 0.5; cursor: not-allowed; }

/* 转申告 CTA */
.ticket-cta {
  text-align: center;
  padding: 16px;
  margin-top: 12px;
  background: rgba(94, 106, 210, 0.08);
  border: 1px solid rgba(94, 106, 210, 0.15);
  border-radius: 8px;
}

.ticket-cta p {
  color: var(--text-secondary);
  font-size: 13px;
  margin-bottom: 10px;
}

.btn-submit-ticket {
  display: inline-block;
  padding: 8px 20px;
  background: var(--accent);
  color: #fff;
  border-radius: 6px;
  text-decoration: none;
  font-size: 13px;
}

/* 反馈 */
.feedback-area {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 0;
}

.feedback-label {
  font-size: 13px;
  color: var(--text-secondary);
}

.btn-feedback {
  padding: 5px 14px;
  background: var(--bg-overlay);
  color: var(--text-primary);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  font-family: inherit;
}

.btn-feedback:hover { border-color: var(--accent); }

.btn-feedback--no:hover { border-color: var(--tag-rejected-text); color: var(--tag-rejected-text); }

/* 流式输出光标动画 */
.streaming-cursor {
  display: inline;
  animation: blink 1s step-end infinite;
  color: var(--accent);
  font-weight: 200;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}
</style>
