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
                      <v-card-text class="d-flex justify-center pb-0 pt-2 px-2 recording-wrapper">
                          <video class="video-custom-class" playsinline></video>
                      </v-card-text>
                  </v-window-item>

                  <v-window-item value="audio">
                      <v-card-text class="d-flex justify-center pb-0 pt-2 px-2 recording-wrapper">
                          <audio class="audio-custom-class" playsinline></audio>
                      </v-card-text>
                  </v-window-item>
              </v-window>
          </v-card-text>
          <v-card-actions>
              <v-spacer/>
              <v-btn :color="blob ? null : 'primary'" :variant="blob ? 'outlined' : 'flat'" @click="onClick()"><v-icon size="x-large">{{isRecording ? 'mdi-stop' : 'mdi-record'}}</v-icon> {{ isRecording ? $vuetify.locale.t('$vuetify.stop_recording') : $vuetify.locale.t('$vuetify.start_recording') }} </v-btn>
              <v-btn :color="blob ? 'primary' : null" :variant="blob ? 'flat' : 'outlined'" @click="onAddToMessage()" :disabled="!blob">{{ $vuetify.locale.t('$vuetify.add_to_message') }}</v-btn>
              <v-btn color="red" variant="flat" @click="closeModal()" :disabled="isRecording">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
          </v-card-actions>
      </v-card>
  </v-dialog>
</template>

<script>
import {getStoreRecordingTab, setStoreRecordingTab} from "@/store/localStore.js";
import bus, {OPEN_RECORDING_MODAL, CORRELATION_ID_SET, FILE_UPLOAD_MODAL_START_UPLOADING, OPEN_FILE_UPLOAD_MODAL} from "@/bus/bus";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {RecordRTCPromisesHandler} from "recordrtc";
import {v4 as uuidv4} from "uuid";

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
    }
  },
  methods: {
    showModal() {
        this.tab = getStoreRecordingTab('video');
        this.$data.show = true;
        this.onShow();
    },
    onUpdateTab(tab) {
        this.onClose();

        console.debug("Setting tab", tab);
        setStoreRecordingTab(tab);

        this.onShow();
    },
    closeModal() {
      this.$data.show = false;
      this.tab = null;
      this.onClose();
    },
    onClose() {
      this.stream?.getTracks().forEach(function(track) {
          track.stop();
      });
      this.stream = null;
      this.blob = null;
      this.fileItemUuid = null;
      this.correlationId = null;
    },
    isVideo() {
        return this.tab == 'video'
    },
    async stopRecording() {
      try {
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
      if (this.isVideo()) {
          this.videoElement = document.querySelector('.recording-wrapper video');
          this.stream = await navigator.mediaDevices.getUserMedia({video: true, audio: true});
      } else {
          this.videoElement = document.querySelector('.recording-wrapper audio');
          this.stream = await navigator.mediaDevices.getUserMedia({video: false, audio: true});
      }

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
      const correlationId = uuidv4();
      bus.emit(CORRELATION_ID_SET, correlationId);
      const files = this.makeFiles();
      bus.emit(OPEN_FILE_UPLOAD_MODAL, {showFileInput: true, shouldSetFileUuidToMessage: true, predefinedFiles: files, correlationId: correlationId, shouldAddDateToTheFilename: true});
      bus.emit(FILE_UPLOAD_MODAL_START_UPLOADING);

      this.closeModal();
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
  },
  beforeUnmount() {
    bus.off(OPEN_RECORDING_MODAL, this.showModal);
  },
  mounted() {
    bus.on(OPEN_RECORDING_MODAL, this.showModal);
  }
}
</script>
