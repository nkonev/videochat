<template>
  <pane :size="provider.presenterPaneSize()" :class="provider.presenterPaneClass">
    <div class="video-presenter-container-element" @contextmenu.stop="onShowContextMenu($event, this)">
      <video v-show="!provider.presenterVideoMute || !provider.presenterAvatarIsSet" @click.self="provider.onClick()" :class="presenterVideoClass" ref="presenterVideoRef"/>
      <img v-show="provider.presenterAvatarIsSet && provider.presenterVideoMute" @click.self="provider.onClick()" class="video-presenter-element" :src="provider.presenterData?.avatar"/>
      <p v-bind:class="[provider.speaking ? 'presenter-element-caption-speaking' : '', 'presenter-element-caption', 'inline-caption-base']">{{ provider.presenterData?.userName ? provider.presenterData?.userName : provider.getLoadingMessage() }} <v-icon v-if="provider.presenterAudioMute">mdi-microphone-off</v-icon></p>

      <VideoButtons v-if="!isMobile()" @requestFullScreen="provider.onButtonsFullscreen" v-show="provider.showControls"/>

      <PresenterContextMenu ref="contextMenuRef" :userName="provider.presenterUserName"/>

      <v-btn v-if="chatStore.pinnedTrackSid" class="presenter-unpin-button" @click="provider.doUnpinVideo()" icon="mdi-pin-off-outline" rounded="0" :title="$vuetify.locale.t('$vuetify.unpin_video')"></v-btn>
    </div>
  </pane>
</template>
<script>
import VideoButtons from "@/VideoButtons.vue";
import PresenterContextMenu from "@/PresenterContextMenu.vue";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore.js";
import { Pane } from 'splitpanes';

export default {
  props: [
    'provider',
  ],
  components: {
    VideoButtons,
    PresenterContextMenu,
    Pane,
  },
  computed: {
    ...mapStores(useChatStore),
    presenterVideoClass() {
      const arr = ["video-presenter-element"];
      if (!this.provider.presenterData?.isScreenShare && this.chatStore.presenterUseCover) {
        arr.push("video-presenter-element-cover");
      }

      return arr;
    },
  },
  methods: {
    onShowContextMenu(e, menuableItem) {
      this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
    },
    getPresenterVideoStreamId() {
      return this.provider.presenterData?.videoStream.trackSid
    },
  },
}
</script>

<style lang="stylus" scoped>
.video-presenter-container-element {
  position relative
  display flex
  flex-direction column
  align-items: center;

  width 100%
  height 100%
}

.video-presenter-element {
  //box-sizing: border-box;
  width: 100% !important
  height: 100% !important
  object-fit: contain;
  background black
}

.video-presenter-element-cover {
  object-fit: cover;
}

.presenter-element-caption-speaking {
  text-shadow: -2px 0 #9cffa1, 0 2px #9cffa1, 2px 0 #9cffa1, 0 -2px #9cffa1;
}

.presenter-element-caption {
  max-width: calc(100% - 1em) // still needed for thin (vertical) video on mobile - it prevents bulging
}

</style>

<style lang="stylus">
.presenter-unpin-button {
  position absolute
  top 0
  right 0
}

.pane-presenter-vertical {
  .presenter-unpin-button {
    right 2px
  }
}

.pane-presenter-vertical-mobile {
  .presenter-unpin-button {
    right 30px
  }
}
</style>
