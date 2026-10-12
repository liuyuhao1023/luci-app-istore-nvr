<template>
  <el-container class="app-layout">
    <!-- 顶部导航 -->
    <el-header height="56px" class="header-bar">
      <div class="header-left">
        <el-icon :size="22" color="#3b82f6"><VideoCameraFilled /></el-icon>
        <span class="logo-title">NVR 摄像头管理系统</span>
        <el-tag type="info" size="small" class="node-badge">软路由节点: {{ hostNode }}</el-tag>
      </div>

      <div class="header-right">
        <!-- 全局录像总开关 (第7.1节需求) -->
        <div class="global-record-control">
          <span class="switch-label">全局录像总开关：</span>
          <el-switch
            v-model="globalRecord"
            active-color="#ef4444"
            inactive-color="#475569"
            active-text="录像开"
            inactive-text="录像关"
            inline-prompt
            :loading="recordSwitchLoading"
            @change="handleGlobalRecordToggle"
          />
        </div>

        <el-divider direction="vertical" class="header-divider" />
        <div class="status-summary">
          <span class="rec-badge" v-if="globalRecord">
            <span class="rec-dot"></span>
            <span class="rec-text-full">REC 录像服务正常</span>
            <span class="rec-text-short">REC</span>
          </span>
          <el-tag v-else type="warning" size="small">
            <span class="rec-text-full">录像已全局暂停</span>
            <span class="rec-text-short">暂停</span>
          </el-tag>
        </div>
      </div>
    </el-header>

    <el-container class="body-container">
      <!-- 左侧主菜单 (桌面端显示) -->
      <el-aside width="220px" class="side-nav">
        <el-menu :default-active="$route.path" router class="nvr-menu">
          <el-menu-item-group title="监控中心">
            <el-menu-item index="/">
              <el-icon><DataBoard /></el-icon>
              <span>系统总览</span>
            </el-menu-item>
            <el-menu-item index="/live">
              <el-icon><VideoCamera /></el-icon>
              <span>实时视频预览</span>
            </el-menu-item>
          </el-menu-item-group>

          <el-menu-item-group title="设备与网络">
            <el-menu-item index="/cameras">
              <el-icon><Connection /></el-icon>
              <span>摄像头与网段接入</span>
            </el-menu-item>
          </el-menu-item-group>

          <el-menu-item-group title="录像与存储">
            <el-menu-item index="/playback">
              <el-icon><Film /></el-icon>
              <span>录像检索与回放</span>
            </el-menu-item>
            <el-menu-item index="/storage">
              <el-icon><Coin /></el-icon>
              <span>网络与本地存储 (SMB)</span>
            </el-menu-item>
          </el-menu-item-group>
        </el-menu>
      </el-aside>

      <!-- 主视图区域 -->
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>

    <!-- 移动端底部快捷导航栏 (<= 768px 显示) -->
    <nav class="mobile-bottom-nav">
      <router-link to="/" class="mobile-nav-item" :class="{ active: $route.path === '/' }">
        <el-icon :size="20"><DataBoard /></el-icon>
        <span>总览</span>
      </router-link>
      <router-link to="/live" class="mobile-nav-item" :class="{ active: $route.path === '/live' }">
        <el-icon :size="20"><VideoCamera /></el-icon>
        <span>实时监控</span>
      </router-link>
      <router-link to="/cameras" class="mobile-nav-item" :class="{ active: $route.path === '/cameras' }">
        <el-icon :size="20"><Connection /></el-icon>
        <span>摄像头</span>
      </router-link>
      <router-link to="/playback" class="mobile-nav-item" :class="{ active: $route.path === '/playback' }">
        <el-icon :size="20"><Film /></el-icon>
        <span>录像回放</span>
      </router-link>
      <router-link to="/storage" class="mobile-nav-item" :class="{ active: $route.path === '/storage' }">
        <el-icon :size="20"><Coin /></el-icon>
        <span>存储设置</span>
      </router-link>
    </nav>
  </el-container>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from './api'

const globalRecord = ref(true)
const recordSwitchLoading = ref(false)
const hostNode = ref(window.location.hostname || '软路由')

const loadGlobalSwitch = async () => {
  try {
    const res: any = await api.getGlobalRecord()
    if (res.code === 0) {
      globalRecord.value = res.enabled
    }
  } catch (e: any) {
    console.error('获取全局录像状态失败', e)
  }
}

const handleGlobalRecordToggle = (val: boolean) => {
  const actionText = val ? '开启' : '关闭'
  ElMessageBox.confirm(
    `确定要${actionText}【全局录像】功能吗？${val ? '开启后符合独立开关的摄像头将继续录制。' : '关闭后所有录像任务将立即正常收尾并停止写入，但不影响实时预览。'}`,
    '系统全局录像开关提示',
    {
      confirmButtonText: '确定变更',
      cancelButtonText: '取消',
      type: val ? 'warning' : 'info',
    }
  ).then(async () => {
    recordSwitchLoading.value = true
    try {
      await api.setGlobalRecord(val)
      ElMessage.success(`已成功${actionText}全局录像功能`)
    } catch (e: any) {
      globalRecord.value = !val
      ElMessage.error(e.message)
    } finally {
      recordSwitchLoading.value = false
    }
  }).catch(() => {
    globalRecord.value = !val
  })
}

onMounted(() => {
  loadGlobalSwitch()
})
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
  background-color: var(--bg-color);
  display: flex;
  flex-direction: column;
}

.header-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
  background-color: var(--panel-bg);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.logo-title {
  font-size: 16px;
  font-weight: bold;
  letter-spacing: 0.5px;
  color: #f8fafc;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.node-badge {
  background-color: #1e293b;
  border-color: #334155;
  color: #94a3b8;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-shrink: 0;
}

.global-record-control {
  display: flex;
  align-items: center;
  gap: 8px;
}

.switch-label {
  font-size: 13px;
  color: #cbd5e1;
}

.rec-text-short {
  display: none;
}

.body-container {
  flex: 1;
  min-height: 0;
}

.side-nav {
  background-color: #111827;
  border-right: 1px solid var(--border-color);
}

.main-content {
  padding: 18px;
  background-color: #0b0f17;
  overflow-y: auto;
  height: calc(100vh - 56px);
}

/* 移动端底部导航默认隐藏 */
.mobile-bottom-nav {
  display: none;
}

@media (max-width: 768px) {
  .header-bar {
    height: 48px !important;
    padding: 0 12px;
  }

  .logo-title {
    font-size: 14px;
  }

  .node-badge,
  .switch-label,
  .header-divider,
  .rec-text-full {
    display: none !important;
  }

  .rec-text-short {
    display: inline;
  }

  .header-right {
    gap: 8px;
  }

  .side-nav {
    display: none !important;
  }

  .main-content {
    padding: 10px 10px 70px 10px;
    height: calc(100vh - 48px);
  }

  .mobile-bottom-nav {
    display: flex;
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    height: 58px;
    background-color: #111827;
    border-top: 1px solid var(--border-color);
    z-index: 1000;
    justify-content: space-around;
    align-items: center;
    padding-bottom: env(safe-area-inset-bottom, 0px);
    box-shadow: 0 -2px 10px rgba(0, 0, 0, 0.4);
  }

  .mobile-nav-item {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 3px;
    color: #94a3b8;
    text-decoration: none;
    font-size: 11px;
    height: 100%;
    transition: color 0.15s, background-color 0.15s;
  }

  .mobile-nav-item.active {
    color: #3b82f6;
    font-weight: 600;
  }
}
</style>

