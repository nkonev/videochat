<template>

  <v-dialog
    v-model="show"
    scrollable
    persistent
    width="fit-content" max-width="100%"
  >
      <v-card>
          <v-sheet elevation="6">
              <v-tabs
                  v-model="tab"
                  bg-color="indigo"
                  @update:modelValue="onUpdateTab"
              >
                  <v-tab value="video">{{$vuetify.locale.t('$vuetify.video')}}</v-tab>
                  <v-tab value="audio">{{$vuetify.locale.t('$vuetify.audio')}}</v-tab>
              </v-tabs>
          </v-sheet>

          <v-card-text class="ma-0 pa-0 wrapper">
              <v-window v-model="tab">
                  <v-window-item value="video">
                      <v-card-text class="d-flex justify-start pb-0 pt-2 px-2 recording-wrapper">
                          <div class="recording-container-element">
                              <div v-if="isRecording" class="inline-caption-base"><v-icon color="red">mdi-record</v-icon>{{recordingLabel}}</div>
                              <span v-if="!mediaDevicesGotten">{{ $vuetify.locale.t('$vuetify.waiting_for_devices') }}</span>
                          </div>
                          <video style="max-width: 100%; max-height: 100%" playsinline></video>
                      </v-card-text>
                  </v-window-item>

                  <v-window-item value="audio">
                      <v-card-text class="d-flex justify-start pb-0 pt-2 px-2 recording-wrapper">
                          <div v-if="isRecording"><v-icon color="red">mdi-record</v-icon>{{recordingLabel}}</div>
                          <span v-if="!mediaDevicesGotten">{{ $vuetify.locale.t('$vuetify.waiting_for_devices') }}</span>
                          <audio class="audio-custom-class" playsinline></audio>
                      </v-card-text>
                  </v-window-item>
              </v-window>
          </v-card-text>
          <v-card-actions>
              <v-spacer/>
              <v-btn variant="outlined" @click="openSettings()" :disabled="isRecording" min-width="0" :title="$vuetify.locale.t('$vuetify.settings')"><v-icon size="large">mdi-cog</v-icon></v-btn>
              <v-btn v-if="mediaDevicesGotten" :color="blob ? null : 'primary'" :variant="blob ? 'outlined' : 'flat'" @click="onClick()"><v-icon size="x-large">{{isRecording ? 'mdi-stop' : 'mdi-record'}}</v-icon> {{ isRecording ? $vuetify.locale.t('$vuetify.stop_recording') : $vuetify.locale.t('$vuetify.start_recording') }} </v-btn>
              <v-btn v-if="mediaDevicesGotten" :color="blob ? 'primary' : null" :variant="blob ? 'flat' : 'outlined'" @click="onAddToMessage()" :disabled="!blob">{{ $vuetify.locale.t('$vuetify.add_to_message') }}</v-btn>
              <v-btn color="red" variant="flat" @click="closeModal()" :disabled="isRecording">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
          </v-card-actions>
      </v-card>
  </v-dialog>
</template>

<script>
import {
    getStoredRecordingAudioDeviceId,
    getStoredRecordingVideoDeviceId,
    getStoreRecordingTab, setStoredRecordingAudioDeviceId, setStoredRecordingVideoDeviceId,
    setStoreRecordingTab
} from "@/store/localStore.js";
import bus, {
    OPEN_RECORDING_MODAL,
    FILE_UPLOAD_MODAL_START_UPLOADING,
    OPEN_FILE_UPLOAD_MODAL,
    MESSAGE_EDIT_SET_FILE_ITEM_UUID, CHANGE_VIDEO_SOURCE, CHANGE_VIDEO_SOURCE_DIALOG
} from "@/bus/bus";
import {mapStores} from "pinia";
import {fileUploadingSessionTypeMedia, useChatStore} from "@/store/chatStore";
import {RecordRTCPromisesHandler} from "recordrtc";
import {v4 as uuidv4} from "uuid";
import {PURPOSE_RECORDING} from "@/utils.js";

