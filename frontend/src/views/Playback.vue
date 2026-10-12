<template>
  <div class="playback-container">
    <div class="playback-toolbar">
      <div class="filter-group">
        <div class="filter-item">
          <span class="filter-label">选择摄像头：</span>
          <el-select v-model="selectedCamId" placeholder="选择摄像头" class="cam-select" @change="queryRecords">
            <el-option
              v-for="cam in cameras"
              :key="cam.id"
              :label="`${cam.name} (${cam.ip})`"
              :value="cam.id"
            />
          </el-select>
        </div>

        <div class="filter-item">
          <span class="filter-label">时间范围：</span>
          <el-date-picker
            v-model="dateRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            class="date-picker"
            @change="queryRecords"
          />
        </div>

        <el-button type="primary" icon="Search" class="search-btn" @click="queryRecords" :loading="loading">
          检索录像
        </el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mt-12">
      <!-- 播放视窗与时间轴 -->
      <el-col :xs="24" :sm="24" :md="16" class="pb-col">
        <el-card shadow="never" class="player-card">
          <div class="player-screen">
            <div v-if="currentPlayRecord" class="playing-state">
              <el-icon :size="50" color="#3b82f6"><VideoPlay /></el-icon>
              <div class="play-filename">{{ currentPlayRecord.file_path }}</div>
              <div class="play-meta">
                时长: {{ currentPlayRecord.duration_sec }} 秒 | 格式: {{ currentPlayRecord.format }} ({{ currentPlayRecord.video_codec }})
              </div>
            </div>
            <div v-else class="empty-player">
              <el-icon :size="48" color="#334155"><Film /></el-icon>
              <span>请在列表中点击选择要回放的录像片段</span>
            </div>
          </div>

          <!-- 时间轴模拟标尺 -->
          <div class="timeline-bar">
            <div class="timeline-header">
              <span>时间轴进度</span>
              <span class="timeline-scale">00:00 ------ 12:00 ------ 24:00</span>
            </div>
            <div class="timeline-track">
              <div
                v-for="r in recordings"
                :key="r.id"
                class="timeline-segment"
                :class="{ active: currentPlayRecord?.id === r.id }"
                @click="playRecord(r)"
              ></div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 录像片段列表 -->
      <el-col :xs="24" :sm="24" :md="8" class="pb-col">
        <el-card shadow="never">
          <template #header>
            <div class="card-header-flex">
              <span>录像切片列表 ({{ recordings.length }})</span>
            </div>
          </template>

          <div class="records-scroll-list">
            <div
              v-for="rec in recordings"
              :key="rec.id"
              class="rec-item"
              :class="{ selected: currentPlayRecord?.id === rec.id }"
              @click="playRecord(rec)"
            >
              <div class="rec-item-header">
                <span class="rec-time">{{ formatTime(rec.start_time) }} - {{ formatTime(rec.end_time) }}</span>
                <el-tag size="small" type="success">{{ rec.duration_sec }}s</el-tag>
              </div>
              <div class="rec-item-footer">
                <span class="rec-size">{{ formatMB(rec.size_bytes) }} MB</span>
                <el-button size="small" link type="primary" @click.stop="downloadRecord(rec)">下载片段</el-button>
              </div>
            </div>

            <div v-if="recordings.length === 0" class="empty-records">
              未检索到符合条件的录像切片
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const cameras = ref<any[]>([])
const selectedCamId = ref<number | null>(null)
const dateRange = ref<any>([])
const recordings = ref<any[]>([])
const currentPlayRecord = ref<any | null>(null)
const loading = ref(false)

const loadCameras = async () => {
  try {
    const res: any = await api.getCameras()
    if (res.code === 0) {
      cameras.value = res.data
      if (cameras.value.length > 0) {
        selectedCamId.value = cameras.value[0].id
        queryRecords()
      }
    }
  } catch (e) {
    console.error(e)
  }
}

