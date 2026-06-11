<template>
  <div class="knowledge-page">
    <div class="knowledge-layout">
      <!-- 左侧：知识库列表 -->
      <aside class="kb-sidebar">
        <div class="sidebar-header">
          <h2>知识库</h2>
          <button class="btn-add-kb" @click="showKBDialog = true">+ 新建</button>
        </div>
        <ul class="kb-list">
          <li
            v-for="kb in kbList"
            :key="kb.id"
            :class="['kb-item', { active: selectedKB?.id === kb.id }]"
            @click="selectKB(kb)"
          >
            <span class="kb-name">{{ kb.name }}</span>
          </li>
        </ul>
        <div v-if="kbList.length === 0" class="empty-hint">暂无知识库</div>
      </aside>

      <!-- 右侧：文章列表 -->
      <main class="articles-main">
        <div class="articles-header">
          <h2>{{ selectedKB ? selectedKB.name + ' — 文章' : '请选择知识库' }}</h2>
          <div class="header-actions">
            <select v-model="statusFilter" @change="fetchArticles" class="status-filter">
              <option :value="-1">全部状态</option>
              <option :value="1">草稿</option>
              <option :value="2">待审核</option>
              <option :value="3">已通过</option>
              <option :value="4">已发布</option>
              <option :value="5">已驳回</option>
              <option :value="0">已停用</option>
            </select>
            <button v-if="selectedKB" class="btn-add-article" @click="goCreate">+ 新建文章</button>
          </div>
        </div>

        <div v-if="!selectedKB" class="empty-hint">请在左侧选择一个知识库</div>

        <table v-else-if="articles.length > 0" class="article-table">
          <thead>
            <tr><th>问题</th><th>分类</th><th>状态</th><th>同步</th><th>更新时间</th><th>操作</th></tr>
          </thead>
          <tbody>
            <tr v-for="a in articles" :key="a.id">
              <td class="question-cell" @click="goEdit(a.id)">{{ a.question }}</td>
              <td>{{ a.category || '-' }}</td>
              <td><span :class="['status-tag', statusClass(a.status)]">{{ statusText(a.status) }}</span></td>
              <td><span :class="['sync-badge', syncClass(a.sync_status)]">{{ syncText(a.sync_status) }}</span></td>
              <td>{{ formatTime(a.updated_at) }}</td>
              <td class="action-cell">
                <button v-if="a.status===1" class="btn-sm btn-primary" @click="handleSubmitReview(a.id)">提交审核</button>
                <button v-if="a.status===2" class="btn-sm btn-primary" @click="goEdit(a.id)">审核</button>
                <button v-if="a.status===3" class="btn-sm btn-success" @click="handlePublish(a.id)">发布</button>
                <button v-if="a.status===4" class="btn-sm btn-warning" @click="handleDisable(a.id)">停用</button>
                <button v-if="a.status===0" class="btn-sm btn-success" @click="handleEnable(a.id)">恢复</button>
                <button v-if="a.sync_status==='failed'" class="btn-sm btn-warning" @click="handleRetrySync(a.id)">重试同步</button>
                <button v-if="a.status===1||a.status===5" class="btn-sm btn-default" @click="goEdit(a.id)">编辑</button>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-else-if="selectedKB && articles.length === 0" class="empty-hint">暂无文章</div>

        <Pagination v-if="selectedKB && total>0" :total="total" v-model:current-page="currentPage" v-model:page-size="pageSize" @update:current-page="fetchArticles" @update:page-size="fetchArticles" />
      </main>
    </div>

    <!-- 新建知识库对话框 -->
    <div v-if="showKBDialog" class="dialog-overlay" @click.self="showKBDialog=false">
      <div class="dialog">
        <h3>新建知识库</h3>
        <div class="form-group"><label>名称</label><input v-model="newKB.name" type="text" /></div>
        <div class="form-group"><label>描述</label><textarea v-model="newKB.description" rows="3" /></div>
        <div class="dialog-actions">
          <button class="btn-cancel" @click="showKBDialog=false">取消</button>
          <button class="btn-primary" @click="handleCreateKB" :disabled="!newKB.name">创建</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Pagination from '../../components/common/Pagination.vue'
