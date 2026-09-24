<template>
  <div class="overview-container">
    <!-- 顶部状态卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-header">
            <span class="stat-title">摄像头管理</span>
            <el-icon color="#3b82f6" :size="20"><VideoCamera /></el-icon>
          </div>
          <div class="stat-value">{{ stats.online_cameras }} <span class="stat-sub">/ {{ stats.total_cameras }} 在线</span></div>
          <div class="stat-footer">
            <span class="online-tag">● 在线: {{ stats.online_cameras }}</span>
            <span class="offline-tag">● 离线: {{ stats.total_cameras - stats.online_cameras }}</span>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-header">
            <span class="stat-title">录像任务</span>
            <el-icon color="#ef4444" :size="20"><Film /></el-icon>
          </div>
          <div class="stat-value text-red">{{ stats.recording_cams }} <span class="stat-sub">路正在录制</span></div>
          <div class="stat-footer">
            <span v-if="stats.global_record" class="rec-badge"><span class="rec-dot"></span> 全局录像中</span>
            <el-tag v-else size="small" type="info">全局已暂停</el-tag>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-header">
            <span class="stat-title">软路由硬件负载</span>
            <el-icon color="#10b981" :size="20"><Cpu /></el-icon>
          </div>
          <div class="stat-value">{{ stats.cpu_load.toFixed(2) }} <span class="stat-sub">CPU 负载</span></div>
          <div class="stat-footer">
            <span>内存已用: {{ stats.mem_usage_pct.toFixed(1) }}% ({{ stats.mem_used_mb }}MB)</span>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-header">
            <span class="stat-title">存储数据盘</span>
            <el-icon color="#f59e0b" :size="20"><Coin /></el-icon>
          </div>
          <div class="stat-value">{{ stats.disk_free_gb.toFixed(1) }} <span class="stat-sub">GB 可用</span></div>
          <div class="stat-footer">
            <span>总计 {{ stats.disk_total_gb.toFixed(1) }} GB (已用 {{ stats.disk_usage_pct.toFixed(1) }}%)</span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 快捷预览与近期状态 -->
    <el-row :gutter="16" class="mt-16">
      <el-col :span="16">
        <el-card shadow="never">
          <template #header>
            <div class="card-header-flex">
              <span>快捷监控概览 (前 4 路)</span>
              <el-button type="primary" link @click="$router.push('/live')">进入完整多画面宫格 &gt;</el-button>
            </div>
          </template>
          <div class="quick-grid">
            <div v-for="cam in quickCameras" :key="cam.id" class="quick-item">
              <div class="quick-header">
                <span class="cam-title">{{ cam.name }} ({{ cam.ip }})</span>
                <span v-if="cam.is_recording" class="rec-badge"><span class="rec-dot"></span> REC</span>
              </div>
              <div class="screen-box">
                <el-icon :size="48" color="#475569"><VideoCamera /></el-icon>
                <div class="screen-tip">实时子码流传输就绪</div>
              </div>
            </div>
            <div v-if="quickCameras.length === 0" class="empty-tip">
              暂未添加摄像头，请前往【摄像头与网段接入】添加或扫描设备
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card shadow="never">
          <template #header>
            <span>系统信息与存储策略</span>
          </template>
          <div class="info-list">
            <div class="info-item">
              <span class="info-k">设备型号：</span>
              <span class="info-v">Intel J3455 (4核心 x86_64)</span>
            </div>
            <div class="info-item">
              <span class="info-k">网卡规格：</span>
              <span class="info-v">双 Intel I226-IT 2.5Gbps</span>
            </div>
            <div class="info-item">
              <span class="info-k">网络存储：</span>
              <span class="info-v">内核 SMB/CIFS 模块已加载</span>
            </div>
            <div class="info-item">
              <span class="info-k">码流录制策略：</span>
              <span class="info-v">原始码流直通复制 (零重编码)</span>
            </div>
            <div class="info-item">
              <span class="info-k">防误写安全机制：</span>
              <span class="info-v text-green">已启用挂载点熔断保护</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import api from '../api'

const stats = ref({
  os: 'linux',
  arch: 'amd64',
  cpu_load: 0.25,
  mem_total_mb: 7977,
  mem_used_mb: 298,
  mem_free_mb: 7551,
  mem_usage_pct: 3.7,
  disk_total_gb: 114.5,
  disk_free_gb: 108.6,
  disk_usage_pct: 2.1,
  total_cameras: 0,
  online_cameras: 0,
  recording_cams: 0,
  global_record: true,
})

const quickCameras = ref<any[]>([])
let pollTimer: any = null

const loadStats = async () => {
  try {
    const res: any = await api.getSystemStats()
    if (res.code === 0) {
      stats.value = res.data
    }
  } catch (e) {
    console.error(e)
  }
}

const loadCameras = async () => {
  try {
    const res: any = await api.getCameras()
    if (res.code === 0) {
      quickCameras.value = res.data.slice(0, 4)
    }
  } catch (e) {
    console.error(e)
  }
}

onMounted(() => {
  loadStats()
  loadCameras()
  pollTimer = setInterval(() => {
    loadStats()
  }, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.stats-row {
  margin-bottom: 8px;
}

.stat-card {
  border-radius: 8px;
}

.stat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-title {
  font-size: 13px;
  color: #94a3b8;
}

.stat-value {
  font-size: 26px;
  font-weight: bold;
  margin: 10px 0 6px 0;
  color: #f1f5f9;
}

.text-red {
  color: #f87171;
}

.text-green {
  color: #34d399;
}

.stat-sub {
  font-size: 13px;
  color: #64748b;
  font-weight: normal;
}

.stat-footer {
  font-size: 12px;
  color: #94a3b8;
  display: flex;
  gap: 12px;
}

.online-tag {
  color: #10b981;
}

.offline-tag {
  color: #ef4444;
}

.mt-16 {
  margin-top: 16px;
}

.card-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.quick-item {
  background-color: #0f172a;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  overflow: hidden;
}

.quick-header {
  padding: 8px 12px;
  background-color: #1e293b;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
}

.screen-box {
  height: 140px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  background-color: #020617;
  color: #475569;
}

.screen-tip {
  font-size: 11px;
  margin-top: 6px;
}

.empty-tip {
  grid-column: span 2;
  text-align: center;
  padding: 40px;
  color: #64748b;
  font-size: 13px;
}

.info-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  border-bottom: 1px solid #1e293b;
  padding-bottom: 8px;
}

.info-k {
  color: #94a3b8;
}

.info-v {
  color: #e2e8f0;
  font-weight: 500;
}
</style>
