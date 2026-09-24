<template>
  <div class="storage-container">
    <div class="action-bar">
      <div class="action-left">
        <el-button type="primary" icon="Plus" @click="openAddDialog('smb')">添加 SMB/CIFS 网络存储 (NAS)</el-button>
        <el-button icon="FolderAdd" @click="openAddDialog('local')">添加本地硬盘目录</el-button>
      </div>
      <div class="action-right">
        <el-button icon="Refresh" circle @click="loadStorages" />
      </div>
    </div>

    <!-- 存储设备列表卡片 -->
    <el-row :gutter="16" class="mt-12">
      <el-col :span="12" v-for="st in storages" :key="st.id">
        <el-card shadow="never" class="storage-card">
          <template #header>
            <div class="card-header">
              <div class="st-title-group">
                <el-icon :size="20" :color="st.type === 'smb' ? '#3b82f6' : '#f59e0b'"><Coin /></el-icon>
                <span class="st-name">{{ st.name }}</span>
                <el-tag size="small" :type="st.type === 'smb' ? 'primary' : 'warning'">
                  {{ st.type === 'smb' ? 'SMB 网络共享 (NAS)' : '本地磁盘' }}
                </el-tag>
              </div>

              <!-- 状态标识 -->
              <el-tag :type="getStatusTag(st.status)" size="small">
                {{ getStatusText(st.status) }}
              </el-tag>
            </div>
          </template>

          <div class="card-body">
            <!-- SMB 路径或挂载点 -->
            <div class="path-info">
              <span class="label">挂载路径：</span>
              <span class="val-mono">{{ st.mount_point }}</span>
            </div>
            <div v-if="st.type === 'smb'" class="path-info">
              <span class="label">远程 NAS：</span>
              <span class="val-mono">//{{ st.server_host }}/{{ st.share_name }} (协议: {{ st.smb_version || '3.0' }})</span>
            </div>

            <!-- 容量进度条 -->
            <div class="usage-section">
              <div class="usage-label">
                <span>存储空间使用情况</span>
                <span>{{ formatGB(st.used_bytes) }} GB / {{ formatGB(st.total_bytes) }} GB</span>
              </div>
              <el-progress
                :percentage="calcUsagePct(st)"
                :status="calcUsagePct(st) > st.alert_threshold_percent ? 'exception' : ''"
                :stroke-width="12"
              />
            </div>

            <!-- 统计标签 -->
            <div class="meta-row">
              <div class="meta-item">
                <span class="meta-k">已分配摄像头：</span>
                <span class="meta-v">{{ st.allocated_cameras }} 路</span>
              </div>
              <div class="meta-item">
                <span class="meta-k">空间告警阈值：</span>
                <span class="meta-v">{{ st.alert_threshold_percent }}%</span>
              </div>
              <div class="meta-item">
                <span class="meta-k">自动清理：</span>
                <span class="meta-v">{{ st.auto_clean_enabled ? '已开启' : '关闭' }}</span>
              </div>
            </div>

            <!-- 异常警告 -->
            <el-alert
              v-if="st.last_error_message"
              :title="st.last_error_message"
              type="error"
              show-icon
              :closable="false"
              class="mt-8"
            />
          </div>

          <!-- 底部操作按钮 -->
          <div class="card-footer">
            <el-button v-if="st.type === 'smb'" size="small" type="primary" @click="testMount(st)">
              挂载与读写测试
            </el-button>
            <el-button size="small" @click="editStorage(st)">编辑设置</el-button>
            <el-popconfirm title="确定删除该存储配置吗？已分配摄像头的录像将会暂停。" @confirm="deleteStorage(st.id)">
              <template #reference>
                <el-button size="small" type="danger" link>删除</el-button>
              </template>
            </el-popconfirm>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 添加 / 编辑存储对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑存储配置' : (formType === 'smb' ? '添加 SMB/CIFS 网络存储 (NAS)' : '添加本地存储目录')"
      width="540px"
    >
      <el-form :model="storageForm" label-width="110px">
        <el-form-item label="存储名称" required>
          <el-input v-model="storageForm.name" placeholder="例如: 威联通NAS主存储 / 本地SATA盘" />
        </el-form-item>

        <!-- SMB 专有项 -->
        <template v-if="formType === 'smb'">
          <el-form-item label="NAS 地址" required>
            <el-input v-model="storageForm.server_host" placeholder="例如: 192.168.1.20 或 nas.lan" />
          </el-form-item>

          <el-form-item label="共享目录名" required>
            <el-input v-model="storageForm.share_name" placeholder="例如: CameraRecordings" />
          </el-form-item>

          <el-row :gutter="10">
            <el-col :span="12">
              <el-form-item label="SMB 账号">
                <el-input v-model="storageForm.username" placeholder="录像专用共享账号" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="SMB 密码">
                <el-input
                  v-model="storageForm.password"
                  type="password"
                  show-password
                  :placeholder="isEdit ? '留空保持原密码' : '共享密码'"
                />
              </el-form-item>
            </el-col>
          </el-row>

          <el-form-item label="SMB 协议版本">
            <el-select v-model="storageForm.smb_version" style="width: 100%">
              <el-option label="SMB 3.0 (现代 NAS 推荐，高速安全)" value="3.0" />
              <el-option label="SMB 2.1 (兼容模式)" value="2.1" />
              <el-option label="自动协商 (Auto)" value="auto" />
            </el-select>
          </el-form-item>

          <el-form-item label="挂载点路径">
            <el-input v-model="storageForm.mount_point" placeholder="留空则自动分配至 /mnt/nvr/storage/..." />
          </el-form-item>
        </template>

        <!-- 本地目录 -->
        <template v-else>
          <el-form-item label="本地目录路径" required>
            <el-input v-model="storageForm.mount_point" placeholder="例如: /mnt/sata1-4/recordings" />
          </el-form-item>
        </template>

        <el-divider content-position="left">保留与清理策略</el-divider>

        <el-row :gutter="10">
          <el-col :span="12">
            <el-form-item label="空间告警(%)">
              <el-input-number v-model="storageForm.alert_threshold_percent" :min="50" :max="99" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="自动清理">
              <el-switch v-model="storageForm.auto_clean_enabled" active-text="启用" inactive-text="关闭" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saveLoading" @click="saveStorage">保存并测试</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const storages = ref<any[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const formType = ref<'local' | 'smb'>('smb')
