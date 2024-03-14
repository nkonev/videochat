<template>
        <v-dialog v-model="show" max-width="440" persistent>
            <v-card v-if="show" :title="$vuetify.locale.t('$vuetify.source_add')">

                <v-card-text class="pb-0">
                    <v-select
                        :items="videoDevices"
                        item-title="label"
                        item-value="deviceId"
                        label="Select video device"
                        v-model="videoDevice"
                        variant="outlined"
                        density="compact"
                    ></v-select>
                    <v-select
                        :items="audioDevices"
                        item-title="label"
                        item-value="deviceId"
                        label="Select audio device"
                        v-model="audioDevice"
                        variant="outlined"
                        density="compact"
                    ></v-select>
                </v-card-text>

                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="flat" v-if="videoDevice != null || audioDevice != null" color="primary" @click="addSource()">{{ $vuetify.locale.t('$vuetify.ok') }}</v-btn>
                    <v-btn variant="flat" color="red" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>

            </v-card>

        </v-dialog>
</template>

<script>
    import bus, {
        ADD_VIDEO_SOURCE,
        ADD_VIDEO_SOURCE_DIALOG,
    } from "./bus/bus";

    export default {
        data () {
            return {
                show: false,
                videoDevices: [],
                videoDevice: null, // actually contains id due to v-select configuration
                audioDevices: [],
                audioDevice: null, // actually contains id due to v-select configuration
            }
        },
        methods: {
            showModal() {
                this.show = true;
                this.requestVideoDeviceItems()
            },
            closeModal() {
                this.show = false;
                this.videoDevices = [];
                this.videoDevice = null;
                this.audioDevices = [];
                this.audioDevice = null;
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
            addSource() {
                bus.emit(ADD_VIDEO_SOURCE, {videoId: this.videoDevice, audioId: this.audioDevice});
                this.closeModal();
            }
        },
        mounted() {
            bus.on(ADD_VIDEO_SOURCE_DIALOG, this.showModal);
        },
        beforeUnmount() {
            bus.off(ADD_VIDEO_SOURCE_DIALOG, this.showModal);
        },
    }
</script>
