<template>

            <v-card-text class="pb-0">

                    <v-row no-gutters class="mb-4">
                        <v-col
                        >
                            <v-checkbox
                                density="comfortable"
                                color="primary"
                                hide-details
                                v-model="audioPresents"
                                @update:modelValue="changeAudioPresents"
                                :label="$vuetify.locale.t('$vuetify.video_i_have_microphone')"
                            ></v-checkbox>
                        </v-col>

                        <v-col
                        >
                            <v-checkbox
                                density="comfortable"
                                color="primary"
                                hide-details
                                v-model="videoPresents"
                                @update:modelValue="changeVideoPresents"
                                :label="$vuetify.locale.t('$vuetify.video_i_have_videocamera')"
                            ></v-checkbox>
                        </v-col>
                    </v-row>

                    <v-select
                        :label="$vuetify.locale.t('$vuetify.video_position')"
                        :items="positionItems"
                        density="comfortable"
                        color="primary"
                        @update:modelValue="changeVideoPosition"
                        v-model="videoPosition"
                        variant="underlined"
                    ></v-select>


                    <v-select
                        :disabled="serverPreferredCodec"
                        :label="$vuetify.locale.t('$vuetify.codec')"
                        :items="codecItems"
                        density="comfortable"
                        color="primary"
                        @update:modelValue="changeCodec"
                        v-model="codec"
                        variant="underlined"
                    ></v-select>

                    <v-select
                        :disabled="serverPreferredVideoResolution"
                        :label="$vuetify.locale.t('$vuetify.video_resolution')"
                        :items="qualityItems"
                        density="comfortable"
                        color="primary"
                        @update:modelValue="changeVideoResolution"
                        v-model="videoResolution"
                        variant="underlined"
                    ></v-select>

                    <v-select
                        :disabled="serverPreferredScreenResolution"
                        :label="$vuetify.locale.t('$vuetify.screen_resolution')"
                        :items="qualityItems"
                        density="comfortable"
                        color="primary"
                        @update:modelValue="changeScreenResolution"
                        v-model="screenResolution"
                        variant="underlined"
                    ></v-select>

                    <v-row no-gutters>
                        <v-col
                        >
                            <v-checkbox
                                density="comfortable"
                                color="primary"
                                hide-details
                                :disabled="serverPreferredVideoSimulcast"
                                v-model="videoSimulcast"
                                @update:modelValue="changeVideoSimulcast"
                                :label="$vuetify.locale.t('$vuetify.video_simulcast')"
                            ></v-checkbox>
                        </v-col>

                        <v-col
                        >
                            <v-checkbox
                                density="comfortable"
                                color="primary"
                                hide-details
                                :disabled="serverPreferredScreenSimulcast"
                                v-model="screenSimulcast"
                                @update:modelValue="changeScreenSimulcast"
                                :label="$vuetify.locale.t('$vuetify.screen_simulcast')"
                            ></v-checkbox>
                        </v-col>
                    </v-row>

                    <v-row no-gutters>
                        <v-col
                        >
                            <v-checkbox
                                density="comfortable"
                                color="primary"
                                hide-details
                                :disabled="serverPreferredRoomDynacast"
                                v-model="roomDynacast"
                                @update:modelValue="changeRoomDynacast"
                                :label="$vuetify.locale.t('$vuetify.room_dynacast')"
                            ></v-checkbox>
                        </v-col>

                        <v-col
                        >
                            <v-checkbox
                                density="comfortable"
                                color="primary"
                                hide-details
                                :disabled="serverPreferredRoomAdaptiveStream"
                                v-model="roomAdaptiveStream"
                                @update:modelValue="changeRoomAdaptiveStream"
                                :label="$vuetify.locale.t('$vuetify.room_adaptive_stream')"
                            ></v-checkbox>
                        </v-col>
                    </v-row>
            </v-card-text>

</template>

<script>
    import bus, {
        REQUEST_CHANGE_VIDEO_PARAMETERS,
        VIDEO_PARAMETERS_CHANGED,
    } from "./bus/bus";
    import {
        setVideoResolution,
        getStoredAudioDevicePresents,
        setStoredAudioPresents,
        getStoredVideoDevicePresents,
        setStoredVideoPresents,
        setScreenResolution,
        setStoredVideoSimulcast,
        setStoredScreenSimulcast,
        setStoredRoomDynacast,
        setStoredRoomAdaptiveStream,
        VIDEO_POSITION_AUTO,
        VIDEO_POSITION_ON_THE_TOP,
        VIDEO_POSITION_SIDE,
        setStoredVideoPosition, getStoredVideoPosition, setStoredCodec
    } from "./store/localStore";
    import {videochat_name} from "./router/routes";
    import videoServerSettingsMixin from "@/mixins/videoServerSettingsMixin";

    export default {
        mixins: [videoServerSettingsMixin()],
        data () {
            return {
                changing: false,

                audioPresents: null,
                videoPresents: null,
                videoPosition: null,
            }
        },
        methods: {
            showModal() {
                this.audioPresents = getStoredAudioDevicePresents();
                this.videoPresents = getStoredVideoDevicePresents();
                this.videoPosition = getStoredVideoPosition();

                this.initServerData();

            },
            isVideoRoute() {
                return this.$route.name == videochat_name
            },
            onVideoParametersChanged() {
                this.changing = false;
            },
            changeVideoResolution(newVideoResolution) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setVideoResolution(newVideoResolution);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeScreenResolution(newVideoResolution) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setScreenResolution(newVideoResolution);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeAudioPresents(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredAudioPresents(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeVideoPresents(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredVideoPresents(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeVideoSimulcast(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredVideoSimulcast(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeScreenSimulcast(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredScreenSimulcast(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeRoomDynacast(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredRoomDynacast(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeRoomAdaptiveStream(v) {
                if (this.isVideoRoute()) {
                    this.changing = true;
                }
                setStoredRoomAdaptiveStream(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeVideoPosition(v) {
                setStoredVideoPosition(v)
            },
            changeCodec(v) {
                setStoredCodec(v)
            },
        },
        computed: {
            qualityItems() {
                // ./frontend/node_modules/livekit-client/dist/room/track/options.d.ts
                return ['h180', 'h360', 'h720', 'h1080', 'h1440', 'h2160']
            },
            codecItems() {
                // ./frontend/node_modules/livekit-client/dist/room/track/options.d.ts
                return ['null', 'vp8', 'h264', 'vp9', 'av1']
            },
            positionItems() {
                return [VIDEO_POSITION_AUTO, VIDEO_POSITION_ON_THE_TOP, VIDEO_POSITION_SIDE]
            },
            chatId() {
                return this.$route.params.id
            },
        },
        mounted() {
            this.showModal()
        },
        created() {
            bus.on(VIDEO_PARAMETERS_CHANGED, this.onVideoParametersChanged)
        },
        beforeUnmount() {
            bus.off(VIDEO_PARAMETERS_CHANGED, this.onVideoParametersChanged)
        },
    }
</script>
