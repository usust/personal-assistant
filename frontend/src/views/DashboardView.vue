<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Calendar, Bell, Document, CircleCheck } from '@element-plus/icons-vue'
import { getHealth } from '@/api/system'

const summary = ref({ tasks: 0, notes: 0, reminders: 0 })
const apiStatus = ref('检查中')

onMounted(async () => {
  try {
    const health = await getHealth()
    apiStatus.value = health.status === 'ok' ? '运行正常' : '状态异常'
  } catch {
    apiStatus.value = '连接失败'
  }
})

const cards = [
  { key: 'tasks', label: '待办任务', icon: Calendar, tone: 'indigo' },
  { key: 'notes', label: '我的笔记', icon: Document, tone: 'amber' },
  { key: 'reminders', label: '近期提醒', icon: Bell, tone: 'rose' },
] as const
</script>

<template>
  <div class="page-heading"><div><p class="eyebrow">OVERVIEW</p><h1>工作台</h1></div><span class="date-chip">2026 · 管理中心</span></div>
  <div class="stat-grid">
    <article v-for="card in cards" :key="card.key" class="stat-card">
      <div :class="['stat-icon', card.tone]"><component :is="card.icon" /></div>
      <div><strong>{{ summary[card.key] }}</strong><p>{{ card.label }}</p></div>
      <span class="stat-link">查看 →</span>
    </article>
  </div>
  <div class="content-grid">
    <article class="panel welcome-panel">
      <p class="eyebrow">GET STARTED</p><h2>你的个人效率中心已就绪</h2><p>前端已与 Gin API 完成连接。接下来可以在当前基础上添加任务、日程、知识库或 AI 助手模块。</p>
      <div class="next-steps"><span>01 创建业务模型</span><span>02 添加 API 路由</span><span>03 构建 Vue 页面</span></div>
    </article>
    <article class="panel status-panel"><div class="status-icon"><CircleCheck /></div><div><p class="eyebrow">SYSTEM STATUS</p><h3>后端服务</h3><p>{{ apiStatus }}</p></div><span class="status-dot"></span></article>
  </div>
</template>