const currentEditId = ref<number | null>(null)
const saveLoading = ref(false)

const storageForm = ref<any>({
  name: '',
  type: 'smb',
  server_host: '',
  share_name: '',
  username: '',
  password: '',
  smb_version: '3.0',
  mount_point: '',
  alert_threshold_percent: 90,
  auto_clean_enabled: true,
  min_retention_days: 3,
})

const loadStorages = async () => {
  try {
    const res: any = await api.getStorages()
    if (res.code === 0) {
      storages.value = res.data
    }
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

const getStatusTag = (status: string) => {
  switch (status) {
    case 'mounted': return 'success'
    case 'offline': return 'danger'
    case 'error': return 'danger'
    default: return 'info'
  }
}

const getStatusText = (status: string) => {
  switch (status) {
    case 'mounted': return '正常挂载 (已验证读写)'
    case 'offline': return '存储脱机'
    case 'error': return '挂载/权限异常'
    default: return '未挂载'
  }
}

const formatGB = (bytes: number) => {
  if (!bytes) return '0.0'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1)
}

const calcUsagePct = (st: any) => {
  if (!st.total_bytes || st.total_bytes === 0) return 0
  const pct = Math.round((st.used_bytes / st.total_bytes) * 100)
  return isNaN(pct) ? 0 : pct
}

const openAddDialog = (type: 'local' | 'smb') => {
  isEdit.value = false
  currentEditId.value = null
  formType.value = type
  storageForm.value = {
    name: '',
    type,
    server_host: '',
    share_name: '',
    username: '',
    password: '',
    smb_version: '3.0',
    mount_point: type === 'local' ? '/mnt/sata1-4/recordings' : '',
    alert_threshold_percent: 90,
    auto_clean_enabled: true,
    min_retention_days: 3,
  }
  dialogVisible.value = true
}

const editStorage = (st: any) => {
  isEdit.value = true
  currentEditId.value = st.id
  formType.value = st.type
  storageForm.value = {
    name: st.name,
    type: st.type,
    server_host: st.server_host,
    share_name: st.share_name,
    username: st.username,
    password: '',
    smb_version: st.smb_version || '3.0',
    mount_point: st.mount_point,
    alert_threshold_percent: st.alert_threshold_percent,
    auto_clean_enabled: st.auto_clean_enabled,
    min_retention_days: st.min_retention_days,
  }
  dialogVisible.value = true
}

const testMount = async (st: any) => {
  try {
    const res: any = await api.mountStorage(st.id)
    if (res.code === 0) {
      ElMessage.success('SMB 挂载成功！读写权限校验通过。')
      loadStorages()
    } else {
      ElMessage.error(`挂载失败: ${res.message}`)
    }
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

const saveStorage = async () => {
  saveLoading.value = true
  try {
    if (isEdit.value && currentEditId.value) {
      await api.updateStorage(currentEditId.value, storageForm.value)
      ElMessage.success('存储配置更新成功')
    } else {
      await api.addStorage(storageForm.value)
      ElMessage.success('存储设备添加成功')
    }
    dialogVisible.value = false
    loadStorages()
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    saveLoading.value = false
  }
}

const deleteStorage = async (id: number) => {
  try {
    await api.deleteStorage(id)
    ElMessage.success('删除成功')
    loadStorages()
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

onMounted(() => {
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

.storage-card {
  border-radius: 8px;
  margin-bottom: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.st-title-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.st-name {
  font-weight: bold;
  font-size: 15px;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.path-info {
  font-size: 12px;
  display: flex;
  align-items: center;
}

.path-info .label {
  color: #94a3b8;
  width: 70px;
}

.val-mono {
  font-family: monospace;
  color: #60a5fa;
}

.usage-section {
  background-color: #0f172a;
  padding: 10px;
  border-radius: 6px;
  border: 1px solid #1e293b;
}

.usage-label {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #cbd5e1;
  margin-bottom: 6px;
}

.meta-row {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #94a3b8;
  border-top: 1px solid #1e293b;
  padding-top: 8px;
}

.meta-v {
  color: #f1f5f9;
  font-weight: 500;
}

.mt-8 {
  margin-top: 8px;
}

.card-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  border-top: 1px solid #1e293b;
  margin-top: 12px;
  padding-top: 10px;
}
</style>
