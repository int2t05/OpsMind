<template>
  <div class="knowledge-edit-page">
    <div class="page-header">
      <button class="btn-back" @click="goBack">← 返回列表</button>
      <h1>{{ isNew ? '新建文章' : '编辑文章' }}</h1>
      <span v-if="article" :class="['status-tag', statusClass(article.status)]">{{ statusText(article.status) }}</span>
    </div>

    <div class="form-card">
      <div class="form-group" v-if="isNew">
        <label>所属知识库 <span class="required">*</span></label>
        <select v-model="form.kb_id" class="form-select">
          <option :value="0" disabled>请选择知识库</option>
          <option v-for="kb in kbList" :key="kb.id" :value="kb.id">{{ kb.name }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>问题 <span class="required">*</span></label>
        <input v-model="form.question" type="text" placeholder="输入常见问题" :disabled="!canEdit" />
      </div>

      <div class="form-group">
        <label>答案 <span class="required">*</span></label>
        <textarea v-model="form.answer" placeholder="输入标准答案" rows="8" :disabled="!canEdit" />
      </div>

      <div class="form-group">
        <label>分类</label>
        <input v-model="form.category" type="text" placeholder="如：账号管理、网络故障" :disabled="!canEdit" />
      </div>

      <div class="form-group">
        <label>标签</label>
        <div class="tags-input">
          <span v-for="(tag, i) in form.tags" :key="i" class="tag">{{ tag }}<button v-if="canEdit" class="tag-remove" @click="removeTag(i)">×</button></span>
          <input v-if="canEdit" v-model="tagInput" type="text" placeholder="输入标签后按回车" class="tag-input" @keydown.enter.prevent="addTag" />
        </div>
      </div>

      <!-- 编辑操作 -->
      <div class="form-actions" v-if="canEdit && !isReview">
        <button class="btn-save" @click="handleSave" :disabled="saving">{{ isNew ? '创建草稿' : '保存修改' }}</button>
        <button v-if="!isNew" class="btn-submit" @click="handleSubmitReview">提交审核</button>
        <button class="btn-cancel" @click="goBack">取消</button>
      </div>

      <!-- 审核区域 -->
      <div class="review-section" v-if="isReview">
        <h3>审核操作</h3>
        <div class="form-group"><label>审核意见</label><textarea v-model="reviewComment" placeholder="驳回时请填写审核意见" rows="3" /></div>
        <div class="review-actions">
          <button class="btn-approve" @click="handleReview(true)">✓ 通过</button>
          <button class="btn-reject" @click="handleReview(false)">✗ 驳回</button>
        </div>
      </div>

      <!-- 已通过/已发布操作 -->
      <div class="form-actions" v-if="article?.status === 3">
        <button class="btn-publish" @click="handlePublish">发布到 RAG</button>
      </div>
      <div class="form-actions" v-if="article?.status === 4">
        <button class="btn-disable" @click="handleDisable">停用</button>
        <button v-if="article?.sync_status === 'failed'" class="btn-retry" @click="handleRetrySync">重试同步</button>
      </div>
      <div class="form-actions" v-if="article?.status === 0">
        <button class="btn-save" @click="handleEnable">恢复为草稿</button>
      </div>

      <!-- 切片状态 -->
      <div v-if="article?.chunks?.length" class="chunk-info">
        <h4>切片同步状态</h4>
        <div v-for="chunk in article.chunks" :key="chunk.id" class="chunk-item">
          <span :class="['sync-badge', chunk.sync_status]">{{ chunk.sync_status }}</span>
          <span class="chunk-content">{{ chunk.content?.substring(0, 80) }}...</span>
          <span v-if="chunk.sync_error" class="sync-error">{{ chunk.sync_error }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { listKnowledgeBases, getArticleDetail, createArticle, updateArticle, submitReview, reviewArticle, publishArticle, disableArticle, enableArticle, retrySyncArticle } from '../../api/knowledge'

const router = useRouter()
const route = useRoute()
const articleId = route.params.id as string
const kbIdFromQuery = route.query.kb_id as string

const isNew = computed(() => !articleId || articleId === 'new')
const isReview = computed(() => article.value?.status === 2)
const canEdit = computed(() => isNew.value || article.value?.status === 1 || article.value?.status === 5)

const kbList = ref<any[]>([])
const article = ref<any>(null)
const saving = ref(false)
const tagInput = ref('')
const reviewComment = ref('')

const form = ref({ kb_id: kbIdFromQuery ? Number(kbIdFromQuery) : 0, question: '', answer: '', category: '', tags: [] as string[] })

onMounted(async () => {
  await fetchKBs()
  if (!isNew.value) await fetchArticle()
})

const fetchKBs = async () => {
  try { const res = await listKnowledgeBases(); kbList.value = (res.data as any).items || (res as any).items || [] } catch (e) { console.error(e) }
}
const fetchArticle = async () => {
  try {
    const res = await getArticleDetail(Number(articleId))
    article.value = res.data
    form.value = { kb_id: res.data.kb_id, question: res.data.question, answer: res.data.answer, category: res.data.category || '', tags: res.data.tags || [] }
  } catch (e) { console.error(e) }
}

const addTag = () => { const t = tagInput.value.trim(); if (t && !form.value.tags.includes(t)) form.value.tags.push(t); tagInput.value = '' }
const removeTag = (i: number) => { form.value.tags.splice(i, 1) }
const goBack = () => { router.back() }

const handleSave = async () => {
  if (!form.value.question || !form.value.answer) { alert('问题和答案不能为空'); return }
  saving.value = true
  try {
    const payload = { question: form.value.question, answer: form.value.answer, category: form.value.category, tags: form.value.tags }
    if (isNew.value) { await createArticle(form.value.kb_id, { ...payload, kb_id: form.value.kb_id }) }
    else { await updateArticle(Number(articleId), payload) }
    router.back()
  } catch (e: any) { alert(e?.message || '保存失败') } finally { saving.value = false }
}
const handleSubmitReview = async () => { try { await submitReview(Number(articleId)); await fetchArticle() } catch (e: any) { alert(e?.message) } }
const handleReview = async (approved: boolean) => {
  if (!approved && !reviewComment.value.trim()) { alert('驳回时必须填写审核意见'); return }
  try { await reviewArticle(Number(articleId), { approved, review_comment: reviewComment.value }); router.back() } catch (e: any) { alert(e?.message) }
}
const handlePublish = async () => { try { await publishArticle(Number(articleId)); await fetchArticle() } catch (e: any) { alert(e?.message) } }
const handleDisable = async () => { if (!confirm('确定停用？')) return; try { await disableArticle(Number(articleId)); await fetchArticle() } catch (e: any) { alert(e?.message) } }
const handleEnable = async () => { try { await enableArticle(Number(articleId)); await fetchArticle() } catch (e: any) { alert(e?.message) } }
const handleRetrySync = async () => { try { await retrySyncArticle(Number(articleId)); await fetchArticle() } catch (e: any) { alert(e?.message) } }

const statusClass = (s: number) => { const m: Record<number,string> = { 0:'disabled',1:'draft',2:'pending',3:'approved',4:'published',5:'rejected' }; return m[s]||'' }
const statusText = (s: number) => { const m: Record<number,string> = { 0:'已停用',1:'草稿',2:'待审核',3:'已通过',4:'已发布',5:'已驳回' }; return m[s]||'未知' }
</script>

<style scoped>
.knowledge-edit-page { max-width: 800px; margin: 0 auto; padding: 20px 24px; }
.page-header { display: flex; align-items: center; gap: 16px; margin-bottom: 20px; }
.page-header h1 { font-size: 20px; font-weight: 600; color: var(--text-primary); }
.btn-back { padding: 6px 12px; background: var(--bg-elevated); border: 1px solid var(--border-default); color: var(--text-secondary); border-radius: 4px; cursor: pointer; font-size: 13px; }
.form-card { background: var(--bg-elevated); border: 1px solid var(--border-default); border-radius: 8px; padding: 24px; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; margin-bottom: 4px; font-size: 14px; color: var(--text-secondary); }
.required { color: var(--tag-rejected-text); }
.form-group input, .form-group textarea, .form-select { width: 100%; padding: 8px 10px; background: var(--bg-subtle); border: 1px solid var(--border-default); border-radius: 4px; color: var(--text-primary); font-size: 14px; resize: vertical; }
.form-group input:focus, .form-group textarea:focus, .form-select:focus { outline: none; border-color: var(--accent); }
.form-group input:disabled, .form-group textarea:disabled { opacity: 0.6; }
.form-select { cursor: pointer; }
.tags-input { display: flex; flex-wrap: wrap; gap: 6px; padding: 6px; background: var(--bg-subtle); border: 1px solid var(--border-default); border-radius: 4px; min-height: 36px; align-items: center; }
.tag { display: inline-flex; align-items: center; gap: 4px; padding: 2px 8px; background: var(--accent); color: #fff; border-radius: 3px; font-size: 12px; }
.tag-remove { background: none; border: none; color: #fff; cursor: pointer; font-size: 14px; padding: 0 2px; }
.tag-input { border: none !important; background: none !important; flex: 1; min-width: 120px; padding: 4px !important; font-size: 13px !important; }
.form-actions { display: flex; gap: 10px; margin-top: 20px; }
.btn-save { padding: 10px 20px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-submit { padding: 10px 20px; background: var(--btn-warning-bg); color: var(--btn-warning-text); border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-cancel { padding: 10px 20px; background: var(--bg-elevated); border: 1px solid var(--border-default); color: var(--text-secondary); border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-approve { padding: 10px 20px; background: var(--btn-success-bg); color: var(--btn-success-text); border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-reject { padding: 10px 20px; background: var(--btn-danger-bg); color: var(--btn-danger-text); border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-publish { padding: 10px 20px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-disable { padding: 10px 20px; background: var(--btn-danger-bg); color: var(--btn-danger-text); border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-retry { padding: 10px 20px; background: var(--btn-warning-bg); color: var(--btn-warning-text); border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.review-section { margin-top: 20px; padding-top: 20px; border-top: 1px solid var(--border-default); }
.review-section h3 { font-size: 16px; color: var(--text-primary); margin-bottom: 12px; }
.review-actions { display: flex; gap: 10px; margin-top: 12px; }
.chunk-info { margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--border-default); }
.chunk-info h4 { font-size: 14px; color: var(--text-primary); margin-bottom: 8px; }
.chunk-item { display: flex; align-items: center; gap: 8px; padding: 6px 0; font-size: 12px; color: var(--text-secondary); }
.chunk-content { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sync-error { color: var(--tag-rejected-text); }
.sync-badge { font-size: 11px; padding: 1px 6px; border-radius: 3px; }
.sync-badge.synced { background: var(--sync-synced-bg); color: var(--sync-synced-text); }
.sync-badge.pending { background: var(--sync-pending-bg); color: var(--sync-pending-text); }
.sync-badge.failed { background: var(--sync-failed-bg); color: var(--sync-failed-text); }
.sync-badge.disabled { background: var(--sync-disabled-bg); color: var(--sync-disabled-text); }
.status-tag { font-size: 12px; padding: 2px 8px; border-radius: 3px; }
.status-tag.draft { background: var(--tag-draft-bg); color: var(--tag-draft-text); }
.status-tag.pending { background: var(--tag-pending-bg); color: var(--tag-pending-text); }
.status-tag.approved { background: var(--tag-approved-bg); color: var(--tag-approved-text); }
.status-tag.published { background: var(--tag-published-bg); color: var(--tag-published-text); }
.status-tag.rejected { background: var(--tag-rejected-bg); color: var(--tag-rejected-text); }
.status-tag.disabled { background: var(--tag-disabled-bg); color: var(--tag-disabled-text); }
</style>
