<template>
  <el-container class="app-layout">
    <!-- 顶部导航 -->
    <el-header height="56px" class="header-bar">
      <div class="header-left">
        <el-icon :size="24" color="#3b82f6"><VideoCameraFilled /></el-icon>
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
            active-text="已开启"
            inactive-text="已关闭"
            :loading="recordSwitchLoading"
            @change="handleGlobalRecordToggle"
          />
        </div>

        <el-divider direction="vertical" />
        <div class="status-summary">
          <span class="rec-badge" v-if="globalRecord">
            <span class="rec-dot"></span>
            REC 录像服务正常
          </span>
          <el-tag v-else type="warning" size="small">录像已全局暂停</el-tag>
        </div>
      </div>
    </el-header>

    <el-container>
      <!-- 左侧主菜单 -->
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
}

.header-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
  background-color: var(--panel-bg);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo-title {
  font-size: 16px;
  font-weight: bold;
  letter-spacing: 0.5px;
  color: #f8fafc;
}

.node-badge {
  background-color: #1e293b;
  border-color: #334155;
  color: #94a3b8;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
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
</style>
