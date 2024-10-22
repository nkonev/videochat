<template>
  <input id="image-input-profile-avatar" type="file" style="display: none;" accept="image/*"/>

  <v-dialog
    v-model="show"
    width="auto"
    scrollable
  >
    <v-card :loading="loading">
        <v-sheet elevation="6">
          <v-tabs
            v-model="tab"
            bg-color="indigo"
            show-arrows
          >
            <v-tab value="choose_language">
              {{ $vuetify.locale.t('$vuetify.language') }}
            </v-tab>
            <v-tab value="user_profile_self" v-if="this.chatStore.currentUser">
              {{ $vuetify.locale.t('$vuetify.user_profile_short') }}
            </v-tab>
            <v-tab value="a_video_settings" v-if="this.chatStore.currentUser">
              {{ $vuetify.locale.t('$vuetify.video') }}
            </v-tab>
            <v-tab value="the_notifications" v-if="this.chatStore.currentUser">
              {{ $vuetify.locale.t('$vuetify.notifications') }}
            </v-tab>
            <v-tab value="message_edit_settings" v-if="this.chatStore.currentUser">
              {{ $vuetify.locale.t('$vuetify.message_edit_settings_tab') }}
            </v-tab>
          </v-tabs>
        </v-sheet>

      <v-card-text class="ma-0 pa-0">
        <v-window v-model="tab">
            <v-window-item value="choose_language">
                <LanguageModalContent/>
            </v-window-item>
            <v-window-item value="user_profile_self" v-if="this.chatStore.currentUser">
                <UserSelfProfileModalContent/>
            </v-window-item>
            <v-window-item value="a_video_settings" v-if="this.chatStore.currentUser">
                <VideoGlobalSettingsModalContent/>
            </v-window-item>
            <v-window-item value="the_notifications" v-if="this.chatStore.currentUser">
                <NotificationSettingsModalContent/>
            </v-window-item>
            <v-window-item value="message_edit_settings" v-if="this.chatStore.currentUser">
                <MessageEditSettingsModalContent/>
            </v-window-item>
        </v-window>
      </v-card-text>

      <v-card-actions>
        <template v-if="tab == 'user_profile_self'">
          <v-btn v-if="hasAva" variant="outlined" @click="removeAvatarFromProfile()">
            <template v-slot:prepend>
              <v-icon>mdi-image-remove</v-icon>
            </template>
            <template v-slot:default>
              {{ $vuetify.locale.t('$vuetify.remove_avatar_btn') }}
            </template>
          </v-btn>
          <v-btn v-if="!hasAva" variant="outlined" @click="openAvatarDialog()">
            <template v-slot:prepend>
              <v-icon>mdi-image-outline</v-icon>
            </template>
            <template v-slot:default>
              {{ $vuetify.locale.t('$vuetify.choose_avatar_btn') }}
            </template>
          </v-btn>
        </template>
        <v-spacer/>
        <v-btn color="red" variant="flat" @click="hideLoginModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
import bus, {OPEN_SETTINGS, REQUEST_CHANGE_VIDEO_PARAMETERS, VIDEO_PARAMETERS_CHANGED} from "@/bus/bus";
import LanguageModalContent from "@/LanguageModalContent.vue";
import VideoGlobalSettingsModalContent from "@/VideoGlobalSettingsModalContent.vue";
import NotificationSettingsModalContent from "@/NotificationSettingsModalContent.vue";
import UserSelfProfileModalContent from "@/UserSelfProfileModalContent.vue";
import MessageEditSettingsModalContent from "@/MessageEditSettingsModalContent.vue";
import {hasLength} from "@/utils";
import axios from "axios";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {videochat_name} from "@/router/routes.js";

const LOADING_COLOR = 'white';

export default {
  data () {
    return {
      tab: null,
      show: false,
      fileInput: null,
      loading: false,
    }
  },
  components: {
      LanguageModalContent,
      VideoGlobalSettingsModalContent,
      NotificationSettingsModalContent,
      UserSelfProfileModalContent,
      MessageEditSettingsModalContent,
  },
  methods: {
    showSettingsModal(tab) {
      if (hasLength(tab)) {
        this.tab = tab;
      }
      this.$data.show = true;
    },
    hideLoginModal() {
      this.$data.show = false;
    },
    setAvatarToProfile(file) {
      this.loading = LOADING_COLOR;
      const config = {
        headers: { 'content-type': 'multipart/form-data' }
      }
      const formData = new FormData();
      formData.append('data', file);
      return axios.post('/api/storage/avatar', formData, config)
        .then((res) => {
          return axios.patch(`/api/aaa/profile`, {avatar: res.data.relativeUrl, avatarBig: res.data.relativeBigUrl})
        }).finally(()=>{
          this.loading = false;
        });
    },
    removeAvatarFromProfile() {
      this.loading = LOADING_COLOR;
      return axios.patch(`/api/aaa/profile`, {removeAvatar: true})
          .finally(()=>{
            this.loading = false;
          });
    },
    openAvatarDialog() {
      this.fileInput.click();
    },
    onRequestVideoParametersChange() {
      if (this.isVideoRoute()) {
        this.loading = LOADING_COLOR;
      }
    },
    onVideoParametersChanged() {
      this.loading = false;
    },
    isVideoRoute() {
      return this.$route.name == videochat_name
    },
  },
  computed: {
    ...mapStores(useChatStore),
    hasAva() {
      const maybeUser = this.chatStore.currentUser;
      return hasLength(maybeUser?.avatarBig) || hasLength(maybeUser?.avatar)
    },
  },
  created() {
  },
  mounted() {
    bus.on(OPEN_SETTINGS, this.showSettingsModal);
    bus.on(REQUEST_CHANGE_VIDEO_PARAMETERS, this.onRequestVideoParametersChange);
    bus.on(VIDEO_PARAMETERS_CHANGED, this.onVideoParametersChanged);
    this.fileInput = document.getElementById('image-input-profile-avatar');
    this.fileInput.onchange = (e) => {
      if (e.target.files.length) {
        const files = Array.from(e.target.files);
        const file = files[0];
        this.setAvatarToProfile(file);
      }
    }
  },
  beforeUnmount() {
    bus.off(OPEN_SETTINGS, this.showSettingsModal);
    bus.off(REQUEST_CHANGE_VIDEO_PARAMETERS, this.onRequestVideoParametersChange);
    bus.off(VIDEO_PARAMETERS_CHANGED, this.onVideoParametersChanged);
    if (this.fileInput) {
      this.fileInput.onchange = null;
    }
    this.fileInput = null;
  },

}
</script>
