<template>
  <div :class="videoButtonsControlClass">
    <v-slide-group show-arrows>
      <template v-if="chatStore.showCallManagement && chatStore.isInCall()">
        <v-btn variant="plain" tile icon :loading="chatStore.initializingVideoCall" @click.stop.prevent="stopCall()" :title="$vuetify.locale.t('$vuetify.leave_call')">
          <v-icon size="x-large" :class="chatStore.shouldPhoneBlink ? 'call-blink' : 'text-red'">mdi-phone</v-icon>
        </v-btn>
      </template>
      <v-btn variant="plain" tile icon v-if="chatStore.canShowMicrophoneButton" @click.stop.prevent="doMuteAudio(!chatStore.localMicrophoneEnabled)" :title="!chatStore.localMicrophoneEnabled ? $vuetify.locale.t('$vuetify.unmute_audio') : $vuetify.locale.t('$vuetify.mute_audio')"><v-icon size="x-large">{{ !chatStore.localMicrophoneEnabled ? 'mdi-microphone-off' : 'mdi-microphone' }}</v-icon></v-btn>
      <v-btn variant="plain" tile icon v-if="chatStore.canShowVideoButton" @click.stop.prevent="doMuteVideo(!chatStore.localVideoEnabled)" :title="!chatStore.localVideoEnabled ? $vuetify.locale.t('$vuetify.unmute_video') : $vuetify.locale.t('$vuetify.mute_video')"><v-icon size="x-large">{{ !chatStore.localVideoEnabled ? 'mdi-video-off' : 'mdi-video' }} </v-icon></v-btn>
      <template v-if="!isMobile()">
        <v-btn variant="plain" tile icon @click.stop.prevent="addScreenSource()" :title="$vuetify.locale.t('$vuetify.screen_share')">
          <v-icon size="x-large">mdi-monitor-screenshot</v-icon>
        </v-btn>
      </template>
      <v-btn variant="plain" tile icon @click.stop.prevent="flipMessages" :title="$vuetify.locale.t('$vuetify.messages')"><v-icon size="x-large">mdi-message-text-outline</v-icon></v-btn>

      <v-btn variant="plain" tile icon @click.stop.prevent="onEnterFullscreen" :title="$vuetify.locale.t('$vuetify.fullscreen')"><v-icon size="x-large">mdi-arrow-expand-all</v-icon></v-btn>

      <v-btn :disabled="videoIsGallery()" tile icon :input-value="presenterValue()" @click="presenterClick" :variant="presenterValue() ? 'tonal' : 'plain'" :title="presenterValue() ? $vuetify.locale.t('$vuetify.video_presenter_disable') : $vuetify.locale.t('$vuetify.video_presenter_enable')"><v-icon size="x-large">mdi-presentation</v-icon></v-btn>

      <v-select
          class="video-position-select"
          :items="positionItems"
          density="compact"
          hide-details
          @update:modelValue="changeVideoPosition"
          v-model="chatStore.videoPosition"
          variant="plain"
      ></v-select>

      <v-btn variant="plain" tile icon @click="addVideoSource()" :title="$vuetify.locale.t('$vuetify.source_add')">
        <v-icon size="x-large">mdi-video-plus</v-icon>
      </v-btn>

      <v-btn variant="plain" tile icon v-if="chatStore.showRecordStartButton" @click="startRecord()" :loading="chatStore.initializingStaringVideoRecord" :title="$vuetify.locale.t('$vuetify.start_record')">
        <v-icon size="x-large">mdi-record-rec</v-icon>
      </v-btn>
      <v-btn variant="plain" tile icon v-if="chatStore.showRecordStopButton" @click="stopRecord()" :loading="chatStore.initializingStoppingVideoRecord" :title="$vuetify.locale.t('$vuetify.stop_record')">
        <v-icon size="x-large" color="red">mdi-stop</v-icon>
      </v-btn>

      <v-btn variant="plain" tile icon @click="openSettings()" :title="$vuetify.locale.t('$vuetify.video_settings')">
        <v-icon size="x-large">mdi-cog</v-icon>
      </v-btn>

    </v-slide-group>
  </div>
</template>

<script>
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore.js";
import videoPositionMixin from "@/mixins/videoPositionMixin.js";
import {stopCall} from "@/utils.js";
import bus, {ADD_SCREEN_SOURCE, ADD_VIDEO_SOURCE_DIALOG, OPEN_SETTINGS} from "@/bus/bus.js";
import {
  positionItems,
  setStoredPresenter,
  setStoredVideoPosition,
} from "@/store/localStore.js";
import axios from "axios";

export default {
  mixins: [
    videoPositionMixin(),
  ],
  data() {
    return {

    }
  },
  computed: {
    ...mapStores(useChatStore),
    videoButtonsControlClass() {
      if (this.videoIsHorizontal() || this.videoIsGallery()) {
        return ["video-buttons-control", "video-buttons-control-horizontal"]
      } else if (this.videoIsVertical())  {
        if (!this.chatStore.presenterEnabled) {
          return ["video-buttons-control", "video-buttons-control-vertical"];
        } else {
          return ["video-buttons-control", "video-buttons-control-horizontal"]
        }
      } else {
        return null;
      }
    },
    positionItems() {
      return positionItems()
    },
    chatId() {
      return this.$route.params.id
    },
  },
  methods: {
    doMuteAudio(requestedState) {
      this.chatStore.localMicrophoneEnabled = requestedState
    },
    doMuteVideo(requestedState) {
      this.chatStore.localVideoEnabled = requestedState
    },
    onEnterFullscreen(e) {
      this.$emit("requestFullScreen");
    },
    flipMessages() {

    },
    stopCall() {
      stopCall(this.chatStore, this.$route, this.$router);
    },
    addScreenSource() {
      bus.emit(ADD_SCREEN_SOURCE);
    },
    changeVideoPosition(v) {
      this.chatStore.videoPosition = v;
      setStoredVideoPosition(v);
    },
    presenterValue() {
      return this.chatStore.presenterEnabled
    },
    presenterClick() {
      const v = !this.chatStore.presenterEnabled;
      this.chatStore.presenterEnabled = v;
      setStoredPresenter(v);
    },
    addVideoSource() {
      bus.emit(ADD_VIDEO_SOURCE_DIALOG);
    },
    startRecord() {
      axios.put(`/api/video/${this.chatId}/record/start`);
      this.chatStore.initializingStaringVideoRecord = true;
    },
    stopRecord() {
      axios.put(`/api/video/${this.chatId}/record/stop`);
      this.chatStore.initializingStoppingVideoRecord = true;
    },
    openSettings() {
      bus.emit(OPEN_SETTINGS, 'a_video_settings') // value matches with SettingsModal.vue :: v-window-item
    },
  }
}
</script>


<style scoped lang="stylus">

.video-buttons-control {
  background rgba(255, 255, 255, 0.65)
  padding-left 0.3em
  padding-right 0.3em
  border-radius 4px
  display: flex;
  max-width 100% // needed for v-slide-group
}

.video-buttons-control-horizontal {
  position: absolute;
  bottom 16px
  z-index 20
}

.video-buttons-control-vertical {
  position: absolute;
  align-self end
  display: flex;
  flex-direction: column;
  z-index 20
  bottom 16px
}

.video-position-select {
  margin-top auto
  margin-bottom auto
  display: inline-flex
  align-self: center
}

</style>

<style lang="stylus">
.video-position-select {
  .v-field__input {
    min-height: unset;
    min-width: unset;
    padding 0 0 0 4px !important
    margin 0 !important
    position: relative;
  }

  div.v-field__append-inner {
      padding 0 !important
  }
}
</style>