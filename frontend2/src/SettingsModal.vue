<template>
  <input id="image-input-profile-avatar" type="file" style="display: none;" accept="image/*"/>

  <v-dialog
    v-model="show"
    width="auto"
    scrollable
  >
    <v-card>
        <v-sheet elevation="6">
          <v-tabs
            v-model="tab"
            bg-color="indigo"
            show-arrows
          >
            <v-tab value="choose_language">
              {{ $vuetify.locale.t('$vuetify.language') }}
            </v-tab>
            <v-tab value="user_profile_self">
              {{ $vuetify.locale.t('$vuetify.user_profile_short') }}
            </v-tab>
            <v-tab value="a_video_settings">
              {{ $vuetify.locale.t('$vuetify.video') }}
            </v-tab>
            <v-tab value="the_notifications">
              {{ $vuetify.locale.t('$vuetify.notifications') }}
            </v-tab>
          </v-tabs>
        </v-sheet>

      <v-card-text class="ma-0 pa-0">
        <v-window v-model="tab">
            <v-window-item value="choose_language">
                <LanguageModalContent/>
            </v-window-item>
            <v-window-item value="user_profile_self">
                <UserSelfProfileModalContent/>
            </v-window-item>
            <v-window-item value="a_video_settings">
                <VideoGlobalSettingsModalContent  v-if="shouldShowVideoSettings()"/>
                <v-container v-else>
                    {{ $vuetify.locale.t('$vuetify.for_video_setting_please_open_chat') }}
                </v-container>
            </v-window-item>
            <v-window-item value="the_notifications">
                <NotificationSettingsModalContent/>
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
import bus, { OPEN_SETTINGS} from "@/bus/bus";
import LanguageModalContent from "@/LanguageModalContent.vue";
import VideoGlobalSettingsModalContent from "@/VideoGlobalSettingsModalContent.vue";
import NotificationSettingsModalContent from "@/NotificationSettingsModalContent.vue";
import UserSelfProfileModalContent from "@/UserSelfProfileModalContent.vue";
import {hasLength} from "@/utils";
import {v4 as uuidv4} from "uuid";
import axios from "axios";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";

export default {
  data () {
    return {
      tab: null,
      show: false,
      fileInput: null,
    }
  },
  components: {
      LanguageModalContent,
      VideoGlobalSettingsModalContent,
      NotificationSettingsModalContent,
      UserSelfProfileModalContent,
  },
  methods: {
    showLoginModal() {
      this.$data.show = true;
    },
    hideLoginModal() {
      this.$data.show = false;
    },
    shouldShowVideoSettings() {
        return !!this.$route.params.id
    },
    setAvatarToProfile(file) {
      const config = {
        headers: { 'content-type': 'multipart/form-data' }
      }
      const formData = new FormData();
      formData.append('data', file);
      return axios.post('/api/storage/avatar', formData, config)
        .then((res) => {
          return axios.patch(`/api/profile`, {avatar: res.data.relativeUrl, avatarBig: res.data.relativeBigUrl}).then((response) => {
            return this.chatStore.fetchUserProfile()
          })
        })
    },
    removeAvatarFromProfile() {
      return axios.patch(`/api/profile`, {removeAvatar: true}).then((response) => {
        return this.chatStore.fetchUserProfile()
      });
    },
    openAvatarDialog() {
      this.fileInput.click();
    },
  },
  computed: {
    ...mapStores(useChatStore),
    ava() {
      const maybeUser = this.chatStore.currentUser;
      if (maybeUser) {
        if (maybeUser.avatarBig) {
          return maybeUser.avatarBig
        } else if (maybeUser.avatar) {
          return maybeUser.avatar
        } else {
          return null
        }
      }
    },
    hasAva() {
      const maybeUser = this.chatStore.currentUser;
      return hasLength(maybeUser?.avatarBig) || hasLength(maybeUser?.avatar)
    },
  },
  created() {
    bus.on(OPEN_SETTINGS, this.showLoginModal);
  },
  mounted() {
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
    bus.off(OPEN_SETTINGS, this.showLoginModal);
    if (this.fileInput) {
      this.fileInput.onchange = null;
    }
    this.fileInput = null;
  },

}
</script>
