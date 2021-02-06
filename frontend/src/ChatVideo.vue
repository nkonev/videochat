<template>
    <v-col cols="12" class="ma-0 pa-0" id="video-container">
        <div>
            <video
                id="local-video"
                style="background-color: black"
                width="320"
                height="240"
            ></video>
        </div>
        <div id="remotes" class="col-6 pt-2"></div>
    </v-col>
</template>

<script>
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import bus, {
        CHANGE_PHONE_BUTTON,
        SHARE_SCREEN_START, SHARE_SCREEN_STATE_CHANGED,
        SHARE_SCREEN_STOP,
        VIDEO_CALL_CHANGED,
        VIDEO_LOCAL_ESTABLISHED
    } from "./bus";
    import {phoneFactory} from "./changeTitle";
    import axios from "axios";
    import {getWebsocketUrlPrefix} from "./utils";
    import { Client, LocalStream, RemoteStream } from 'ion-sdk-js';
    import { IonSFUJSONRPCSignal } from 'ion-sdk-js/lib/signal/json-rpc-impl';

    export default {
        data() {
            return {
                clientLocal: null,
                streams: {},
                localVideo: null,
                remotesDiv: null,
                signalLocal: null,
            }
        },
        props: ['chatDto'],
        computed: {
            chatId() {
                return this.$route.params.id
            },
            ...mapGetters({currentUser: GET_USER}),
            myUserName() {
                return this.currentUser.login
                //return 'user' + Math.floor(Math.random() * 100)
            }
        },
        methods: {
            joinSession(configObj) {
                const config = {
                    iceServers: [
                        {
                            urls: "stun:stun.l.google.com:19302",
                        },
                    ],
                };
                this.signalLocal = new IonSFUJSONRPCSignal(
                    "ws://localhost:7000/ws"
                );
                this.localVideo = document.getElementById("local-video");
                this.remotesDiv = document.getElementById("remotes");

                this.clientLocal = new Client(this.signalLocal, config);

                this.signalLocal.onopen = () => {
                    this.clientLocal.join(`chat${this.chatId}`);
                    this.startPublishing();
                }
                this.signalLocal.onerror = () => { console.error("Error in signal"); }
                this.signalLocal.onclose = () => { console.info("Signal is closed"); }

                this.clientLocal.ontrack = (track, stream) => {
                    console.log("got track", track.id, "for stream", stream.id);
                    if (track.kind === "video") {
                        track.onunmute = () => {
                            console.log("Stream id", stream.id);
                            if (!this.streams[stream.id]) {
                                this.streams[stream.id] = stream;
                                let remoteVideo = document.createElement("video");
                                remoteVideo.srcObject = stream;
                                remoteVideo.autoplay = true;
                                remoteVideo.muted = true;

                                this.remotesDiv.appendChild(remoteVideo);
                                stream.onremovetrack = () => {
                                    this.streams[stream.id] = null;
                                    try {
                                        this.remotesDiv.removeChild(remoteVideo);
                                    } catch (e) {
                                        console.debug("Something wrong on removing child", e, remoteVideo, this.remotesDiv);
                                    }
                                };
                            }
                        };
                    }
                };

                window.addEventListener('beforeunload', this.leaveSession)
            },
            startPublishing() {
                LocalStream.getUserMedia({
                    resolution: "vga",
                    audio: true,
                })
                    .then((media) => {
                        this.localVideo.srcObject = media;
                        this.localVideo.autoplay = true;
                        this.localVideo.controls = true;
                        this.localVideo.muted = true;
                        this.clientLocal.publish(media);
                    })
                    .catch(console.error);
            },

            leaveSession() {
                this.clientLocal.close();

                this.clientLocal = null;
                this.signalLocal = null;
                this.localVideo = null;
                this.streams = {};
                this.remotesDiv = null;

                bus.$emit(VIDEO_CALL_CHANGED, {usersCount: 0}); // restore initial state
                this.notifyAboutLeaving();
                window.removeEventListener('beforeunload', this.leaveSession);
            },
            getConfig() {
                return axios
                    .get(`/api/chat/${this.chatId}/video/config`)
                    .then(response => response.data)
            },

            notifyAboutJoining() {
                if (this.chatId) {
                    //axios.put(`/api/chat/${this.chatId}/video/notify`);
                }
            },
            notifyAboutLeaving() {
                if (this.chatId) {
                    //axios.put(`/api/chat/${this.chatId}/video/notify`);
                }
            },
            onDoubleClick(e) {
              const elem = e.target;
              if (elem.requestFullscreen) {
                  elem.requestFullscreen();
              } else if (elem.webkitRequestFullscreen) { // Safari
                  elem.webkitRequestFullscreen();
              }
            },
            onStartScreenSharing() {

            },
            onStopScreenSharing() {

                bus.$emit(SHARE_SCREEN_STATE_CHANGED, false);
            },
        },
        mounted() {
            bus.$emit(VIDEO_LOCAL_ESTABLISHED);
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, false));

            this.getConfig().then(config => {
                this.joinSession(config);
            })

        },
        beforeDestroy() {
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, true));

            this.leaveSession();
        },
        created() {
            bus.$on(SHARE_SCREEN_START, this.onStartScreenSharing);
            bus.$on(SHARE_SCREEN_STOP, this.onStopScreenSharing);
        },
        destroyed() {
            bus.$off(SHARE_SCREEN_START, this.onStartScreenSharing);
            bus.$off(SHARE_SCREEN_STOP, this.onStopScreenSharing);
        },
    }
</script>

<style lang="stylus" scoped>
    #video-container {
        display: flex;
        flex-direction: row;
        overflow-x: auto;
        overflow-y: hidden;
        height 100%
    }
</style>