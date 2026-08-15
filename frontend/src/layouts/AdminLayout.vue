<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { Calendar, House, List, Setting, SwitchButton } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

function logout() {
  auth.signOut()
  router.push('/login')
}
</script>

<template>
  <div class="admin-shell">
    <aside class="sidebar">
      <div class="brand"><span class="brand-mark">P</span><span>Personal Assistant</span></div>
      <el-menu :default-active="route.path" router class="nav-menu">
        <el-menu-item index="/"><el-icon><House /></el-icon><span>工作台</span></el-menu-item>
        <el-menu-item index="/calendar"><el-icon><Calendar /></el-icon><span>日历</span></el-menu-item>
        <el-menu-item index="/tasks"><el-icon><List /></el-icon><span>任务清单</span></el-menu-item>
        <el-menu-item index="/settings"><el-icon><Setting /></el-icon><span>系统设置</span></el-menu-item>
      </el-menu>
      <button class="logout-button" type="button" @click="logout">
        <el-icon><SwitchButton /></el-icon><span>退出登录</span>
      </button>
    </aside>
    <main class="main-area">
      <section class="page-content"><router-view /></section>
    </main>
  </div>
</template>