import { listKnowledgeBases, createKnowledgeBase, listArticles, submitReview, publishArticle, disableArticle, enableArticle, retrySyncArticle } from '../../api/knowledge'

interface KB { id: number; name: string }
interface Article { id: number; question: string; category: string; status: number; sync_status: string; updated_at: string }

const router = useRouter()
const kbList = ref<KB[]>([])
const selectedKB = ref<KB | null>(null)
const articles = ref<Article[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)
const statusFilter = ref(-1)  // 默认显示全部状态
const showKBDialog = ref(false)
const newKB = ref({ name: '', description: '' })

onMounted(() => { fetchKBs() })

const fetchKBs = async () => {
  try { const res = await listKnowledgeBases(); kbList.value = (res.data as any).items || (res as any).items || [] } catch (e) { console.error(e) }
}
const selectKB = (kb: KB) => { selectedKB.value = kb; currentPage.value = 1; fetchArticles() }
const fetchArticles = async () => {
  if (!selectedKB.value) return
  try {
    const params: any = { page: currentPage.value, page_size: pageSize.value }
    if (statusFilter.value !== -1) params.status = statusFilter.value
    const res = await listArticles(selectedKB.value.id, params)
    const list = Array.isArray(res.data) ? res.data : (res.data.articles || [])
    articles.value = list; total.value = res.data.total || list.length || 0
  } catch (e) { console.error(e) }
}
const handleCreateKB = async () => {
  try { await createKnowledgeBase(newKB.value); showKBDialog.value = false; newKB.value = { name: '', description: '' }; await fetchKBs() } catch (e: any) { alert(e?.message || '创建失败') }
}
const goCreate = () => { router.push(`/admin/knowledge/new?kb_id=${selectedKB.value!.id}`) }
const goEdit = (id: number) => { router.push(`/admin/knowledge/${id}`) }
const handleSubmitReview = async (id: number) => { try { await submitReview(id); await fetchArticles() } catch (e: any) { alert(e?.message) } }
const handlePublish = async (id: number) => { try { await publishArticle(id); await fetchArticles() } catch (e: any) { alert(e?.message) } }
const handleDisable = async (id: number) => { if (!confirm('确定停用？')) return; try { await disableArticle(id); await fetchArticles() } catch (e: any) { alert(e?.message) } }
const handleEnable = async (id: number) => { try { await enableArticle(id); await fetchArticles() } catch (e: any) { alert(e?.message) } }
const handleRetrySync = async (id: number) => { try { await retrySyncArticle(id); await fetchArticles() } catch (e: any) { alert(e?.message) } }

const statusClass = (s: number) => { const m: Record<number,string> = { '-1':'disabled',0:'disabled',1:'draft',2:'pending',3:'approved',4:'published',5:'rejected' }; return m[s]||'' }
const statusText = (s: number) => { const m: Record<number,string> = { '-1':'已停用',0:'已停用',1:'草稿',2:'待审核',3:'已通过',4:'已发布',5:'已驳回' }; return m[s]||'未知' }
const syncClass = (s: string) => { const m: Record<string,string> = { synced:'synced',pending:'pending',failed:'failed',disabled:'disabled' }; return m[s]||'' }
const syncText = (s: string) => { const m: Record<string,string> = { synced:'已同步',pending:'待同步',failed:'失败',disabled:'已停用' }; return m[s]||s||'-' }
const formatTime = (t: string) => t ? new Date(t).toLocaleString('zh-CN') : '-'
</script>

