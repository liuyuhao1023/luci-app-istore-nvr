import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 15000,
})

api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const msg = error.response?.data?.error || error.message || '请求服务发生异常'
    return Promise.reject(new Error(msg))
  }
)

export default {
  // 1. 摄像头接口
  getCameras(params?: any) {
    return api.get('/cameras', { params })
  },
  getCamera(id: number) {
    return api.get(`/cameras/${id}`)
  },
  addCamera(data: any) {
    return api.post('/cameras', data)
  },
  updateCamera(id: number, data: any) {
    return api.put(`/cameras/${id}`, data)
  },
  deleteCamera(id: number) {
    return api.delete(`/cameras/${id}`)
  },
  testCamera(data: any) {
    return api.post('/cameras/test', data)
  },
  scanDevices(data: { subnets: string[]; use_multicast: boolean }) {
    return api.post('/cameras/scan', data)
  },
  toggleCameraRecord(id: number, enabled: boolean) {
    return api.put(`/cameras/${id}/record`, { enabled })
  },
  batchRecord(ids: number[], enabled: boolean) {
    return api.post('/cameras/batch/record', { ids, enabled })
  },
  batchStorage(ids: number[], storageId: number) {
    return api.post('/cameras/batch/storage', { ids, storage_id: storageId })
  },

  // 2. 存储管理接口 (含 SMB)
  getStorages() {
    return api.get('/storages')
  },
  addStorage(data: any) {
    return api.post('/storages', data)
  },
  updateStorage(id: number, data: any) {
    return api.put(`/storages/${id}`, data)
  },
  deleteStorage(id: number) {
    return api.delete(`/storages/${id}`)
  },
  mountStorage(id: number) {
    return api.post(`/storages/${id}/mount`)
  },

  // 3. 录像接口
  getGlobalRecord() {
    return api.get('/record/global')
  },
  setGlobalRecord(enabled: boolean) {
    return api.post('/record/global', { enabled })
  },
  queryRecordings(params: any) {
    return api.get('/record/query', { params })
  },

  // 4. 流媒体
  requestStream(id: number, stream: 'main' | 'sub') {
    return api.get(`/stream/${id}/request?stream=${stream}`)
  },
  releaseStream(id: number, stream: 'main' | 'sub') {
    return api.post(`/stream/${id}/release?stream=${stream}`)
  },

  // 5. 系统状态
  getSystemStats() {
    return api.get('/system/stats')
  },
}
