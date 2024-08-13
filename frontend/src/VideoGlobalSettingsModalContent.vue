<template>

            <v-card-text class="pb-0">

                    <v-row no-gutters class="mb-4">
                        <v-col>
                            <v-checkbox
                                density="comfortable"
                                color="primary"
                                hide-details
                                v-model="audioPresents"
                                @update:modelValue="changeAudioPresents"
                                :label="$vuetify.locale.t('$vuetify.video_i_have_microphone')"
                            ></v-checkbox>
                        </v-col>

                        <v-col>
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

                    <v-btn variant="outlined" class="mb-4" @click="openDevicesSettings()">{{ $vuetify.locale.t('$vuetify.default_devices_for_call') }}</v-btn>

                    <v-select
                        :label="$vuetify.locale.t('$vuetify.video_position')"
                        :items="positionItems"
                        density="comfortable"
                        hide-details
                        color="primary"
                        @update:modelValue="changeVideoPosition"
                        v-model="chatStore.videoPosition"
                        variant="underlined"
                    ></v-select>

                    <v-row no-gutters class="my-4">
                        <v-col>
                            <v-checkbox
                                :disabled="!isPresenterEnabled()"
                                density="comfortable"
                                color="primary"
                                hide-details
                                v-model="chatStore.presenterEnabled"
                                @update:modelValue="changePresenterEnabled"
                                :label="$vuetify.locale.t('$vuetify.video_presenter_enable')"
                            ></v-checkbox>
                        </v-col>
                    </v-row>

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
                        :items="displayQualityItems"
                        density="comfortable"
                        color="primary"
                        @update:modelValue="changeVideoResolution"
                        v-model="videoResolution"
                        variant="underlined"
                    ></v-select>

                    <v-select
                        :disabled="serverPreferredScreenResolution"
                        :label="$vuetify.locale.t('$vuetify.screen_resolution')"
                        :items="screenQualityItems"
                        density="comfortable"
                        color="primary"
                        @update:modelValue="changeScreenResolution"
                        v-model="screenResolution"
                        variant="underlined"
                    ></v-select>

                    <v-row no-gutters>
                        <v-col>
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

                        <v-col>
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
                        <v-col>
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

                        <v-col>
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
      CHANGE_VIDEO_SOURCE,
      CHANGE_VIDEO_SOURCE_DIALOG, CHOOSING_VIDEO_SOURCE_CANCELED,
      REQUEST_CHANGE_VIDEO_PARAMETERS,
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
      setStoredVideoPosition,
      setStoredCodec,
      NULL_CODEC,
      NULL_SCREEN_RESOLUTION,
      setStoredCallVideoDeviceId,
      setStoredCallAudioDeviceId,
      setStoredPresenter,
      positionItems
    } from "./store/localStore";
    import {videochat_name} from "./router/routes";
    import videoServerSettingsMixin from "@/mixins/videoServerSettingsMixin";
    import {PURPOSE_CALL} from "@/utils.js";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore.js";
    import videoPositionMixin from "@/mixins/videoPositionMixin.js";

    export default {
        mixins: [
            videoServerSettingsMixin(),
            videoPositionMixin(),
        ],
        data () {
            return {
                audioPresents: null,
                videoPresents: null,

                tempStream: null,
            }
        },
        methods: {
            showModal() {
                this.audioPresents = getStoredAudioDevicePresents();
                this.videoPresents = getStoredVideoDevicePresents();

                this.initPositionAndPresenter();

                this.initServerData();
            },
            isVideoRoute() {
                return this.$route.name == videochat_name
            },
            changeVideoResolution(newVideoResolution) {
                setVideoResolution(newVideoResolution);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeScreenResolution(newVideoResolution) {
                setScreenResolution(newVideoResolution);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeAudioPresents(v) {
                setStoredAudioPresents(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeVideoPresents(v) {
                setStoredVideoPresents(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changePresenterEnabled(v) {
              setStoredPresenter(v);
            },
            changeVideoSimulcast(v) {
                setStoredVideoSimulcast(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeScreenSimulcast(v) {
                setStoredScreenSimulcast(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeRoomDynacast(v) {
                setStoredRoomDynacast(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeRoomAdaptiveStream(v) {
                setStoredRoomAdaptiveStream(v);
                bus.emit(REQUEST_CHANGE_VIDEO_PARAMETERS);
            },
            changeVideoPosition(v) {
                this.chatStore.videoPosition = v;
                setStoredVideoPosition(v);
            },
            changeCodec(v) {
                setStoredCodec(v)
            },
            async requestPermissions() {
                this.tempStream = await navigator.mediaDevices.getUserMedia({video: true, audio: true});
            },
            requestPermissionsClose() {
                let i = 0;
                if (this.tempStream) {
                    for (const track of this.tempStream.getTracks()) {
                        track.stop();
                        i++;
                    }
                }
                console.info("Closed test "+ i +" tracks");
            },
            async openDevicesSettings() {
                if (!this.isVideoRoute()) {
                    await this.requestPermissions();
                }
                bus.emit(CHANGE_VIDEO_SOURCE_DIALOG, PURPOSE_CALL);
            },
            onChangeVideoSource({videoId, audioId, purpose}) {
                if (!this.isVideoRoute()) {
                    this.requestPermissionsClose();
                    if (purpose === PURPOSE_CALL) {
                        setStoredCallVideoDeviceId(videoId);
                        setStoredCallAudioDeviceId(audioId);
                    }
                }
            },
            onChoosingVideoSourceCanceled() {
              this.requestPermissionsClose();
            },
        },
        computed: {
          ...mapStores(useChatStore),
          displayQualityItems() {
                // ./frontend/node_modules/livekit-client/dist/room/track/options.d.ts
                return ['h180', 'h360', 'h720', 'h1080', 'h1440', 'h2160']
            },
            screenQualityItems() {
                // ./frontend/node_modules/livekit-client/dist/room/track/options.d.ts
                return [NULL_SCREEN_RESOLUTION, 'h180', 'h360', 'h720', 'h1080', 'h1440', 'h2160']
            },
            codecItems() {
                // ./frontend/node_modules/livekit-client/dist/room/track/options.d.ts
                return [NULL_CODEC, 'vp8', 'h264', 'vp9', 'av1']
            },
            positionItems() {
                return positionItems()
            },
            chatId() {
                return this.$route.params.id
            },
        },
        mounted() {
            bus.on(CHANGE_VIDEO_SOURCE, this.onChangeVideoSource);
            bus.on(CHOOSING_VIDEO_SOURCE_CANCELED, this.onChoosingVideoSourceCanceled);

            this.showModal();
        },
        created() {
        },
        beforeUnmount() {
            bus.off(CHANGE_VIDEO_SOURCE, this.onChangeVideoSource);
            bus.off(CHOOSING_VIDEO_SOURCE_CANCELED, this.onChoosingVideoSourceCanceled);
        },
    }
</script>
