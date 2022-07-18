<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="440" persistent>
            <v-card v-if="show" :disabled="changing" :loading="changing">
                <v-card-title>{{ $vuetify.lang.t('$vuetify.video_settings') }}</v-card-title>

                <v-card-text class="px-4 py-0">
                    <v-row no-gutters>
                        <v-col
                        >
                            <v-checkbox
                                dense
                                v-model="audioPresents"
                                @change="changeAudioPresents"
                                :label="$vuetify.lang.t('$vuetify.video_i_have_microphone')"
                            ></v-checkbox>
                        </v-col>

                        <v-col
                        >
                            <v-checkbox
                                dense
                                v-model="videoPresents"
                                @change="changeVideoPresents"
                                :label="$vuetify.lang.t('$vuetify.video_i_have_videocamera')"
                            ></v-checkbox>
                        </v-col>
                    </v-row>

                    <v-select
                        :disabled="serverPreferredVideoResolution"
                        :messages="$vuetify.lang.t('$vuetify.video_resolution')"
                        :items="qualityItems"
                        dense
                        solo
                        @change="changeVideoResolution"
                        v-model="videoQuality"
                    ></v-select>

                    <v-select
                        :disabled="serverPreferredScreenResolution"
                        :messages="$vuetify.lang.t('$vuetify.screen_resolution')"
                        :items="qualityItems"
                        dense
                        solo
                        @change="changeScreenResolution"
                        v-model="screenQuality"
                    ></v-select>

                    <v-row no-gutters>
                        <v-col
                        >
                            <v-checkbox
                                dense
                                :disabled="serverPreferredVideoSimulcast"
                                v-model="videoSimulcast"
                                @change="changeVideoSimulcast"
                                :label="$vuetify.lang.t('$vuetify.video_simulcast')"
                            ></v-checkbox>
                        </v-col>

                        <v-col
                        >
                            <v-checkbox
                                dense
                                :disabled="serverPreferredScreenSimulcast"
                                v-model="screenSimulcast"
                                @change="changeScreenSimulcast"
                                :label="$vuetify.lang.t('$vuetify.screen_simulcast')"
                            ></v-checkbox>
                        </v-col>
                    </v-row>

                    <v-row no-gutters>
                        <v-col
                        >
                            <v-checkbox
                                dense
                                :disabled="serverPreferredRoomDynacast"
                                v-model="roomDynacast"
                                @change="changeRoomDynacast"
                                :label="$vuetify.lang.t('$vuetify.room_dynacast')"
                            ></v-checkbox>
                        </v-col>

                        <v-col
                        >
                            <v-checkbox
                                dense
                                :disabled="serverPreferredRoomAdaptiveStream"
                                v-model="roomAdaptiveStream"
                                @change="changeRoomAdaptiveStream"
                                :label="$vuetify.lang.t('$vuetify.room_adaptive_stream')"
                            ></v-checkbox>
                        </v-col>

                    </v-row>

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
        OPEN_VIDEO_SETTINGS,
        REQUEST_CHANGE_VIDEO_PARAMETERS,
        VIDEO_PARAMETERS_CHANGED,
    } from "./bus";
    import {
        getVideoResolution,
        setVideoResolution,
        getStoredAudioDevicePresents,
        setStoredAudioPresents,
        getStoredVideoDevicePresents,
        setStoredVideoPresents,
        hasLength,
        setScreenResolution,
        getScreenResolution,
        setStoredVideoSimulcast,
        setStoredScreenSimulcast,
        getStoredRoomDynacast,
        getStoredVideoSimulcast,
        getStoredScreenSimulcast, setStoredRoomDynacast, isSet, setStoredRoomAdaptiveStream, getStoredRoomAdaptiveStream
    } from "./utils";
    import axios from "axios";
    import {videochat_name} from "./routes";

    export default {
        data () {
            return {
                changing: false,
                show: false,

                audioPresents: null,
                videoPresents: null,

                serverPreferredVideoResolution: false,
                serverPreferredScreenResolution: false,
                videoQuality: null,
                screenQuality: null,

                serverPreferredVideoSimulcast: false,
                serverPreferredScreenSimulcast: false,
                videoSimulcast: true,
                screenSimulcast: true,

                serverPreferredRoomDynacast: false,
                roomDynacast: true,

                serverPreferredRoomAdaptiveStream: false,
                roomAdaptiveStream: true,
            }
        },
        methods: {
            showModal() {
                this.audioPresents = getStoredAudioDevicePresents();
                this.videoPresents = getStoredVideoDevicePresents();

                this.videoQuality = getVideoResolution();
                this.screenQuality = getScreenResolution();
                this.serverPreferredVideoResolution = false;
                this.serverPreferredScreenResolution = false;

                this.videoSimulcast = getStoredVideoSimulcast();
                this.screenSimulcast = getStoredScreenSimulcast()
                this.serverPreferredVideoSimulcast = false;
                this.serverPreferredScreenSimulcast = false;

                this.roomDynacast = getStoredRoomDynacast();
                this.serverPreferredRoomDynacast = false;

                this.roomAdaptiveStream = getStoredRoomAdaptiveStream();
                this.serverPreferredRoomAdaptiveStream = false;

                this.show = true;
                axios
                    .get(`/api/video/${this.chatId}/config`)
                    .then(response => response.data)
                    .then(respData => {
                        if (hasLength(respData.videoResolution)) {
                            this.serverPreferredVideoResolution = true;
                            this.videoQuality = respData.videoResolution;
                        }
                        if (hasLength(respData.screenResolution)) {
                            this.serverPreferredScreenResolution = true;
                            this.screenQuality = respData.screenResolution;
                        }
                        if (isSet(respData.videoSimulcast)) {
                            this.serverPreferredVideoSimulcast = true;
                            this.videoSimulcast = respData.videoSimulcast;
                        }
                        if (isSet(respData.screenSimulcast)) {
                            this.serverPreferredScreenSimulcast = true;
                            this.screenSimulcast = respData.screenSimulcast;
                        }
                        if (isSet(respData.roomDynacast)) {
                            this.serverPreferredRoomDynacast = true;
                            this.roomDynacast = respData.roomDynacast;
                        }
                        if (isSet(respData.roomAdaptiveStream)) {
                            this.serverPreferredRoomAdaptiveStream = true;
                            this.roomAdaptiveStream = respData.roomAdaptiveStream;
                        }
                    })

            },
            closeModal() {
                this.show = false;
            },
            isVideoRoute() {
                return this.$route.name == videochat_name
            },
            onVideoParametersChanged() {
                if (!this.show) {
                    return
                }
                this.changing = false;
            },
            changeVideoResolution(newVideoResolution) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setVideoResolution(newVideoResolution);
                bus.$emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeScreenResolution(newVideoResolution) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setScreenResolution(newVideoResolution);
                bus.$emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeAudioPresents(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredAudioPresents(v);
                bus.$emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeVideoPresents(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredVideoPresents(v);
                bus.$emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeVideoSimulcast(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredVideoSimulcast(v);
                bus.$emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeScreenSimulcast(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredScreenSimulcast(v);
                bus.$emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeRoomDynacast(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredRoomDynacast(v);
                bus.$emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeRoomAdaptiveStream(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredRoomAdaptiveStream(v);
                bus.$emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
        },
        computed: {
            qualityItems() {
                // ./frontend/node_modules/livekit-client/dist/room/track/options.d.ts
                return ['h180', 'h360', 'h720', 'h1080', 'h1440', 'h2160']
            },
            chatId() {
                return this.$route.params.id
            },
        },
        created() {
            bus.$on(OPEN_VIDEO_SETTINGS, this.showModal);
            bus.$on(VIDEO_PARAMETERS_CHANGED, this.onVideoParametersChanged)
        },
        destroyed() {
            bus.$off(OPEN_VIDEO_SETTINGS, this.showModal);
            bus.$off(VIDEO_PARAMETERS_CHANGED, this.onVideoParametersChanged)
        },
    }
</script>