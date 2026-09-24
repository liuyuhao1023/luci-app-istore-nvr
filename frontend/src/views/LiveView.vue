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
        <el-button size="small" type="primary" plain @click="autoArrangeSlots">
          <el-icon><Grid /></el-icon>
          一键顺序排列
        </el-button>
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
          :key="slot.camera ? `slot-${idx}-${slot.camera.id}-${slot.streamType}` : `slot-empty-${idx}`"
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
                :key="slot.playUrl"
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

interface SavedSlotConfig {
  camId: number | null
  streamType: 'main' | 'sub'
}

const cameras = ref<any[]>([])

// 1. 从 localStorage 读取上次退出的宫格模式，默认 4 宫格
const savedLayout = Number(localStorage.getItem('nvr_live_layout'))
const layoutCount = ref<number>(
  savedLayout && [1, 4, 9, 16, 25, 36, 64].includes(savedLayout) ? savedLayout : 4
)

const activeSlotIndex = ref<number>(0)
const selectedCamId = ref<number | null>(null)
const isFullScreen = ref(false)
const fullScreenTarget = ref<HTMLElement | null>(null)

const slots = ref<Slot[]>([])

// 持久化当前槽位配置
const saveSlotState = () => {
  try {
    const state: SavedSlotConfig[] = slots.value.map((s) => ({
      camId: s.camera ? s.camera.id : null,
      streamType: s.streamType,
    }))
    localStorage.setItem('nvr_live_slots', JSON.stringify(state))
  } catch (e) {
    console.error('保存槽位状态失败:', e)
  }
}

// 异步载入视频流
const loadSlotStream = async (slot: Slot) => {
  if (!slot.camera) return
  try {
    slot.playUrl = '' // 重置为空以触发 loading 态与 iframe 销毁
    const res: any = await api.requestStream(slot.camera.id, slot.streamType)
    if (res.code === 0 && res.data) {
      slot.playUrl = res.data.iframe_url
    }
  } catch (e: any) {
    console.error('加载视频流失败:', e)
  }
}

