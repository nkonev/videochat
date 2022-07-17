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
                        :messages="$vuetify.lang.t('$vuetify.videoResolution')"
                        :items="qualityItems"
                        dense
                        solo
                        @change="changeVideoResolution"
                        v-model="videoQuality"
                    ></v-select>

                    <v-select
                        :disabled="serverPreferredScreenResolution"
                        :messages="$vuetify.lang.t('$vuetify.screenResolution')"
                        :items="qualityItems"
                        dense
                        solo
                        @change="changeScreenResolution"
                        v-model="screenQuality"
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
        setStoredVideoPresents, hasLength, setScreenResolution, getScreenResolution
    } from "./utils";
    import axios from "axios";
    import {videochat_name} from "./routes";

    export default {
        data () {
            return {
                changing: false,
                show: false,
                serverPreferredVideoResolution: false,
                serverPreferredScreenResolution: false,

                audioPresents: null,
                videoPresents: null,
                videoQuality: null,
                screenQuality: null,
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