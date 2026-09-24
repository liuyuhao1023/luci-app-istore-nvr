import { createRouter, createWebHistory } from 'vue-router'
import Overview from '../views/Overview.vue'
import LiveView from '../views/LiveView.vue'
import Cameras from '../views/Cameras.vue'
import Playback from '../views/Playback.vue'
import Storage from '../views/Storage.vue'

const routes = [
  { path: '/', name: 'Overview', component: Overview },
  { path: '/live', name: 'LiveView', component: LiveView },
  { path: '/cameras', name: 'Cameras', component: Cameras },
  { path: '/playback', name: 'Playback', component: Playback },
  { path: '/storage', name: 'Storage', component: Storage },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
