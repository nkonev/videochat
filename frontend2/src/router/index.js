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
  registration_name, root,
  root_name, video_suffix, videochat_name,
  wrong_confirmation_token,
  wrong_confirmation_token_name,
  wrong_user,
  wrong_user_name
} from "@/router/routes";
import vuetify from "@/plugins/vuetify";
import bus, {CLOSE_SIMPLE_MODAL, OPEN_SIMPLE_MODAL} from "@/bus/bus";
import {useChatStore} from "@/store/chatStore";

const routes = [
    {
        name: root_name,
        path: root,
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
});

router.beforeEach((to, from, next) => {
  const chatStore = useChatStore();

  if (from.name == videochat_name && to.name != videochat_name && chatStore.leavingVideoAcceptableParam != true) {
    bus.emit(OPEN_SIMPLE_MODAL, {
      buttonName: vuetify.locale.t('$vuetify.ok'),
      title: vuetify.locale.t('$vuetify.leave_call'),
      text: vuetify.locale.t('$vuetify.leave_call_text'),
      actionFunction: ()=> {
        next();
        bus.emit(CLOSE_SIMPLE_MODAL);
      },
      cancelFunction: ()=>{
        next(false)
      }
    });
  } else {
    chatStore.leavingVideoAcceptableParam = false;
    next();
  }
});

export default router
