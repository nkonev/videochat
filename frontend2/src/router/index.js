// Composables
import {createRouter, createWebHistory} from 'vue-router'
import {chat_list_name, chat_name, chat_view_name} from "@/router/routes";

const routes = [
    {
        name: chat_list_name,
        path: '/front2/chats',
        component: () => import('@/ChatList.vue'),
    },
    {
        name: chat_view_name,
        path: '/front2/chat/:id',
        component: () => import('@/ChatView.vue'),
    },
]

const router = createRouter({
    history: createWebHistory(process.env.BASE_URL),
    routes,
})

export default router
