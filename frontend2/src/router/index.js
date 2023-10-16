// Composables
import {createRouter, createWebHistory} from 'vue-router'
import {
  chat,
  chat_list_name,
  chat_name,
  chats,
  confirmation_pending,
  confirmation_pending_name,
  forgot_password,
  forgot_password_name, password_restore_check_email,
  password_restore_check_email_name, password_restore_enter_new, password_restore_enter_new_name,
  prefix,
  profile,
  profile_list_name,
  profile_name,
  profiles,
  registration,
  registration_name,
  root_name, video_suffix, videochat_name,
  wrong_confirmation_token,
  wrong_confirmation_token_name,
  wrong_user,
  wrong_user_name
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
      name: videochat_name,
      path: chat + `/:id` + video_suffix,
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
    {
      name: registration_name,
      path: registration,
      component: () => import('@/UserRegistration.vue'),
    },
    {
      name: confirmation_pending_name,
      path: confirmation_pending,
      component: () => import('@/UserRegistrationPendingConfirmation.vue'),
    },
    {
      name: wrong_confirmation_token_name,
      path: wrong_confirmation_token,
      component: () => import('@/UserRegistrationWrongConfirmationToken.vue'),
    },
    {
      name: wrong_user_name,
      path: wrong_user,
      component: () => import('@/UserRegistrationWrongUsername.vue'),
    },
    {
      name: forgot_password_name,
      path: forgot_password,
      component: () => import('@/UserRestorePassword.vue'),
    },
    {
        name: password_restore_check_email_name,
        path: password_restore_check_email,
        component: () => import('@/UserRestorePasswordCheckEmail.vue'),
    },
    {
        name: password_restore_enter_new_name,
        path: password_restore_enter_new,
        component: () => import('@/UserRestorePasswordEnterNew.vue'),
    },

]

const router = createRouter({
    history: createWebHistory(process.env.BASE_URL),
    routes,
})

export default router
