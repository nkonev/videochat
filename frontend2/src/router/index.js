// Composables
import {createRouter, createWebHistory} from 'vue-router'
import {
  chat,
  chat_list_name,
  chat_name,
  chats,
  prefix, profile,
  profile_list_name,
  profile_name,
  profiles,
  root_name
} from "@/router/routes";

const routes = [
    {
        name: root_name,
        path: prefix,
        component: () => import('@/Welcome.vue'),
    },
    {
        name: chat_list_name,
        path: chats,
        component: () => import('@/ChatList.vue'),
    },
    {
        name: chat_name,
        path: chat + `/:id`,
        component: () => import('@/ChatView.vue'),
    },
    {
        name: profile_name,
        path: profile + `/:id`,
        component: () => import('@/UserProfile.vue'),
    },
    {
      name: profile_list_name,
      path: profiles,
      component: () => import('@/UserList.vue'),
    },

]

const router = createRouter({
    history: createWebHistory(process.env.BASE_URL),
    routes,
})

export default router
