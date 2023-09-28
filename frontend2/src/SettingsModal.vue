<template>
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
        <div id="prepending-buttons"/>
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

export default {
  data () {
    return {
      tab: null,
      show: false,
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
  },
  created() {
    bus.on(OPEN_SETTINGS, this.showLoginModal);
  },
  destroyed() {
    bus.off(OPEN_SETTINGS, this.showLoginModal);
  },

}
</script>
