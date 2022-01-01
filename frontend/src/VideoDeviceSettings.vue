<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="440" persistent>
            <v-card v-if="show" :disabled="changing" :loading="changing">
                <v-card-title>{{ $vuetify.lang.t('$vuetify.video_settings') }}</v-card-title>

                <v-card-text class="px-4 py-0">
                    <v-select
                        messages="Video device"
                        :items="videoDevices"
                        item-text="label"
                        item-value="deviceId"
                        label="Video device"
                        @change="changeVideoDevice"
                        dense
                        solo
                        v-model="videoDevice"
                    ></v-select>
                    <v-select
                        messages="Audio device"
                        :items="audioDevices"
                        item-text="label"
                        item-value="deviceId"
                        label="Audio device"
                        dense
                        solo
                        v-model="audioDevice"
                    ></v-select>
                </v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn color="error" class="mr-4" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                    <v-spacer/>
                </v-card-actions>

            </v-card>

        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {
        CHANGE_DEVICE,
        DEVICE_CHANGED,
        OPEN_DEVICE_SETTINGS,
    } from "./bus";
    import {videochat_name} from "./routes";

    export default {
        data () {
            return {
                changing: false,
                show: false,
                videoDevices: [],
                videoDevice: null,
                audioDevices: [],
                audioDevice: null,
                elementIdToProcess: null,
            }
        },
        methods: {
            showModal(elementIdToProcess) {
                this.show = true;
                this.elementIdToProcess = elementIdToProcess;
                this.requestVideoDeviceItems()
            },
            closeModal() {
                this.show = false;
                this.videoDevices = [];
                this.videoDevice = null;
                this.audioDevices = [];
                this.audioDevice = null;
                this.elementIdToProcess = null;
            },
            isVideoRoute() {
                return this.$route.name == videochat_name
            },
            requestVideoDeviceItems() {
                navigator.mediaDevices.enumerateDevices()
                    .then((devices) => {
                        devices.forEach((device) => {
                            console.log(device.kind + ": " + device.label + " id = " + device.deviceId);
                            if (device.kind == 'videoinput') {
                                this.videoDevices.push(device);
                            }
                            if (device.kind == 'audioinput') {
                                this.audioDevices.push(device);
                            }
                        });
                    })
                    .catch((err) => {
                        console.log(err.name + ": " + err.message);
                    });
            },
            onVideoDeviceChanged() {
                if (!this.show) {
                    return
                }
                this.changing = false;
            },
            changeVideoDevice(newVideoDeviceId) {
                console.log("Invoked changeVideoDevice");
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                bus.$emit(CHANGE_DEVICE, {kind: 'video', deviceId: newVideoDeviceId, elementIdToProcess: this.elementIdToProcess});
            },

        },
        computed: {
            chatId() {
                return this.$route.params.id
            },
        },
        created() {
            bus.$on(OPEN_DEVICE_SETTINGS, this.showModal);
            bus.$on(DEVICE_CHANGED, this.onVideoDeviceChanged)
        },
        destroyed() {
            bus.$off(OPEN_DEVICE_SETTINGS, this.showModal);
            bus.$off(DEVICE_CHANGED, this.onVideoDeviceChanged)
        },
    }
</script>