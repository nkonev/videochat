<template>
    <v-menu
        attach="#video-splitpanes"
        :class="className()"
        :model-value="showContextMenu"
        :transition="false"
        :open-on-click="false"
        :open-on-focus="false"
        :open-on-hover="false"
        :open-delay="0"
        :close-delay="0"
        :close-on-back="false"
        @update:modelValue="onUpdate"
    >
        <v-list>
            <v-list-item
                v-for="(item, index) in getContextMenuItems()"
                :key="index"
                :disabled="!item.enabled"
                @click="item.action"
            >
              <template v-slot:prepend>
                <v-icon v-if="item.icon" :color="item.iconColor">
                  {{item.icon}}
                </v-icon>
              </template>
              <template v-slot:title>{{ item.title }}</template>
            </v-list-item>
        </v-list>
    </v-menu>
</template>

<script>
import contextMenuMixin from "@/mixins/contextMenuMixin";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import bus, {PIN_VIDEO, UN_PIN_VIDEO} from "@/bus/bus.js";
import videoPositionMixin from "@/mixins/videoPositionMixin.js";

export default {
    mixins: [
      contextMenuMixin(),
      videoPositionMixin(),
    ],
    computed: {
        ...mapStores(useChatStore),
    },
    props: [
        'isLocal',
        'shouldShowMuteAudio',
        'shouldShowMuteVideo',
        'shouldShowClose',
        'shouldShowVideoKick',
        'shouldShowAudioMute',
        'audioMute',
        'videoMute',
        'userName'
    ],
    methods:{
        className() {
            return "user-video-item-context-menu"
        },
        onShowContextMenu(e, menuableItem) {
          this.onShowContextMenuBase(e, menuableItem);
        },
        onCloseContextMenu() {
          this.onCloseContextMenuBase();
        },
        getContextMenuItems() {
            const ret = [];
            if (this.menuableItem) {
                ret.push({
                    title: this.userName,
                    icon: 'mdi-close',
                    action: () => {
                        this.onCloseContextMenu()
                    },
                    enabled: true,
                });

                if (this.shouldShowMuteAudio) {
                  ret.push({
                    title: this.audioMute ? this.$vuetify.locale.t('$vuetify.unmute_audio') : this.$vuetify.locale.t('$vuetify.mute_audio'),
                    icon: this.audioMute ? 'mdi-microphone-off' : 'mdi-microphone',
                    action: () => {
                      if (this.audioMute) {
                        this.menuableItem.doMuteAudio(false)
                      } else {
                        this.menuableItem.doMuteAudio(true)
                      }
                    },
                    enabled: true,
                  });
                }

              if (this.shouldShowMuteVideo) {
                ret.push({
                  title: this.videoMute ? this.$vuetify.locale.t('$vuetify.unmute_video') : this.$vuetify.locale.t('$vuetify.mute_video'),
                  icon: this.videoMute ? 'mdi-video-off' : 'mdi-video',
                  action: () => {
                    if (this.videoMute) {
                      this.menuableItem.doMuteVideo(false)
                    } else {
                      this.menuableItem.doMuteVideo(true)
                    }
                  },
                  enabled: true,
                });
              }

              if (this.shouldShowClose) {
                ret.push({
                  title: this.$vuetify.locale.t('$vuetify.close_video'),
                  icon: 'mdi-close',
                  iconColor: 'error',
                  action: () => {
                    this.menuableItem.onLocalClose()
                  },
                  enabled: true,
                });
              }

              if (this.shouldShowVideoKick) {
                ret.push({
                  title: this.$vuetify.locale.t('$vuetify.kick'),
                  icon: 'mdi-block-helper',
                  iconColor: 'error',
                  action: () => {
                    this.menuableItem.kickRemote()
                  },
                  enabled: true,
                });
              }

              if (this.shouldShowAudioMute) {
                ret.push({
                  title: this.$vuetify.locale.t('$vuetify.force_mute'),
                  icon: 'mdi-microphone-off',
                  iconColor: 'error',
                  action: () => {
                    this.menuableItem.forceMuteRemote()
                  },
                  enabled: true,
                });
              }

              if (this.pinningIsAvailable()) {
                ret.push({
                  title: this.$vuetify.locale.t('$vuetify.pin_video'),
                  icon: 'mdi-pin',
                  action: () => {
                    bus.emit(PIN_VIDEO, this.menuableItem.getVideoStreamId())
                  },
                  enabled: true,
                });
                if (this.chatStore.pinnedTrackSid) {
                  ret.push({
                    title: this.$vuetify.locale.t('$vuetify.unpin_video'),
                    icon: 'mdi-pin-off-outline',
                    action: () => {
                      bus.emit(UN_PIN_VIDEO)
                    },
                    enabled: true,
                  });
                }
              }
            }
            return ret;
        },
    }
}
</script>
