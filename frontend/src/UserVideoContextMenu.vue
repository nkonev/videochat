<template>
    <v-menu
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
                @click="item.action"
            >
              <template v-slot:prepend>
                <v-icon :color="item.iconColor">
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

export default {
    mixins: [
      contextMenuMixin(),
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
        'videoMute'
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
                if (this.isMobile()) {
                    ret.push({
                        title: this.$vuetify.locale.t('$vuetify.close'),
                        icon: 'mdi-close',
                        action: () => {
                            this.onCloseContextMenu()
                        }
                    });
                }
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
                    }
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
                  }
                });
              }

              if (this.shouldShowClose) {
                ret.push({
                  title: this.$vuetify.locale.t('$vuetify.close_video'),
                  icon: 'mdi-close',
                  action: () => {
                    this.menuableItem.onClose()
                  }
                });
              }

              if (this.shouldShowVideoKick) {
                ret.push({
                  title: this.$vuetify.locale.t('$vuetify.kick'),
                  icon: 'mdi-block-helper',
                  action: () => {
                    this.menuableItem.kickRemote()
                  }
                });
              }

              if (this.shouldShowAudioMute) {
                ret.push({
                  title: this.$vuetify.locale.t('$vuetify.force_mute'),
                  icon: 'mdi-microphone-off',
                  action: () => {
                    this.menuableItem.forceMuteRemote()
                  }
                });
              }
            }
            return ret;
        },
    }
}
</script>
