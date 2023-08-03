// Composables
import { createRouter, createWebHistory } from 'vue-router'
import {chat_list_name, chat_name} from "@/router/routes";

const routes = [
  {
    name: chat_name,
    path: '/front2/chat/:id',
    component: () => import('@/ChatView.vue'),
  },
  {
    name: chat_list_name,
    path: '/front2',
    component: () => import('@/ChatList.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
})

export default router
