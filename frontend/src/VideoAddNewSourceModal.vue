<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="440" persistent>
            <v-card v-if="show">
                <v-card-title>{{ $vuetify.lang.t('$vuetify.source_add') }}</v-card-title>

                <v-card-text class="px-4 py-0">
                    <v-select
                        messages="Video device"
                        :items="videoDevices"
                        item-text="label"
                        item-value="deviceId"
                        label="Select video device"
                        dense
                        solo
                        v-model="videoDevice"
                    ></v-select>
                    <v-select
                        messages="Audio device"
                        :items="audioDevices"
                        item-text="label"
                        item-value="deviceId"
                        label="Select audio device"
                        dense
                        solo
                        v-model="audioDevice"
                    ></v-select>
                </v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn v-if="videoDevice != null || audioDevice != null" color="primary" class="mr-4" @click="addSource()">{{ $vuetify.lang.t('$vuetify.ok') }}</v-btn>
                    <v-btn color="error" class="mr-4" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                    <v-spacer/>
                </v-card-actions>

            </v-card>

        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {
        ADD_VIDEO_SOURCE,
        ADD_VIDEO_SOURCE_DIALOG,
        OPEN_DEVICE_SETTINGS,
    } from "./bus";

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
                bus.$emit(ADD_VIDEO_SOURCE, this.videoDevice, this.audioDevice);
                this.closeModal();
            }
        },
        created() {
            bus.$on(ADD_VIDEO_SOURCE_DIALOG, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_DEVICE_SETTINGS, this.showModal);
        },
    }
</script>