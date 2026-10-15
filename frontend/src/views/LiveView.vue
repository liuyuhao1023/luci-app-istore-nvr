<template>
  <div class="live-layout">
    <!-- 顶部操作栏 -->
    <div class="live-toolbar">
      <div class="toolbar-left">
        <span class="toolbar-title">实时视频监控中心</span>
        <el-tag size="small" type="success">已连接 {{ cameras.length }} 路</el-tag>
      </div>

      <div class="toolbar-right">
        <span class="control-label">宫格布局：</span>
        <el-radio-group v-model="layoutCount" size="small" @change="handleLayoutChange">
          <el-radio-button :value="1">1</el-radio-button>
          <el-radio-button :value="4">4</el-radio-button>
          <el-radio-button :value="9">9</el-radio-button>
          <el-radio-button :value="16">16</el-radio-button>
          <el-radio-button :value="25">25</el-radio-button>
          <el-radio-button :value="36">36</el-radio-button>
        </el-radio-group>

        <el-divider direction="vertical" class="toolbar-divider" />
        <div class="toolbar-actions">
          <el-button size="small" type="primary" plain @click="autoArrangeSlots">
            <el-icon><Grid /></el-icon>
            <span class="btn-text-desktop">一键顺序排列</span>
            <span class="btn-text-mobile">排布</span>
          </el-button>
          <el-button size="small" @click="refreshAllStreams">
            <el-icon><Refresh /></el-icon>
            <span class="btn-text-desktop">刷新全部</span>
            <span class="btn-text-mobile">刷新</span>
          </el-button>
          <el-button size="small" @click="toggleFullScreen">
            <el-icon><FullScreen /></el-icon>
            <span>全屏</span>
          </el-button>
        </div>
      </div>
    </div>

    <!-- 主展示区 -->
    <div class="live-body" ref="fullScreenTarget">
      <!-- 左侧摄像头列表 (手机端自动变为顶部横向滑动条) -->
      <div class="channel-sidebar" v-if="!isFullScreen">
        <div class="sidebar-header-row" @click="isMobile && (mobileSidebarCollapsed = !mobileSidebarCollapsed)">
          <div>
            <span class="sidebar-title">摄像头列表</span>
            <span class="sidebar-hint"> (点击载入窗口 #{{ activeSlotIndex + 1 }})</span>
          </div>
          <span v-if="isMobile" class="collapse-toggle">
            {{ mobileSidebarCollapsed ? '展开 ▼' : '收起 ▲' }}
          </span>
        </div>
        <div class="cam-list" v-show="!isMobile || !mobileSidebarCollapsed">
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
            <div class="cam-item-right">
              <span v-if="cam.video_codec" class="codec-badge" :class="cam.video_codec?.includes('265') ? 'h265' : 'h264'">
                {{ cam.video_codec?.includes('265') ? 'H.265' : 'H.264' }}
              </span>
              <span v-if="cam.is_recording" class="rec-badge mini"><span class="rec-dot"></span>REC</span>
            </div>
          </div>
          <div v-if="cameras.length === 0" class="empty-text">
            暂无设备，请先在设备管理添加
          </div>
        </div>
      </div>

      <!-- 视频宫格 -->
      <div class="grid-container" :style="gridStyle">
        <div
          v-for="(slot, idx) in slots"
          :key="`slot-${idx}`"
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
              <span v-if="slot.camera.is_recording" class="rec-badge">
                <span class="rec-dot"></span> REC
              </span>
              <el-tag
                size="small"
                :type="slot.streamType === 'main' ? 'danger' : 'info'"
                class="stream-tag"
                @click.stop="toggleStreamType(slot, idx)"
              >
                {{ slot.streamType === 'main' ? '主码流' : '子码流' }}
              </el-tag>
              <!-- 状态指示 -->
              <span class="status-dot" :class="slotStatus[idx]"></span>
              <el-button
                circle
                size="small"
                type="danger"
                icon="Close"
                @click.stop="removeSlotCamera(idx)"
              />
            </div>
          </div>

          <!-- 播放器区域 -->
          <div class="cell-player">
            <div v-if="slot.camera" class="video-wrapper">
              <!-- WebRTC 极低延迟播放器 (原生支持 H.264 与 H.265 硬解) -->
              <iframe
                v-if="slot.playUrl"
                :key="`${idx}-${slot.camera?.id}-${slot.streamType}`"
                :src="slot.playUrl"
                class="video-player"
                allow="autoplay; fullscreen"
                frameborder="0"
              ></iframe>
              <!-- loading 状态 -->
              <div v-else-if="slotStatus[idx] === 'loading'" class="video-loading">
                <el-icon class="is-loading" :size="28" color="#3b82f6"><Loading /></el-icon>
                <span>正在连接视频流...</span>
              </div>
              <!-- 错误状态 -->
              <div v-else-if="slotStatus[idx] === 'error'" class="video-error">
                <el-icon :size="32" color="#ef4444"><Warning /></el-icon>
                <span>视频流连接失败</span>
                <el-button size="small" type="primary" @click.stop="retrySlot(idx)">重试</el-button>
              </div>
              <!-- 初始化中 -->
              <div v-else class="video-loading">
                <el-icon class="is-loading" :size="28" color="#3b82f6"><Loading /></el-icon>
                <span>正在建立会话...</span>
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
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

// ─── 类型定义 ───────────────────────────────────────
interface Slot {
  camera: any | null
  streamType: 'main' | 'sub'
  playUrl?: string
}

interface SavedSlotConfig {
  camId: number | null
  streamType: 'main' | 'sub'
}

type SlotStatusType = 'init' | 'loading' | 'playing' | 'error'

// ─── 状态 ────────────────────────────────────────────
const cameras = ref<any[]>([])
const slots = ref<Slot[]>([])
const slotStatus = ref<Record<number, SlotStatusType>>({})
const activeSlotIndex = ref<number>(0)
const selectedCamId = ref<number | null>(null)
const isFullScreen = ref(false)
const fullScreenTarget = ref<HTMLElement | null>(null)
const isMobile = ref(window.innerWidth <= 768)
const mobileSidebarCollapsed = ref(false)
const updateIsMobile = () => {
  isMobile.value = window.innerWidth <= 768
}

// 从 localStorage 恢复布局
const savedLayout = Number(localStorage.getItem('nvr_live_layout'))
const layoutCount = ref<number>(
  savedLayout && [1, 4, 9, 16, 25, 36].includes(savedLayout) ? savedLayout : 4
)

// ─── 持久化 ──────────────────────────────────────────
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

// ─── 加载视频流（WebRTC 超低延迟版本）────────────────
const loadSlotStream = async (slot: Slot, idx: number) => {
  if (!slot.camera) return

  slot.playUrl = ''
  slotStatus.value[idx] = 'loading'

  try {
    const res: any = await api.requestStream(slot.camera.id, slot.streamType)
    if (res.code === 0 && res.data && res.data.iframe_url) {
      slot.playUrl = res.data.iframe_url
      slotStatus.value[idx] = 'playing'
    } else {
      slotStatus.value[idx] = 'error'
    }
  } catch (e: any) {
    console.error(`[slot ${idx}] 加载视频流失败:`, e)
    slotStatus.value[idx] = 'error'
  }
}

// ─── 重试 ────────────────────────────────────────────
const retrySlot = (idx: number) => {
  const slot = slots.value[idx]
  if (slot?.camera) {
    loadSlotStream(slot, idx)
  }
}

// ─── 切换布局 ─────────────────────────────────────────
const handleLayoutChange = (count: number) => {
  layoutCount.value = count
  localStorage.setItem('nvr_live_layout', String(count))

  const currentSlots = [...slots.value]
  const newSlots: Slot[] = []
  const usedCamIds = new Set<number>()

  for (let i = 0; i < count; i++) {
    if (i < currentSlots.length && currentSlots[i]) {
      const existing = currentSlots[i]
      if (existing.camera) usedCamIds.add(existing.camera.id)
      newSlots.push(existing)
    } else {
      const unusedCam = cameras.value.find((c) => !usedCamIds.has(c.id)) || null
      if (unusedCam) usedCamIds.add(unusedCam.id)
      const s: Slot = {
        camera: unusedCam,
        streamType: count === 1 ? 'main' : 'sub',
      }
      newSlots.push(s)
      if (unusedCam) {
        loadSlotStream(s, i)
      }
    }
  }

  // 保证 activeSlotIndex 在合法范围内
  if (activeSlotIndex.value >= count) {
    activeSlotIndex.value = 0
  }

  slots.value = newSlots
  saveSlotState()
}

// ─── 布局样式 ─────────────────────────────────────────
const gridStyle = computed(() => {
  const count = layoutCount.value
  if (isMobile.value) {
    const mobileCols = count === 1 ? 1 : 2
    return {
      display: 'grid',
      gridTemplateColumns: `repeat(${mobileCols}, 1fr)`,
      gridAutoRows: 'max-content',
      alignContent: 'start',
      gap: '4px',
    }
  }
  const cols = Math.ceil(Math.sqrt(count))
  return {
    display: 'grid',
    gridTemplateColumns: `repeat(${cols}, 1fr)`,
    gridTemplateRows: `repeat(${cols}, 1fr)`,
    gap: '2px',
  }
})

// ─── 选择摄像头（左侧列表点击）────────────────────────
const selectCamera = (cam: any) => {
  selectedCamId.value = cam.id
  if (activeSlotIndex.value < 0 || activeSlotIndex.value >= slots.value.length) {
    activeSlotIndex.value = 0
  }
  const idx = activeSlotIndex.value
  const s = slots.value[idx]
  if (s) {
    s.camera = cam
    s.streamType = layoutCount.value === 1 ? 'main' : (s.streamType || 'sub')
    loadSlotStream(s, idx)
    saveSlotState()

    // 仅在多宫格模式下自动跳到下一个空闲窗口；单画面模式下固定在当前窗口，便于用户连续点击切换查看
    if (slots.value.length > 1) {
      const nextEmpty = slots.value.findIndex((slot, i) => i > idx && !slot.camera)
      if (nextEmpty !== -1) {
        activeSlotIndex.value = nextEmpty
      } else if (idx < slots.value.length - 1) {
        activeSlotIndex.value++
      }
    }
  }
}

// ─── 点击空窗口分配 ───────────────────────────────────
const assignSelectedToSlot = (idx: number) => {
  activeSlotIndex.value = idx
  if (selectedCamId.value) {
    const cam = cameras.value.find((c) => c.id === selectedCamId.value)
    if (cam) {
      slots.value[idx].camera = cam
      slots.value[idx].streamType = layoutCount.value === 1 ? 'main' : 'sub'
      loadSlotStream(slots.value[idx], idx)
      saveSlotState()
      return
    }
  }
  ElMessage.info('请在左侧列表中点击选择要载入的摄像头')
}

// ─── 移除单个画面 ─────────────────────────────────────
const removeSlotCamera = (idx: number) => {
  slots.value[idx].camera = null
  slots.value[idx].playUrl = undefined
  slotStatus.value[idx] = 'init'
  saveSlotState()
}

// ─── 一键顺序排列 ─────────────────────────────────────
const autoArrangeSlots = () => {
  const count = layoutCount.value
  const newSlots: Slot[] = []
  const newStatus: Record<number, SlotStatusType> = {}

  for (let i = 0; i < count; i++) {
    const cam = cameras.value[i] || null
    const s: Slot = {
      camera: cam,
      streamType: 'main',
    }
    newSlots.push(s)
    newStatus[i] = cam ? 'init' : 'init'
  }

  slots.value = newSlots
  slotStatus.value = newStatus
  saveSlotState()

  // 逐个加载，避免同时发起太多请求
  newSlots.forEach((s, i) => {
    if (s.camera) {
      setTimeout(() => loadSlotStream(s, i), i * 200)
    }
  })

  ElMessage.success(`已按顺序分配前 ${Math.min(count, cameras.value.length)} 路摄像头`)
}

// ─── 主/子码流切换 ────────────────────────────────────
const toggleStreamType = (slot: Slot, idx: number) => {
  const nextType = slot.streamType === 'main' ? 'sub' : 'main'
  slot.streamType = nextType
  loadSlotStream(slot, idx)
  saveSlotState()
  ElMessage.success(`已切换为${nextType === 'main' ? '主码流' : '子码流'}`)
}

// ─── 刷新全部流 ───────────────────────────────────────
const refreshAllStreams = () => {
  slots.value.forEach((s, i) => {
    if (s.camera) {
      setTimeout(() => loadSlotStream(s, i), i * 150)
    }
  })
  ElMessage.success('正在刷新所有视频流...')
}

// ─── 全屏 ─────────────────────────────────────────────
const toggleFullScreen = () => {
  if (!document.fullscreenElement) {
    fullScreenTarget.value?.requestFullscreen()
    isFullScreen.value = true
  } else {
    document.exitFullscreen()
    isFullScreen.value = false
  }
}
document.addEventListener('fullscreenchange', () => {
  if (!document.fullscreenElement) isFullScreen.value = false
})

// ─── 初始化 ───────────────────────────────────────────
onMounted(async () => {
  window.addEventListener('resize', updateIsMobile)

  // 拉取摄像头列表
  try {
    const res: any = await api.getCameras()
    if (res.code === 0) {
      cameras.value = res.data || []
    }
  } catch (e) {
    console.error('获取摄像头列表失败:', e)
  }

  // 从 localStorage 恢复槽位配置
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
    let streamType: 'main' | 'sub' = 'main'

    // 优先恢复历史配置，严禁重复
    if (savedSlots[i] && savedSlots[i].camId) {
      const candidate = cameras.value.find((c) => c.id === savedSlots[i].camId)
      if (candidate && !usedCamIds.has(candidate.id)) {
        cam = candidate
        streamType = savedSlots[i].streamType || 'main'
      }
    }

    // 没有历史配置则从空闲摄像头中补充
    if (!cam) {
      const unused = cameras.value.find((c) => !usedCamIds.has(c.id))
      if (unused) {
        cam = unused
        streamType = 'main'
      }
    }

    if (cam) usedCamIds.add(cam.id)

    const s: Slot = { camera: cam, streamType }
    newSlots.push(s)
  }

  slots.value = newSlots
  saveSlotState()

  // 错开加载，避免并发请求过多
  newSlots.forEach((s, i) => {
    if (s.camera) {
      setTimeout(() => loadSlotStream(s, i), i * 300)
    }
  })
})

