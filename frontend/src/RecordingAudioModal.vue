<template>
    <div style="overflow-y: auto">
        <v-card-text class="d-flex justify-center pb-0 pt-2 px-2 recording-wrapper">
            <audio class="audio-custom-class" playsinline></audio>
        </v-card-text>
        <v-card-actions>
            <v-spacer/>
            <v-btn :color="blob ? null : 'primary'" :variant="blob ? 'outlined' : 'flat'" @click="onClick()"><v-icon size="x-large">{{isRecording ? 'mdi-stop' : 'mdi-record'}}</v-icon> {{ isRecording ? $vuetify.locale.t('$vuetify.stop_recording') : $vuetify.locale.t('$vuetify.start_recording') }} </v-btn>
            <v-btn :color="blob ? 'primary' : null" :variant="blob ? 'flat' : 'outlined'" @click="onAddToMessage()" :disabled="!blob">{{ $vuetify.locale.t('$vuetify.add_to_message') }}</v-btn>
            <v-btn color="red" variant="flat" @click="hideModal()" :disabled="isRecording">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
        </v-card-actions>
    </div>
</template>

<script>
import bus, {CORRELATION_ID_SET, FILE_UPLOAD_MODAL_START_UPLOADING, OPEN_FILE_UPLOAD_MODAL} from "@/bus/bus";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {RecordRTCPromisesHandler} from "recordrtc";
import {v4 as uuidv4} from "uuid";

export default {
    data () {
        return {
            show: false,
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
        hideModal() {
            this.onClose();
            this.$emit('closemodal');
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
            this.videoElement = document.querySelector('.recording-wrapper audio');

            this.stream = await navigator.mediaDevices.getUserMedia({video: false, audio: true});

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
                type: 'audio',
                checkForInactiveTracks: true,
                bufferSize: 16384
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
            const file = new File([this.blob], "video.mp3", {
                type: 'audio/mp3'
            });
            return [file]
        },
        onAddToMessage() {
            const correlationId = uuidv4();
            bus.emit(CORRELATION_ID_SET, correlationId);
            const files = this.makeFiles();
            bus.emit(OPEN_FILE_UPLOAD_MODAL, {showFileInput: true, shouldSetFileUuidToMessage: true, predefinedFiles: files, correlationId: correlationId, shouldAddDateToTheFilename: true});
            bus.emit(FILE_UPLOAD_MODAL_START_UPLOADING);

            this.hideModal();
        },
    },
    computed: {
        ...mapStores(useChatStore),
    },
    mounted() {
        this.$nextTick(()=>{
            this.onShow();
        });
    },
}
</script>
