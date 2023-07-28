// Composables
import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    name: 'view',
    path: '/front2/chat/:id',
    component: () => import('@/ChatView.vue'),
  },
  {
    name: 'list',
    path: '/front2',
    component: () => import('@/ChatList.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
})

export default router