export default {
  data () {
    return {
      tab: null,
      show: false,

      videoElement: null,
      recorder: null,
      isRecording: false,
      stream: null,
      blob: null,
      recordingCounter: 0,
      recordingLabel: "",
      recordingLabelUpdateInterval: null,
      mediaDevicesGotten: false,
      fileItemUuid: null,
      overrideVideoDeviceId: null,
      overrideAudioDeviceId: null,
    }
  },
  methods: {
    showModal({fileItemUuid}) {
        this.tab = getStoreRecordingTab('video');
        this.$data.show = true;
        this.fileItemUuid = fileItemUuid;
        this.onShow();
    },
    onUpdateTab(tab) {
        this.onClose();

        console.debug("Setting tab", tab);
        setStoreRecordingTab(tab);

        this.onShow();
    },
    onFileItemUuid({fileItemUuid, chatId}) {
      if (chatId == this.chatId) {
          this.fileItemUuid = fileItemUuid;
      }
    },
    closeModal() {
      this.$data.show = false;
      this.tab = null;
      this.mediaDevicesGotten = false;
      this.fileItemUuid = null;
      this.overrideVideoDeviceId = null;
      this.overrideAudioDeviceId = null;
      this.onClose();
    },
    onClose() {
      this.stream?.getTracks().forEach(function(track) {
          track.stop();
      });
      this.stream = null;
      this.blob = null;
      this.recordingCounter = 0;
      this.recordingLabel = "";
      this.recordingLabelUpdateInterval = null;
    },
    isVideo() {
        return this.tab == 'video'
    },
    async stopRecording() {
      try {
          this.recordingCounter = 0;
          this.recordingLabel = "";
          clearInterval(this.recordingLabelUpdateInterval);
          this.recordingLabelUpdateInterval = null;

          await this.recorder.stopRecording();

          this.videoElement.srcObject = null;
          this.videoElement.autoplay = false;

          this.blob = await this.recorder.getBlob();
          this.videoElement.src = URL.createObjectURL(this.blob);
          this.recorder.stream.getTracks(t => t.stop());
      } finally {
          // reset recorder's state
          await this.recorder.reset();

          // clear the memory
          await this.recorder.destroy();

          // so that we can record again
          this.recorder = null;
          this.isRecording = false;

          // reset
          this.videoElement.muted = false;
          this.videoElement.controls = true;
      }
    },
    async onShow() {
      await this.$nextTick();

      const video = this.overrideVideoDeviceId ? {
            deviceId: this.overrideVideoDeviceId
      } : true;

      const audio = this.overrideAudioDeviceId ? {
            deviceId: this.overrideAudioDeviceId,
            echoCancellation: true,
            noiseSuppression: true,
      } : true;

      if (this.isVideo()) {
          this.videoElement = document.querySelector('.recording-wrapper video');
          this.stream = await navigator.mediaDevices.getUserMedia({video: video, audio: audio});
      } else {
          this.videoElement = document.querySelector('.recording-wrapper audio');
          this.stream = await navigator.mediaDevices.getUserMedia({video: false, audio: audio});
      }
      this.mediaDevicesGotten = true;

      // set
      this.videoElement.muted = true;
      this.videoElement.controls = false;
      this.videoElement.autoplay = true;
      this.videoElement.srcObject = this.stream;
    },
    async startRecording() {
      this.isRecording = true;

      // set
      this.videoElement.muted = true;
      this.videoElement.autoplay = true;
      this.videoElement.controls = false;
      this.videoElement.srcObject = this.stream;

      this.blob = null;

      if (this.isVideo()) {
          this.recorder = new RecordRTCPromisesHandler(this.stream, {
              type: 'video',
              mimeType: 'video/webm',
          });
      } else {
          this.recorder = new RecordRTCPromisesHandler(this.stream, {
              type: 'audio',
              checkForInactiveTracks: true,
              bufferSize: 16384
          });
      }
      const getCurrentTime = ()=>{
          return Math.round(+new Date()/1000)
      }
      this.recordingCounter = getCurrentTime();
      this.recordingLabelUpdateInterval = setInterval(()=>{
          const delta = getCurrentTime() - this.recordingCounter;
          this.recordingLabel = "" + delta + " " + this.$vuetify.locale.t('$vuetify.seconds')
      }, 500)
      await this.recorder.startRecording();

      // helps releasing camera on stopRecording
      this.recorder.stream = this.stream;
    },
    onClick() {
      if (this.isRecording) {
          this.stopRecording();
      } else {
          this.startRecording();
      }
    },
    makeFiles() {
      if (this.isVideo()) {
          const file = new File([this.blob], "video.webm", {
              type: 'video/webm'
          });
          return [file]
      } else {
          const file = new File([this.blob], "audio.mp3", {
              type: 'audio/mp3'
          });
          return [file]
      }
    },
    onAddToMessage() {
      this.chatStore.correlationId = uuidv4();
      const files = this.makeFiles();
      bus.emit(OPEN_FILE_UPLOAD_MODAL, {showFileInput: true, shouldSetFileUuidToMessage: true, fileItemUuid: this.fileItemUuid, predefinedFiles: files, correlationId: this.chatStore.correlationId, shouldAddDateToTheFilename: true, fileUploadingSessionType: fileUploadingSessionTypeMedia, isMessageRecording: true});
      bus.emit(FILE_UPLOAD_MODAL_START_UPLOADING);

      this.closeModal();
    },
    openSettings() {
        bus.emit(CHANGE_VIDEO_SOURCE_DIALOG, PURPOSE_RECORDING);
    },
    onChangeVideoSource({videoId, audioId, purpose}) {
        if (this.show && purpose == PURPOSE_RECORDING) {
            this.overrideVideoDeviceId = videoId;
            this.overrideAudioDeviceId = audioId;
            setStoredRecordingVideoDeviceId(this.overrideVideoDeviceId);
            setStoredRecordingAudioDeviceId(this.overrideAudioDeviceId);
            this.onUpdateTab(this.tab)
        }
    },
  },
  watch: {
    show(newValue) {
        if (!newValue) {
            this.closeModal();
        }
    },
  },
  computed: {
    ...mapStores(useChatStore),
    chatId() {
      return this.$route.params.id
    },
  },
  beforeUnmount() {
    bus.off(OPEN_RECORDING_MODAL, this.showModal);
    bus.off(MESSAGE_EDIT_SET_FILE_ITEM_UUID, this.onFileItemUuid);
    bus.off(CHANGE_VIDEO_SOURCE, this.onChangeVideoSource);
  },
  mounted() {
    bus.on(OPEN_RECORDING_MODAL, this.showModal);
    bus.on(MESSAGE_EDIT_SET_FILE_ITEM_UUID, this.onFileItemUuid);
    bus.on(CHANGE_VIDEO_SOURCE, this.onChangeVideoSource);

    this.overrideVideoDeviceId = getStoredRecordingVideoDeviceId();
    this.overrideAudioDeviceId = getStoredRecordingAudioDeviceId();
  }
}
</script>

<style lang="stylus" scoped>
.recording-container-element {
    position relative
    display flex
    flex-direction column
    align-items: center;
}

</style>