// 切换宫格布局 (保留已有槽位的视频流，绝不重复覆盖成同一摄像头)
const handleLayoutChange = (count: number) => {
  layoutCount.value = count
  localStorage.setItem('nvr_live_layout', String(count))

  const currentSlots = [...slots.value]
  const newSlots: Slot[] = []

  // 记录当前已经被分配在画面中的摄像头 ID，避免重复显示同一路画面
  const usedCamIds = new Set<number>()

  for (let i = 0; i < count; i++) {
    if (i < currentSlots.length && currentSlots[i]) {
      const existing = currentSlots[i]
      if (existing.camera) {
        usedCamIds.add(existing.camera.id)
      }
      newSlots.push(existing)
    } else {
      // 新扩充的窗口：优先在可用摄像头列表中挑选尚未分配的摄像头
      const unusedCam = cameras.value.find((c) => !usedCamIds.has(c.id)) || null
      if (unusedCam) {
        usedCamIds.add(unusedCam.id)
      }
      const s: Slot = {
        camera: unusedCam,
        streamType: count === 1 ? 'main' : 'sub',
      }
      if (unusedCam) {
        loadSlotStream(s)
      }
      newSlots.push(s)
    }
  }

  // 裁剪多余的窗口
  slots.value = newSlots
  saveSlotState()
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

// 点击左侧摄像头快速分配到当前选中窗口
const selectCamera = (cam: any) => {
  selectedCamId.value = cam.id
  if (activeSlotIndex.value >= 0 && activeSlotIndex.value < slots.value.length) {
    const s = slots.value[activeSlotIndex.value]
    s.camera = cam
    s.playUrl = ''
    loadSlotStream(s)
    saveSlotState()

    // 智能向后寻址：跳至下一个空闲窗口，防止连点导致后续窗口全部被替换为同一路摄像头
    const nextEmptyIdx = slots.value.findIndex(
      (slot, i) => i > activeSlotIndex.value && !slot.camera
    )
    if (nextEmptyIdx !== -1) {
      activeSlotIndex.value = nextEmptyIdx
    } else if (activeSlotIndex.value < slots.value.length - 1) {
      activeSlotIndex.value++
    }
  }
}

// 点击空窗口载入左侧选中的摄像头
const assignSelectedToSlot = (idx: number) => {
  activeSlotIndex.value = idx
  if (selectedCamId.value) {
    const cam = cameras.value.find((c) => c.id === selectedCamId.value)
    if (cam) {
      slots.value[idx].camera = cam
      slots.value[idx].playUrl = ''
      loadSlotStream(slots.value[idx])
      saveSlotState()
      return
    }
  }
  ElMessage.info('请在左侧列表中点击选择要载入的摄像头')
}

// 移除单个槽位画面
const removeSlotCamera = (idx: number) => {
  slots.value[idx].camera = null
  slots.value[idx].playUrl = undefined
  saveSlotState()
}

// 一键按顺序自动分配摄像头通道
const autoArrangeSlots = () => {
  const count = layoutCount.value
  const newSlots: Slot[] = []
  for (let i = 0; i < count; i++) {
    const cam = cameras.value[i] || null
    const streamType = count === 1 ? 'main' : 'sub'
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
  saveSlotState()
  ElMessage.success(`已按顺序分配前 ${Math.min(count, cameras.value.length)} 路摄像头`)
}

// 主/子码流切换
const toggleStreamType = (slot: Slot) => {
  const nextType = slot.streamType === 'main' ? 'sub' : 'main'
  if (nextType === 'main' && slot.camera?.video_codec?.toLowerCase().includes('265')) {
    ElMessage.warning('提示: 该摄像头主码流为 4K/2K H.265 编码，部分浏览器硬件可能无法直接解码，建议优先使用子码流')
  }
  slot.streamType = nextType
  slot.playUrl = ''
  loadSlotStream(slot)
  saveSlotState()
  ElMessage.success(`已切换至: ${slot.streamType === 'main' ? '主码流(高清)' : '子码流(流畅)'}`)
}

// 刷新全部已加载画面的视频流
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
      cameras.value = res.data || []
    }
  } catch (e) {
    console.error(e)
  }

  // 恢复保存的槽位配置，并严格去重
  let savedSlots: SavedSlotConfig[] = []
  try {
    const raw = localStorage.getItem('nvr_live_slots')
    if (raw) savedSlots = JSON.parse(raw)
  } catch (e) {
    console.error('解析缓存槽位失败:', e)
  }

  const count = layoutCount.value
  const newSlots: Slot[] = []
  const usedCamIds = new Set<number>()

  for (let i = 0; i < count; i++) {
    let cam = null
    let streamType: 'main' | 'sub' = count === 1 ? 'main' : 'sub'

    // 优先从历史存储中恢复，但严禁重复使用同一摄像头
    if (savedSlots[i] && savedSlots[i].camId) {
      const candidate = cameras.value.find((c) => c.id === savedSlots[i].camId)
      if (candidate && !usedCamIds.has(candidate.id)) {
        cam = candidate
        // 多画面默认子码流
        streamType = count === 1 ? 'main' : (savedSlots[i].streamType || 'sub')
      }
    }

    // 若未分配或已被去重，则从剩余未使用的摄像头中选一个补充
    if (!cam) {
      const unused = cameras.value.find((c) => !usedCamIds.has(c.id))
      if (unused) {
        cam = unused
        streamType = count === 1 ? 'main' : 'sub'
      }
    }

    if (cam) {
      usedCamIds.add(cam.id)
    }

    const s: Slot = { camera: cam, streamType }
    if (cam) {
      loadSlotStream(s)
    }
    newSlots.push(s)
  }

  slots.value = newSlots
  saveSlotState()
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