const queryRecords = async () => {
  if (!selectedCamId.value) return
  loading.value = true
  try {
    const params: any = { camera_id: selectedCamId.value }
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_time = dateRange.value[0]
      params.end_time = dateRange.value[1]
    }
    const res: any = await api.queryRecordings(params)
    if (res.code === 0) {
      recordings.value = res.data
      if (recordings.value.length > 0) {
        currentPlayRecord.value = recordings.value[0]
      }
    }
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

const playRecord = (rec: any) => {
  currentPlayRecord.value = rec
}

const downloadRecord = (rec: any) => {
  window.open(`/api/record/download/${rec.id}`, '_blank')
}

const formatTime = (iso: string) => {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toTimeString().split(' ')[0]
}

const formatMB = (bytes: number) => {
  if (!bytes) return '0.0'
  return (bytes / (1024 * 1024)).toFixed(2)
}

onMounted(() => {
  loadCameras()
})
</script>

<style scoped>
.playback-toolbar {
  background-color: var(--panel-bg);
  padding: 12px 16px;
  border-radius: 6px;
}

.filter-group {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.filter-item {
  display: flex;
  align-items: center;
}

.filter-label {
  font-size: 13px;
  color: #94a3b8;
  margin-right: 8px;
  white-space: nowrap;
}

.cam-select {
  width: 200px;
}

.date-picker {
  width: 340px;
}

.mt-12 {
  margin-top: 12px;
}

.pb-col {
  margin-bottom: 12px;
}

.player-card {
  border-radius: 6px;
}

.player-screen {
  height: 420px;
  background-color: #000;
  display: flex;
  justify-content: center;
  align-items: center;
  border-radius: 4px;
  padding: 12px;
  box-sizing: border-box;
  text-align: center;
}

.playing-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  color: #cbd5e1;
  max-width: 100%;
}

.play-filename {
  font-family: monospace;
  font-size: 12px;
  color: #60a5fa;
  word-break: break-all;
}

.play-meta {
  font-size: 12px;
  color: #94a3b8;
}

.empty-player {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: #475569;
  font-size: 13px;
}

.timeline-bar {
  margin-top: 14px;
  background-color: #0f172a;
  padding: 10px;
  border-radius: 4px;
  border: 1px solid #1e293b;
}

.timeline-header {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: #64748b;
  margin-bottom: 8px;
  gap: 8px;
}

.timeline-scale {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.timeline-track {
  height: 24px;
  background-color: #1e293b;
  border-radius: 4px;
  display: flex;
  align-items: center;
  padding: 2px 4px;
  gap: 4px;
}

.timeline-segment {
  height: 16px;
  flex: 1;
  background-color: #10b981;
  border-radius: 2px;
  cursor: pointer;
  opacity: 0.8;
}

.timeline-segment:hover, .timeline-segment.active {
  opacity: 1;
  background-color: #3b82f6;
  box-shadow: 0 0 6px #3b82f6;
}

.card-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.records-scroll-list {
  height: 480px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rec-item {
  background-color: #0f172a;
  border: 1px solid #1e293b;
  border-radius: 4px;
  padding: 8px 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.rec-item:hover, .rec-item.selected {
  background-color: #1e293b;
  border-color: #3b82f6;
}

.rec-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: #e2e8f0;
}

.rec-item-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 6px;
  font-size: 11px;
  color: #64748b;
}

.empty-records {
  text-align: center;
  padding: 60px 0;
  color: #475569;
  font-size: 13px;
}

@media (max-width: 768px) {
  .playback-toolbar {
    padding: 10px 12px;
  }

  .filter-group {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }

  .filter-item {
    width: 100%;
  }

  .cam-select,
  .date-picker {
    flex: 1;
    width: 100% !important;
  }

  .search-btn {
    width: 100%;
  }

  .player-screen {
    height: 210px;
  }

  .records-scroll-list {
    height: 300px;
  }
}
</style>
