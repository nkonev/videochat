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
                        :messages="$vuetify.lang.t('$vuetify.quality')"
                        :items="qualityItems"
                        dense
                        solo
                        @change="changeVideoResolution"
                        v-model="videoQuality"
                    ></v-select>

                    <!--
                    <v-select
                        messages="Video device"
                        :items="['Frontal camera', 'Back camera']"
                        label="Video device"
                        dense
                        solo
                    ></v-select>-->
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
        getStoredAudioPresents,
        setStoredAudioPresents,
        getStoredVideoPresents,
        setStoredVideoPresents
    } from "./utils";
    import {videochat_name} from "./routes";

    export default {
        data () {
            return {
                changing: false,
                show: false,

                audioPresents: null,
                videoPresents: null,
                videoQuality: null,
            }
        },
        methods: {
            showModal() {
                this.audioPresents = getStoredAudioPresents();
                this.videoPresents = getStoredVideoPresents();
                this.videoQuality = getVideoResolution();
                this.show = true;
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
                // https://github.com/pion/ion-sdk-js/blob/master/src/stream.ts#L10
                return ['qvga', 'vga', 'shd', 'hd', 'fhd', 'qhd']
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