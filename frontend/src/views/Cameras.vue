<template>
  <div class="cameras-container">
    <!-- 顶部操作栏 -->
    <div class="action-bar">
      <div class="action-left">
        <el-button type="primary" icon="Plus" @click="openAddDialog">手动添加摄像头</el-button>
        <el-button type="success" icon="Search" @click="openScanDialog">跨网段自动发现与扫描</el-button>
        <el-divider direction="vertical" />
        <el-button :disabled="selectedIds.length === 0" @click="handleBatchRecord(true)">批量开启录像</el-button>
        <el-button :disabled="selectedIds.length === 0" @click="handleBatchRecord(false)">批量停止录像</el-button>
        <el-button :disabled="selectedIds.length === 0" @click="openBatchStorageDialog">批量修改存储位置</el-button>
      </div>

      <div class="action-right">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索名称 / IP / 型号"
          prefix-icon="Search"
          clearable
          style="width: 240px"
          @input="loadCameras"
        />
        <el-button icon="Refresh" circle @click="loadCameras" />
      </div>
    </div>

    <!-- 摄像头列表表格 -->
    <el-card shadow="never" class="mt-12 table-card">
      <el-table
        :data="cameras"
        v-loading="loading"
        @selection-change="handleSelectionChange"
        row-key="id"
      >
        <el-table-column type="selection" width="45" />
        <el-table-column label="在线" width="70" align="center">
          <template #default="{ row }">
            <el-tooltip :content="row.is_online ? '设备在线' : (row.last_offline_reason || '离线中')" placement="top">
              <span class="status-dot" :class="row.is_online ? 'online' : 'offline'"></span>
            </el-tooltip>
          </template>
        </el-table-column>

        <el-table-column prop="name" label="设备名称" min-width="140">
          <template #default="{ row }">
            <span class="cam-name">{{ row.name }}</span>
            <div class="cam-sub">{{ row.group || '默认分组' }}</div>
          </template>
        </el-table-column>

        <el-table-column label="IP 地址 / 跨网段配置" min-width="160">
          <template #default="{ row }">
            <span class="cam-ip">{{ row.ip }}:{{ row.rtsp_port }}</span>
            <div v-if="row.network_interface" class="iface-tag">
              出口网卡: {{ row.network_interface }}
            </div>
            <div v-if="row.subnet" class="subnet-tag">
              网段: {{ row.subnet }}
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="model" label="型号 / 编码" min-width="140">
          <template #default="{ row }">
            <div>{{ row.model || '海康威视 IPC' }}</div>
            <div class="codec-tag">{{ row.video_codec || 'H.264' }} | {{ row.resolution || '1080P' }}</div>
          </template>
        </el-table-column>

        <!-- 独立录像开关 (第7.2节需求) -->
        <el-table-column label="录像独立开关" width="130" align="center">
          <template #default="{ row }">
            <el-switch
              v-model="row.record_enabled"
              active-color="#ef4444"
              inactive-color="#475569"
              @change="(val: boolean) => toggleRecord(row, val)"
            />
          </template>
        </el-table-column>

        <!-- 实时录像状态 (REC 标识) -->
        <el-table-column label="录像状态" width="110" align="center">
          <template #default="{ row }">
            <span v-if="row.is_recording" class="rec-badge">
              <span class="rec-dot"></span> REC
            </span>
            <el-tag v-else-if="!row.record_enabled" size="small" type="info">未启用</el-tag>
            <el-tag v-else size="small" type="warning">等待上线</el-tag>
          </template>
        </el-table-column>

        <!-- 录像存储位置 (第17.4节需求) -->
        <el-table-column label="录像存储位置" min-width="140">
          <template #default="{ row }">
            <el-tag size="small" :type="getStorageTagType(row.storage_id)">
              {{ getStorageName(row.storage_id) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="180" align="center" fixed="right">
          <template #default="{ row }">
            <el-button size="small" link type="primary" @click="testCamera(row)">测试</el-button>
            <el-button size="small" link type="primary" @click="editCamera(row)">编辑</el-button>
            <el-popconfirm title="确定删除该摄像头吗？" @confirm="deleteCamera(row.id)">
              <template #reference>
                <el-button size="small" link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加 / 编辑摄像头对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑摄像头配置' : '添加摄像头 (适配跨网段与多网卡)'"
      width="580px"
    >
      <el-form :model="camForm" label-width="110px">
        <el-form-item label="设备名称" required>
          <el-input v-model="camForm.name" placeholder="例如: 1楼东门海康枪机" />
        </el-form-item>

        <el-form-item label="摄像头 IP" required>
          <el-input v-model="camForm.ip" placeholder="支持任意跨网段 IP，如 192.168.2.64" />
        </el-form-item>

        <el-row :gutter="10">
          <el-col :span="12">
            <el-form-item label="RTSP 端口">
              <el-input-number v-model="camForm.rtsp_port" :min="1" :max="65535" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="HTTP/ONVIF 端口">
              <el-input-number v-model="camForm.http_port" :min="1" :max="65535" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="10">
          <el-col :span="12">
            <el-form-item label="访问账号">
              <el-input v-model="camForm.username" placeholder="默认 admin" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="访问密码">
              <el-input
                v-model="camForm.password"
                type="password"
                show-password
                :placeholder="isEdit ? '留空保持原密码' : '请输入摄像头密码'"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="出口物理网卡">
          <el-select v-model="camForm.network_interface" placeholder="默认走系统路由表" clearable style="width: 100%">
            <el-option label="默认自动选择 (系统路由)" value="" />
            <el-option label="eth0 (2.5G 专用网口)" value="eth0" />
            <el-option label="eth1 (2.5G 监控口)" value="eth1" />
            <el-option label="br-lan (局域网网桥)" value="br-lan" />
          </el-select>
        </el-form-item>

        <el-form-item label="所属网段标签">
          <el-input v-model="camForm.subnet" placeholder="例如: 192.168.2.0/24 监控专网" />
        </el-form-item>

        <el-form-item label="所属设备分组">
          <el-input v-model="camForm.group" placeholder="例如: 1号办公楼 / 车库" />
        </el-form-item>

        <el-divider content-position="left">录像独立配置</el-divider>

        <el-form-item label="启用录像">
          <el-switch v-model="camForm.record_enabled" active-color="#ef4444" />
        </el-form-item>

        <el-form-item label="录像存储位置">
          <el-select v-model="camForm.storage_id" placeholder="选择存储设备" style="width: 100%">
            <el-option
              v-for="st in storages"
              :key="st.id"
              :label="`${st.name} (${st.type === 'smb' ? 'SMB NAS' : '本地盘'})`"
              :value="st.id"
            />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="testCurrentForm" :loading="testLoading">连通性测试</el-button>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveCamera" :loading="saveLoading">保存配置</el-button>
      </template>
    </el-dialog>

    <!-- 跨网段扫描与自动探测对话框 -->
    <el-dialog v-model="scanDialogVisible" title="跨网段摄像头自动发现与扫描" width="680px">
      <div class="scan-panel">
        <el-form label-width="120px">
          <el-form-item label="扫描网段列表">
            <el-input
              v-model="scanSubnetsInput"
              type="textarea"
              rows="3"
              placeholder="每行一个网段或范围，例如：
192.168.1.0/24
192.168.2.1-192.168.2.100"
            />
          </el-form-item>
          <el-form-item label="组播发现">
            <el-checkbox v-model="scanUseMulticast">同时执行本地局域网 ONVIF 组播探测</el-checkbox>
          </el-form-item>
        </el-form>

        <div class="scan-action">
          <el-button type="primary" :loading="scanning" @click="startScan">
            <el-icon><Search /></el-icon>
            {{ scanning ? '正在并发扫描探测网段...' : '开始跨网段探测' }}
          </el-button>
        </div>

        <el-divider v-if="discoveredDevices.length > 0">扫描结果 (找到 {{ discoveredDevices.length }} 台设备)</el-divider>

        <el-table
          v-if="discoveredDevices.length > 0"
          :data="discoveredDevices"
          max-height="260"
          @selection-change="handleDiscoveredSelect"
        >
          <el-table-column type="selection" width="45" />
          <el-table-column prop="ip" label="IP 地址" width="140" />
          <el-table-column prop="brand" label="品牌" width="100">
            <template #default="{ row }">
              <el-tag :type="row.brand === 'hikvision' ? 'danger' : 'info'" size="small">
                {{ row.brand === 'hikvision' ? '海康威视' : '通用ONVIF' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="model" label="型号/描述" />
          <el-table-column prop="probe_method" label="探测途径" width="130" />
        </el-table>
      </div>

      <template #footer>
        <el-button @click="scanDialogVisible = false">关闭</el-button>
        <el-button
          type="success"
          :disabled="selectedDiscovered.length === 0"
          @click="importDiscovered"
        >
          一键批量导入选中设备 ({{ selectedDiscovered.length }})
        </el-button>
      </template>
    </el-dialog>

    <!-- 批量修改存储位置对话框 -->
    <el-dialog v-model="batchStorageVisible" title="批量修改摄像头录像存储位置" width="440px">
      <el-form label-width="100px">
        <el-form-item label="目标存储设备">
          <el-select v-model="targetBatchStorageId" style="width: 100%">
            <el-option
              v-for="st in storages"
              :key="st.id"
              :label="`${st.name} (${st.type === 'smb' ? 'SMB NAS' : '本地'})`"
              :value="st.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchStorageVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmBatchStorage">确认批量变更</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const cameras = ref<any[]>([])
const storages = ref<any[]>([])
const loading = ref(false)
const searchKeyword = ref('')
const selectedIds = ref<number[]>([])

// 对话框表单
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentEditId = ref<number | null>(null)
const saveLoading = ref(false)
const testLoading = ref(false)

const camForm = ref<any>({
  name: '',
  ip: '',
  rtsp_port: 554,
  http_port: 80,
  username: 'admin',
  password: '',
  network_interface: '',
  subnet: '',
  group: '默认分组',
  record_enabled: false,
  storage_id: 1,
})

// 扫描
const scanDialogVisible = ref(false)
const scanning = ref(false)
const scanSubnetsInput = ref('192.168.1.0/24')
const scanUseMulticast = ref(true)
const discoveredDevices = ref<any[]>([])
const selectedDiscovered = ref<any[]>([])

// 批量存储
const batchStorageVisible = ref(false)
const targetBatchStorageId = ref<number | null>(null)

const loadCameras = async () => {
  loading.value = true
  try {
    const res: any = await api.getCameras({ keyword: searchKeyword.value })
    if (res.code === 0) {
      cameras.value = res.data
    }
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

const loadStorages = async () => {
  try {
    const res: any = await api.getStorages()
    if (res.code === 0) {
      storages.value = res.data
      if (storages.value.length > 0 && !camForm.value.storage_id) {
        camForm.value.storage_id = storages.value[0].id
      }
    }
  } catch (e) {
    console.error(e)
  }
}

const getStorageName = (id: number) => {
  const st = storages.value.find((s) => s.id === id)
  return st ? st.name : `存储#${id}`
}

const getStorageTagType = (id: number) => {
  const st = storages.value.find((s) => s.id === id)
  if (st && st.type === 'smb') return 'warning'
  return 'primary'
}

const handleSelectionChange = (selection: any[]) => {
  selectedIds.value = selection.map((item) => item.id)
}

const toggleRecord = async (row: any, val: boolean) => {
  try {
    await api.toggleCameraRecord(row.id, val)
    ElMessage.success(`摄像头 [${row.name}] 录像已${val ? '开启' : '关闭'}`)
    loadCameras()
  } catch (e: any) {
    row.record_enabled = !val
    ElMessage.error(e.message)
  }
}

const handleBatchRecord = async (enabled: boolean) => {
  try {
    await api.batchRecord(selectedIds.value, enabled)
    ElMessage.success(`已批量${enabled ? '开启' : '停止'} ${selectedIds.value.length} 台摄像头的录像`)
    loadCameras()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

const openBatchStorageDialog = () => {
  if (storages.value.length > 0) {
    targetBatchStorageId.value = storages.value[0].id
  }
  batchStorageVisible.value = true
}

const confirmBatchStorage = async () => {
  if (!targetBatchStorageId.value) return
  try {
    await api.batchStorage(selectedIds.value, targetBatchStorageId.value)
    ElMessage.success('批量修改录像存储位置成功')
    batchStorageVisible.value = false
    loadCameras()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

const openAddDialog = () => {
  isEdit.value = false
  currentEditId.value = null
  camForm.value = {
    name: '',
    ip: '',
    rtsp_port: 554,
    http_port: 80,
    username: 'admin',
    password: '',
    network_interface: '',
    subnet: '',
    group: '默认分组',
    record_enabled: false,
    storage_id: storages.value[0]?.id || 1,
  }
  dialogVisible.value = true
}

const editCamera = (row: any) => {
  isEdit.value = true
  currentEditId.value = row.id
  camForm.value = {
    name: row.name,
    ip: row.ip,
    rtsp_port: row.rtsp_port,
    http_port: row.http_port,
    username: row.username,
    password: '',
    network_interface: row.network_interface,
    subnet: row.subnet,
    group: row.group,
    record_enabled: row.record_enabled,
    storage_id: row.storage_id,
  }
  dialogVisible.value = true
}

const testCurrentForm = async () => {
  testLoading.value = true
  try {
    const res: any = await api.testCamera(camForm.value)
    if (res.code === 0) {
      ElMessage.success(`连通成功！检测到型号: ${res.data?.model || '海康IPC'}, 编码: ${res.data?.video_codec || 'H.264'}`)
    } else {
      ElMessage.warning(`连通异常: ${res.message}`)
    }
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    testLoading.value = false
  }
}

const testCamera = async (row: any) => {
  try {
    const res: any = await api.testCamera({ ...row, password: '' })
    if (res.code === 0) {
      ElMessage.success(`摄像头 [${row.name}] 在线正常！`)
    } else {
      ElMessage.warning(`连接异常: ${res.message}`)
    }
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

const saveCamera = async () => {
  if (!camForm.value.ip) {
    ElMessage.warning('请输入摄像头 IP 地址')
    return
  }
  saveLoading.value = true
  try {
    if (isEdit.value && currentEditId.value) {
      await api.updateCamera(currentEditId.value, camForm.value)
      ElMessage.success('更新摄像头配置成功')
    } else {
      await api.addCamera(camForm.value)
      ElMessage.success('添加摄像头成功')
    }
    dialogVisible.value = false
    loadCameras()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    saveLoading.value = false
  }
}

const deleteCamera = async (id: number) => {
  try {
    await api.deleteCamera(id)
    ElMessage.success('删除成功')
    loadCameras()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

const openScanDialog = () => {
  discoveredDevices.value = []
  selectedDiscovered.value = []
  scanDialogVisible.value = true
}

const startScan = async () => {
  const subnets = scanSubnetsInput.value
    .split('\n')
    .map((s) => s.trim())
    .filter((s) => s.length > 0)

  scanning.value = true
  try {
    const res: any = await api.scanDevices({
      subnets,
      use_multicast: scanUseMulticast.value,
    })
    if (res.code === 0) {
      discoveredDevices.value = res.data
      if (res.data.length === 0) {
        ElMessage.info('未在指定网段中扫描到开放端口的摄像头')
      } else {
        ElMessage.success(`探测完成，发现 ${res.data.length} 台设备！`)
      }
    }
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    scanning.value = false
  }
}

const handleDiscoveredSelect = (selection: any[]) => {
  selectedDiscovered.value = selection
}

const importDiscovered = async () => {
  let successCount = 0
  for (const dev of selectedDiscovered.value) {
    try {
      await api.addCamera({
        name: `海康-${dev.ip.split('.').slice(-2).join('.')}`,
        ip: dev.ip,
        rtsp_port: dev.port || 554,
        http_port: 80,
        username: 'admin',
        brand: dev.brand || 'hikvision',
        model: dev.model || '',
        storage_id: storages.value[0]?.id || 1,
      })
      successCount++
    } catch (e) {
      console.error(e)
    }
  }
  ElMessage.success(`已成功导入 ${successCount} 台摄像头！`)
  scanDialogVisible.value = false
  loadCameras()
}

onMounted(() => {
  loadCameras()
  loadStorages()
})
</script>

<style scoped>
.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: var(--panel-bg);
  padding: 10px 16px;
  border-radius: 6px;
}

.action-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.table-card {
  border-radius: 6px;
}

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-dot.online {
  background-color: #10b981;
  box-shadow: 0 0 6px #10b981;
}

.status-dot.offline {
  background-color: #ef4444;
}

.cam-name {
  font-weight: 500;
  color: #f1f5f9;
}

.cam-sub {
  font-size: 11px;
  color: #64748b;
}

.cam-ip {
  color: #60a5fa;
  font-family: monospace;
}

.iface-tag, .subnet-tag {
  font-size: 10px;
  color: #94a3b8;
}

.codec-tag {
  font-size: 11px;
  color: #a78bfa;
}

.scan-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.scan-action {
  display: flex;
  justify-content: center;
}
</style>
