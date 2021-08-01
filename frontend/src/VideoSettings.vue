<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="440" persistent>
            <v-card :disabled="changing" :loading="changing">
                <v-card-title>Video settings</v-card-title>

                <v-card-text class="px-4 py-0">
                    <!--<v-row no-gutters>
                        <v-col
                        >
                            <v-checkbox
                                dense
                                :model="audioPresents"
                                :label="`I have a microphone`"
                            ></v-checkbox>
                        </v-col>

                        <v-col
                        >
                            <v-checkbox
                                dense
                                :model="videoPresents"
                                :label="`I have a videocamera`"
                            ></v-checkbox>
                        </v-col>
                    </v-row>-->

                    <v-select
                        messages="Quality"
                        :items="qualityItems"
                        label="Quality"
                        dense
                        solo
                        @change="changeVideoResolution"
                        :value="videoQuality"
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
                    <v-btn color="error" class="mr-4" @click="closeModal()">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>

            </v-card>

        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {
        OPEN_VIDEO_SETTINGS,
        REQUEST_CHANGE_VIDEO_RESOLUTION,
        VIDEO_RESOLUTION_CHANGED,
    } from "./bus";
    import {KEY_RESOLUTION} from "./utils";

    const defaultResolution = 'hd';

    export default {
        data () {
            return {
                changing: false,
                show: false,
                audioPresents: true,
                videoPresents: true,
            }
        },
        methods: {
            showModal() {
                this.$data.show = true;
            },
            closeModal() {
                this.show = false;
            },

            onVideoResolutionChanged(res) {
                console.log("onVideoResolutionChanged", res);
                this.videoQuality = res;
                this.changing = false;
            },

            changeVideoResolution(newVideoResolution) {
                console.log("Setting new video resolution", newVideoResolution);
                this.changing = true;
                localStorage.setItem(KEY_RESOLUTION, newVideoResolution);
                bus.$emit(REQUEST_CHANGE_VIDEO_RESOLUTION, newVideoResolution);
            }
        },
        computed: {
            qualityItems() {
                // https://github.com/pion/ion-sdk-js/blob/master/src/stream.ts#L10
                return ['qvga', 'vga', 'shd', 'hd', 'fhd', 'qhd']
            },
            videoQuality: {
                get() {
                    return localStorage.getItem(KEY_RESOLUTION);

                    let got = localStorage.getItem(KEY_RESOLUTION);
                    if (!got) {
                        localStorage.setItem(KEY_RESOLUTION, defaultResolution);
                        got = localStorage.getItem(KEY_RESOLUTION);
                    }
                    return got;

                },
                set(newVideoResolution) {
                    localStorage.setItem(KEY_RESOLUTION, newVideoResolution);
                }
            }
        },
        created() {
            bus.$on(OPEN_VIDEO_SETTINGS, this.showModal);
            bus.$on(VIDEO_RESOLUTION_CHANGED, this.onVideoResolutionChanged)
        },
        destroyed() {
            bus.$off(OPEN_VIDEO_SETTINGS, this.showModal);
            bus.$off(VIDEO_RESOLUTION_CHANGED, this.onVideoResolutionChanged)
        },
    }
</script>