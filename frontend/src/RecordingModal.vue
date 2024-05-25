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
            >
                <v-tab value="video">{{$vuetify.locale.t('$vuetify.video')}}</v-tab>
                <v-tab value="audio">{{$vuetify.locale.t('$vuetify.audio')}}</v-tab>
            </v-tabs>
        </v-sheet>

        <v-card-text class="ma-0 pa-0 wrapper">
            <v-window v-model="tab">
                <v-window-item value="video">
                    <v-card-text class="d-flex justify-center py-0 px-2">
                        <video class="video-custom-class" playsinline></video>
                    </v-card-text>
                </v-window-item>

                <v-window-item value="audio">
                </v-window-item>
            </v-window>
        </v-card-text>
      <v-card-actions>
        <v-spacer/>
          <v-btn :color="blob ? null : 'primary'" :variant="blob ? 'outlined' : 'flat'" @click="onClick()">{{ isRecording ? $vuetify.locale.t('$vuetify.stop_recording') : $vuetify.locale.t('$vuetify.start_recording') }}</v-btn>
          <v-btn :color="blob ? 'primary' : null" :variant="blob ? 'flat' : 'outlined'" @click="onAddToMessage()" :disabled="!blob">{{ $vuetify.locale.t('$vuetify.add_to_message') }}</v-btn>
          <v-btn color="red" variant="flat" @click="hideModal()" :disabled="isRecording">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
import bus, {FILE_UPLOAD_MODAL_START_UPLOADING, OPEN_FILE_UPLOAD_MODAL, OPEN_RECORDING_MODAL} from "@/bus/bus";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {RecordRTCPromisesHandler} from "recordrtc";

export default {
  data () {
    return {
      tab: null,
      show: false,
      fileItemUuid: null,
      correlationId: null,
      videoElement: null,
      recorder: null,
      isRecording: false,
      stream: null,
      blob: null,
    }
  },
  components: {
  },
  methods: {
    showSettingsModal({fileItemUuid, correlationId}) {
      this.fileItemUuid = fileItemUuid;
      this.correlationId = correlationId;

      this.$data.show = true;
      this.$nextTick(()=>{
          this.onShow();
      })
    },
    hideModal() {
      this.$data.show = false;
      this.onClose();
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
        this.videoElement = document.querySelector('.wrapper video');

        this.stream = await navigator.mediaDevices.getUserMedia({video: true, audio: true});

        // set
        this.videoElement.muted = true;
        this.videoElement.controls = false;
        this.videoElement.autoplay = true;
        this.videoElement.srcObject = this.stream;
    },
    onClose() {
        this.stream.getTracks().forEach(function(track) {
            track.stop();
        });
        this.stream = null;
        this.blob = null;
        this.fileItemUuid = null;
        this.correlationId = null;
    },
    async startRecording() {
        this.isRecording = true;

        // set
        this.videoElement.muted = true;
        this.videoElement.autoplay = true;
        this.videoElement.controls = false;
        this.videoElement.srcObject = this.stream;

        this.blob = null;

        this.recorder = new RecordRTCPromisesHandler(this.stream, {
            type: 'video'
        });
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
        const file = new File([this.blob], "video.webm", {
            type: 'video/webm'
        });
        return [file]
    },
    onAddToMessage() {
        const fileItemId = this.fileItemUuid;
        const correlationId = this.correlationId;
        const files = this.makeFiles();
        bus.emit(OPEN_FILE_UPLOAD_MODAL, {showFileInput: true, fileItemUuid: fileItemId, shouldSetFileUuidToMessage: true, predefinedFiles: files, correlationId: correlationId, shouldAddDateToTheFilename: true});
        bus.emit(FILE_UPLOAD_MODAL_START_UPLOADING);

        this.hideModal();
    },
  },
  computed: {
    ...mapStores(useChatStore),
  },
  created() {
    bus.on(OPEN_RECORDING_MODAL, this.showSettingsModal);
  },
  beforeUnmount() {
    bus.off(OPEN_RECORDING_MODAL, this.showSettingsModal);
  },
  mounted() {
  }
}
</script>
