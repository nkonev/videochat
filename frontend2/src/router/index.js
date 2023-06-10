// Composables
import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    name: 'root',
    path: '/front2',
    component: () => import('@/Default.vue'),
  },
  {
    name: 'list',
    path: '/front2/list',
    component: () => import('@/ChatList.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
})

export default router