<style scoped>
.knowledge-page { height: 100%; }
.knowledge-layout { display: flex; height: 100%; }
.kb-sidebar { width: 220px; min-width: 220px; border-right: 1px solid var(--border-default); padding: 16px; overflow-y: auto; }
.sidebar-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.sidebar-header h2 { font-size: 15px; font-weight: 600; color: var(--text-primary); }
.btn-add-kb { padding: 4px 10px; font-size: 12px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; }
.kb-list { list-style: none; padding: 0; margin: 0; }
.kb-item { padding: 10px 12px; border-radius: 6px; cursor: pointer; display: flex; justify-content: space-between; align-items: center; margin-bottom: 2px; color: var(--text-secondary); font-size: 13px; }
.kb-item:hover { background: var(--bg-subtle); }
.kb-item.active { background: var(--accent); color: #fff; }
.articles-main { flex: 1; padding: 20px 24px; overflow-y: auto; }
.articles-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.articles-header h2 { font-size: 18px; font-weight: 600; color: var(--text-primary); }
.header-actions { display: flex; gap: 10px; align-items: center; }
.status-filter { padding: 6px 10px; background: var(--bg-elevated); border: 1px solid var(--border-default); color: var(--text-primary); border-radius: 4px; font-size: 13px; }
.btn-add-article { padding: 8px 16px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; }
.article-table { width: 100%; border-collapse: collapse; }
.article-table th, .article-table td { padding: 10px 12px; text-align: left; border-bottom: 1px solid var(--border-default); font-size: 13px; color: var(--text-primary); }
.article-table th { font-weight: 600; color: var(--text-secondary); font-size: 12px; }
.question-cell { cursor: pointer; color: var(--accent); max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.question-cell:hover { text-decoration: underline; }
.action-cell { display: flex; gap: 4px; flex-wrap: wrap; }
.btn-sm { padding: 4px 10px; font-size: 12px; border: none; border-radius: 4px; cursor: pointer; white-space: nowrap; }
.btn-primary { background: var(--accent); color: #fff; }
.btn-success { background: var(--btn-success-bg); color: var(--btn-success-text); }
.btn-warning { background: var(--btn-warning-bg); color: var(--btn-warning-text); }
.btn-default { background: var(--bg-elevated); color: var(--text-secondary); border: 1px solid var(--border-default); }
.btn-cancel { padding: 8px 16px; background: var(--bg-elevated); color: var(--text-secondary); border: 1px solid var(--border-default); border-radius: 4px; cursor: pointer; }
.status-tag { font-size: 12px; padding: 2px 6px; border-radius: 3px; }
.status-tag.draft { background: var(--tag-draft-bg); color: var(--tag-draft-text); }
.status-tag.pending { background: var(--tag-pending-bg); color: var(--tag-pending-text); }
.status-tag.approved { background: var(--tag-approved-bg); color: var(--tag-approved-text); }
.status-tag.published { background: var(--tag-published-bg); color: var(--tag-published-text); }
.status-tag.rejected { background: var(--tag-rejected-bg); color: var(--tag-rejected-text); }
.status-tag.disabled { background: var(--tag-disabled-bg); color: var(--tag-disabled-text); }
.sync-badge { font-size: 12px; padding: 2px 6px; border-radius: 3px; }
.sync-badge.synced { background: var(--sync-synced-bg); color: var(--sync-synced-text); }
.sync-badge.pending { background: var(--sync-pending-bg); color: var(--sync-pending-text); }
.sync-badge.failed { background: var(--sync-failed-bg); color: var(--sync-failed-text); }
.sync-badge.disabled { background: var(--sync-disabled-bg); color: var(--sync-disabled-text); }
.dialog-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center; z-index: 100; }
.dialog { background: var(--bg-elevated); border: 1px solid var(--border-default); border-radius: 8px; padding: 24px; width: 400px; }
.dialog h3 { margin-bottom: 16px; font-size: 16px; color: var(--text-primary); }
.form-group { margin-bottom: 12px; }
.form-group label { display: block; margin-bottom: 4px; font-size: 13px; color: var(--text-secondary); }
.form-group input, .form-group textarea { width: 100%; padding: 8px 10px; background: var(--bg-subtle); border: 1px solid var(--border-default); border-radius: 4px; color: var(--text-primary); font-size: 13px; resize: vertical; }
.dialog-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px; }
.empty-hint { padding: 60px; text-align: center; color: var(--text-secondary); font-size: 14px; }
</style>