// ─── 销毁 ─────────────────────────────────────────────
onBeforeUnmount(() => {
  window.removeEventListener('resize', updateIsMobile)
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
  margin-bottom: 8px;
  flex-shrink: 0;
  flex-wrap: wrap;
  gap: 8px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-title {
  font-size: 14px;
  font-weight: bold;
}

.control-label {
  font-size: 12px;
  color: #94a3b8;
  margin-right: 4px;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-text-mobile {
  display: none;
}

.live-body {
  display: flex;
  flex: 1;
  gap: 8px;
  min-height: 0;
  overflow: hidden;
}

/* ── 侧边栏 ── */
.channel-sidebar {
  width: 200px;
  flex-shrink: 0;
  background-color: var(--panel-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 10px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.sidebar-title {
  font-size: 13px;
  font-weight: bold;
  color: #cbd5e1;
}

.sidebar-hint {
  font-size: 11px;
  color: #64748b;
}

.collapse-toggle {
  font-size: 11px;
  color: #3b82f6;
  cursor: pointer;
}

.cam-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
  flex: 1;
}

.cam-item {
  padding: 7px 8px;
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

.cam-item:hover,
.cam-item.active {
  background-color: #1e293b;
  border-color: #3b82f6;
}

.cam-item-main {
  display: flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
}

.cam-item-main .name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100px;
}

.cam-item-right {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot.online {
  background-color: #10b981;
}

.dot.offline {
  background-color: #ef4444;
}

.codec-badge {
  font-size: 9px;
  padding: 1px 3px;
  border-radius: 2px;
  font-weight: bold;
}

.codec-badge.h264 {
  background-color: #1d4ed8;
  color: #fff;
}

.codec-badge.h265 {
  background-color: #7c3aed;
  color: #fff;
}

/* ── 宫格 ── */
.grid-container {
  flex: 1;
  background-color: #000;
  border: 1px solid #111827;
  border-radius: 2px;
  padding: 1px;
  min-height: 0;
  overflow: hidden;
}

.grid-cell {
  background-color: #050811;
  border: 1px solid #0f172a;
  border-radius: 0;
  display: flex;
  overflow: hidden;
  position: relative;
  transition: all 0.15s;
}

.grid-cell.selected {
  outline: 2px solid #3b82f6;
  outline-offset: -2px;
  z-index: 5;
}

/* ── 单元格浮动 OSD 顶栏（专业监控 NVR 风格） ── */
.cell-header {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  padding: 4px 8px;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.85) 0%, rgba(0, 0, 0, 0.35) 65%, transparent 100%);
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  z-index: 10;
  opacity: 0.85;
  transition: opacity 0.2s;
  pointer-events: auto;
}

.grid-cell:hover .cell-header {
  opacity: 1;
}

.cell-title {
  color: #fff;
  font-weight: 500;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.9);
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cell-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.stream-tag {
  cursor: pointer;
  font-size: 10px;
}

/* 状态小圆点 */
.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.playing {
  background-color: #10b981;
}

.status-dot.loading,
.status-dot.init {
  background-color: #f59e0b;
  animation: blink 1s infinite;
}

.status-dot.error {
  background-color: #ef4444;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

/* ── 播放器区域（占满整个单元格） ── */
.cell-player {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  position: relative;
  background-color: #000;
  min-height: 0;
}

.video-wrapper {
  width: 100%;
  height: 100%;
  position: relative;
}

.video-player {
  width: 100%;
  height: 100%;
  object-fit: fill; /* 紧凑满屏无黑边，完全还原普通录像机监视屏输出效果 */
  background-color: #000;
  display: block;
}

.video-loading,
.video-error {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  font-size: 12px;
  gap: 8px;
  background-color: #000;
}

.cell-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  color: #475569;
  font-size: 11px;
  cursor: pointer;
  width: 100%;
  height: 100%;
  justify-content: center;
}

.cell-placeholder:hover {
  color: #64748b;
}

/* REC 徽章 */
.rec-badge {
  display: flex;
  align-items: center;
  gap: 3px;
  background-color: #dc2626;
  color: #fff;
  font-size: 9px;
  font-weight: bold;
  padding: 1px 4px;
  border-radius: 2px;
}

.rec-badge.mini {
  font-size: 9px;
}

.rec-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background-color: #fff;
  animation: blink 1s infinite;
}

.empty-text {
  text-align: center;
  color: #64748b;
  font-size: 12px;
  padding: 20px 0;
}

/* ── 移动端适配 (<= 768px) ── */
@media (max-width: 768px) {
  .live-toolbar {
    padding: 8px 10px;
    gap: 6px;
  }

  .toolbar-left {
    width: 100%;
    justify-content: space-between;
  }

  .toolbar-right {
    width: 100%;
    justify-content: space-between;
  }

  .control-label,
  .toolbar-divider,
  .btn-text-desktop {
    display: none !important;
  }

  .btn-text-mobile {
    display: inline;
  }

  .live-body {
    flex-direction: column;
    overflow: hidden;
  }

  .channel-sidebar {
    width: 100%;
    padding: 8px 10px;
    box-sizing: border-box;
    max-height: 92px;
  }

  .sidebar-header-row {
    margin-bottom: 6px;
  }

  .cam-list {
    flex-direction: row;
    overflow-x: auto;
    overflow-y: hidden;
    gap: 6px;
    padding-bottom: 2px;
  }

  .cam-item {
    flex-shrink: 0;
    padding: 5px 10px;
    gap: 6px;
  }

  .cam-item-main .name {
    max-width: 110px;
  }

  .grid-container {
    overflow-y: auto;
    padding: 2px;
  }

  .grid-cell {
    aspect-ratio: 16 / 9;
    min-height: 105px;
    border-radius: 4px;
  }

  .cell-title {
    max-width: 90px;
    font-size: 10px;
  }
}
</style>

