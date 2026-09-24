<template>
  <div class="live-layout">
    <!-- 顶部操作栏 -->
    <div class="live-toolbar">
      <div class="toolbar-left">
        <span class="toolbar-title">实时视频监控中心</span>
        <el-tag size="small" type="success">当前已连接 {{ cameras.length }} 路摄像头</el-tag>
      </div>

      <div class="toolbar-right">
        <!-- 宫格布局切换 (1, 4, 9, 16, 25, 36, 64) -->
        <span class="control-label">宫格布局：</span>
        <el-radio-group v-model="layoutCount" size="small" @change="handleLayoutChange">
          <el-radio-button :label="1">1</el-radio-button>
          <el-radio-button :label="4">4</el-radio-button>
          <el-radio-button :label="9">9</el-radio-button>
          <el-radio-button :label="16">16</el-radio-button>
          <el-radio-button :label="25">25</el-radio-button>
          <el-radio-button :label="36">36</el-radio-button>
          <el-radio-button :label="64">64</el-radio-button>
        </el-radio-group>

        <el-divider direction="vertical" />
        <el-button size="small" @click="refreshAllStreams">
          <el-icon><Refresh /></el-icon>
          刷新流
        </el-button>
        <el-button size="small" @click="toggleFullScreen">
          <el-icon><FullScreen /></el-icon>
          全屏模式
        </el-button>
      </div>
    </div>

    <!-- 主展示区：左侧通道快速切换，右侧多宫格 -->
    <div class="live-body" ref="fullScreenTarget">
      <!-- 左侧通道快速切换 -->
      <div class="channel-sidebar" v-if="!isFullScreen">
        <div class="sidebar-title">摄像头列表 (点击载入)</div>
        <div class="cam-list">
          <div
            v-for="cam in cameras"
            :key="cam.id"
            class="cam-item"
            :class="{ active: selectedCamId === cam.id }"
            @click="selectCamera(cam)"
          >
            <div class="cam-item-main">
              <span class="dot" :class="cam.is_online ? 'online' : 'offline'"></span>
              <span class="name">{{ cam.name }}</span>
            </div>
            <span v-if="cam.is_recording" class="rec-badge mini"><span class="rec-dot"></span> REC</span>
          </div>
          <div v-if="cameras.length === 0" class="empty-text">
            暂无设备，请先在设备管理添加
          </div>
        </div>
      </div>

      <!-- 视频宫格网格 -->
      <div class="grid-container" :style="gridStyle">
        <div
          v-for="(slot, idx) in slots"
          :key="idx"
          class="grid-cell"
          :class="{ selected: activeSlotIndex === idx }"
          @click="activeSlotIndex = idx"
        >
          <!-- 单元格顶栏 -->
          <div class="cell-header">
            <span class="cell-title">
              #{{ idx + 1 }} {{ slot.camera ? slot.camera.name : '空闲窗口' }}
            </span>
            <div class="cell-actions" v-if="slot.camera">
              <!-- 红色 REC 录像中徽章 (第7.4节需求) -->
              <span v-if="slot.camera.is_recording" class="rec-badge">
                <span class="rec-dot"></span> REC
              </span>
              <!-- 主/子码流切换 -->
              <el-tag
                size="small"
                :type="slot.streamType === 'main' ? 'danger' : 'info'"
                class="stream-tag"
                @click.stop="toggleStreamType(slot)"
              >
                {{ slot.streamType === 'main' ? '主码流(高清)' : '子码流(流畅)' }}
              </el-tag>
              <el-button
                circle
                size="small"
                type="danger"
                icon="Close"
                @click.stop="removeSlotCamera(idx)"
              />
            </div>
          </div>

          <!-- 视频播放画布区域 -->
          <div class="cell-player">
            <!-- 真正可播放的低延迟视频流容器 -->
            <div v-if="slot.camera" class="video-wrapper">
              <iframe
                v-if="slot.playUrl"
                :src="slot.playUrl"
                class="video-iframe"
                allow="autoplay; fullscreen"
                frameborder="0"
              ></iframe>
              <div v-else class="video-loading">
                <el-icon class="is-loading" :size="28" color="#3b82f6"><Loading /></el-icon>
                <span>正在建立视频流会话...</span>
              </div>
            </div>
            <div v-else class="cell-placeholder" @click="assignSelectedToSlot(idx)">
              <el-icon :size="32" color="#334155"><Plus /></el-icon>
              <span>点击载入选中摄像头</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

interface Slot {
  camera: any | null
  streamType: 'main' | 'sub'
  playUrl?: string
}

const cameras = ref<any[]>([])
const layoutCount = ref<number>(4)
const activeSlotIndex = ref<number>(0)
const selectedCamId = ref<number | null>(null)
const isFullScreen = ref(false)
const fullScreenTarget = ref<HTMLElement | null>(null)

const slots = ref<Slot[]>([])

const loadSlotStream = async (slot: Slot) => {
  if (!slot.camera) return
  try {
    const res: any = await api.requestStream(slot.camera.id, slot.streamType)
    if (res.code === 0 && res.data) {
      // 使用 MediaMTX 自带的高性能 WebRTC/HLS 播放视窗
      slot.playUrl = res.data.iframe_url
    }
  } catch (e: any) {
    console.error('加载视频流失败:', e)
  }
}

const initSlots = (count: number) => {
  const newSlots: Slot[] = []
  for (let i = 0; i < count; i++) {
    const streamType = count === 1 ? 'main' : 'sub'
    const cam = cameras.value[i] || null
    const s: Slot = {
      camera: cam,
      streamType,
    }
    if (cam) {
      loadSlotStream(s)
    }
    newSlots.push(s)
  }
  slots.value = newSlots
}

const handleLayoutChange = (count: number) => {
  initSlots(count)
}

const gridStyle = computed(() => {
  const count = layoutCount.value
  const cols = Math.ceil(Math.sqrt(count))
  return {
    display: 'grid',
    gridTemplateColumns: `repeat(${cols}, 1fr)`,
    gridTemplateRows: `repeat(${cols}, 1fr)`,
    gap: '6px',
  }
})

const selectCamera = (cam: any) => {
  selectedCamId.value = cam.id
  if (activeSlotIndex.value >= 0 && activeSlotIndex.value < slots.value.length) {
    const s = slots.value[activeSlotIndex.value]
    s.camera = cam
    loadSlotStream(s)
    if (activeSlotIndex.value < slots.value.length - 1) {
      activeSlotIndex.value++
    }
  }
}

const assignSelectedToSlot = (idx: number) => {
  if (selectedCamId.value) {
    const cam = cameras.value.find((c) => c.id === selectedCamId.value)
    if (cam) {
      slots.value[idx].camera = cam
      loadSlotStream(slots.value[idx])
      return
    }
  }
  ElMessage.info('请先在左侧列表中点击选择一个摄像头')
}

const removeSlotCamera = (idx: number) => {
  slots.value[idx].camera = null
  slots.value[idx].playUrl = undefined
}

const toggleStreamType = (slot: Slot) => {
  slot.streamType = slot.streamType === 'main' ? 'sub' : 'main'
  loadSlotStream(slot)
  ElMessage.success(`已切换至: ${slot.streamType === 'main' ? '主码流(高清)' : '子码流(流畅)'}`)
}

const refreshAllStreams = () => {
  slots.value.forEach((s) => {
    if (s.camera) {
      loadSlotStream(s)
    }
  })
  ElMessage.success('已刷新所有视频流')
}

const toggleFullScreen = () => {
  if (!document.fullscreenElement) {
    fullScreenTarget.value?.requestFullscreen()
    isFullScreen.value = true
  } else {
    document.exitFullscreen()
    isFullScreen.value = false
  }
}

onMounted(async () => {
  try {
    const res: any = await api.getCameras()
    if (res.code === 0) {
      cameras.value = res.data
    }
  } catch (e) {
    console.error(e)
  }
  initSlots(layoutCount.value)
})
</script>

<style scoped>
.live-layout {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.live-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: var(--panel-bg);
  padding: 8px 16px;
  border-radius: 6px;
  margin-bottom: 10px;
}

.toolbar-title {
  font-size: 14px;
  font-weight: bold;
  margin-right: 12px;
}

.control-label {
  font-size: 12px;
  color: #94a3b8;
  margin-right: 8px;
}

.live-body {
  display: flex;
  flex: 1;
  gap: 10px;
  min-height: calc(100vh - 140px);
}

.channel-sidebar {
  width: 220px;
  background-color: var(--panel-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 10px;
  display: flex;
  flex-direction: column;
}

.sidebar-title {
  font-size: 13px;
  font-weight: bold;
  color: #cbd5e1;
  margin-bottom: 10px;
}

.cam-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  overflow-y: auto;
}

.cam-item {
  padding: 8px 10px;
  background-color: #0f172a;
  border: 1px solid #1e293b;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  transition: all 0.2s;
}

.cam-item:hover, .cam-item.active {
  background-color: #1e293b;
  border-color: #3b82f6;
}

.cam-item-main {
  display: flex;
  align-items: center;
  gap: 6px;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.dot.online {
  background-color: #10b981;
}

.dot.offline {
  background-color: #ef4444;
}

.rec-badge.mini {
  font-size: 9px;
  padding: 1px 4px;
}

.grid-container {
  flex: 1;
  background-color: #020617;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 6px;
}

.grid-cell {
  background-color: #0b0f19;
  border: 1px solid #1e293b;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  transition: border-color 0.2s;
}

.grid-cell.selected {
  border-color: #3b82f6;
  box-shadow: 0 0 0 1px #3b82f6;
}

.cell-header {
  padding: 4px 8px;
  background-color: #141c2b;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  z-index: 2;
}

.cell-title {
  color: #cbd5e1;
  font-weight: 500;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cell-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.stream-tag {
  cursor: pointer;
  font-size: 10px;
}

.cell-player {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
  position: relative;
  background-color: #000;
}

.video-wrapper {
  width: 100%;
  height: 100%;
  position: relative;
}

.video-iframe {
  width: 100%;
  height: 100%;
  border: none;
  background-color: #000;
}

.video-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #94a3b8;
  font-size: 12px;
  gap: 8px;
}

.cell-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  color: #475569;
  font-size: 11px;
  cursor: pointer;
}

.empty-text {
  text-align: center;
  color: #64748b;
  font-size: 12px;
  padding: 20px 0;
}
</style>
