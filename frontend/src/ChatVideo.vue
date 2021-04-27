<template>
    <v-col cols="12" class="ma-0 pa-0" id="video-container">
        <UserVideo ref="localVideoComponent" :key="localPublisherKey"/>
    </v-col>
</template>

<script>
    import Vue from 'vue';
    import {mapGetters} from "vuex";
    import {
        GET_MUTE_AUDIO,
        GET_MUTE_VIDEO,
        GET_USER,
        SET_MUTE_AUDIO,
        SET_MUTE_VIDEO, SET_SHARE_SCREEN,
        SET_SHOW_CALL_BUTTON, SET_SHOW_HANG_BUTTON,
        SET_VIDEO_CHAT_USERS_COUNT
    } from "./store";
    import bus, {
        AUDIO_START_MUTING, REQUEST_CHANGE_VIDEO_RESOLUTION,
        SHARE_SCREEN_START,
        SHARE_SCREEN_STOP, VIDEO_CALL_CHANGED, VIDEO_RESOLUTION_CHANGED,
        VIDEO_START_MUTING
    } from "./bus";
    import axios from "axios";
    import { Client, LocalStream } from 'ion-sdk-js';
    import { IonSFUJSONRPCSignal } from 'ion-sdk-js/lib/signal/json-rpc-impl';
    import UserVideo from "./UserVideo";
    import {audioMuteDefault, getWebsocketUrlPrefix} from "./utils";
    import { v4 as uuidv4 } from 'uuid';
    import Vuetify from 'vuetify/lib/framework'

    const ComponentClass = Vue.extend(UserVideo);
    const vuetify = new Vuetify();

    const peerId = uuidv4();
    let pingTimerId;
    const pingInterval = 5000;
    const videoProcessRestartInterval = 1000;
    const askUserNameInterval = 1000;

    export default {
        data() {
            return {
                clientLocal: null,
                streams: {},
                remotesDiv: null,
                signalLocal: null,
                dataChannel: null,
                localMedia: null,
                localPublisherKey: 1,
                closingStarted: false,
                chatId: null
            }
        },
        props: ['chatDto'],
        computed: {
            ...mapGetters({currentUser: GET_USER, videoMuted: GET_MUTE_VIDEO, audioMuted: GET_MUTE_AUDIO}),
            myUserName() {
                return this.currentUser.login
            }
        },
        methods: {
            joinSession(configObj) {
                const config = {
                    iceServers: configObj.ICEServers.map((iceServConf)=>{
                        const result = {
                            urls: iceServConf.URLs
                        }
                        if (iceServConf.Username) {
                            result.username = iceServConf.Username;
                        }
                        if (iceServConf.Credential) {
                            result.credential = iceServConf.Credential;
                        }
                        return result;
                    })
                };
                console.info("Created webrtc config", JSON.stringify(config));

                this.signalLocal = new IonSFUJSONRPCSignal(
                    getWebsocketUrlPrefix()+`/api/video/${this.chatId}/ws`
                );
                this.remotesDiv = document.getElementById("video-container");

                this.clientLocal = new Client(this.signalLocal, config);

                this.signalLocal.onerror = (e) => { console.error("Error in signal", e); }
                this.signalLocal.onclose = () => {
                  console.info("Signal is closed, something gonna happen");
                  this.tryRestartVideoProcess();
                }

                this.signalLocal.onopen = () => {
                    this.clientLocal.join(`chat${this.chatId}`, peerId).then(()=>{
                        this.getAndPublishCamera()
                            .then(()=>{
                              this.notifyAboutJoining();
                            }).then(value => {
                                this.startHealthCheckPing();
                            })
                            .catch(reason => {
                              console.error("Error during publishing camera stream, won't restart...", reason);
                              this.$refs.localVideoComponent.setUserName('Error get getUserMedia');
                            });
                    })
                }

                // adding remote tracks
                this.clientLocal.ontrack = (track, stream) => {
                    console.debug("got track", track.id, "for stream", stream.id);
                    track.onunmute = () => {
                        if (!this.streams[stream.id]) {
                            console.log("set track", track.id, "for stream", stream.id, "vuetify", this.$vuetify);

                            const component = new ComponentClass({vuetify: vuetify});
                            component.$mount();
                            this.remotesDiv.appendChild(component.$el);
                            component.setSource(stream);
                            this.streams[stream.id] = {stream, component};

                            stream.onremovetrack = () => {
                                this.removeTrack(stream.id, track.id, component)
                            };

                            this.askUserNameWithRetries(stream.id);
                        }
                    }
                };
            },
            removeTrack(streamId, trackId, component) {
              console.log("removed track", trackId, "for stream", streamId);
              try {
                this.remotesDiv.removeChild(component.$el);
                component.$destroy();
              } catch (e) {
                console.debug("Something wrong on removing child", e, component.$el, this.remotesDiv);
              }
              delete this.streams[streamId];
            },
            tryRestartWithResetOncloseHandler() {
                this.signalLocal.onclose = null; // remove onclose handler with restart in order to prevent cyclic restarts
                this.tryRestartVideoProcess();
            },
            leaveSession() {
                if (pingTimerId) {
                    clearInterval(pingTimerId);
                }
                for (const prop in this.streams) {
                    console.log("Cleaning stream " + prop);
                    const component = this.streams[prop].component;
                    this.removeTrack(prop, '_not_set', component);
                }
                if (this.localMedia) {
                    this.localMedia.getTracks().forEach(t => t.stop());
                }
                if (this.clientLocal) {
                    this.clientLocal.close();
                }
                this.clientLocal = null;
                this.signalLocal = null;
                this.streams = {};
                this.remotesDiv = null;
                this.localMedia = null;

                this.$store.commit(SET_MUTE_VIDEO, false);
                this.$store.commit(SET_MUTE_AUDIO, audioMuteDefault);
            },
            startHealthCheckPing() {
                let localStreamId = this.$refs.localVideoComponent.getStreamId();
                console.log("Setting up ping every", pingInterval, "ms");
                pingTimerId = setInterval(()=>{
                    this.signalLocal.call("userByStreamId", {streamId: localStreamId}).then(value => {
                        if (!value.found) {
                            console.warn("Detected absence of self user on server, restarting...");
                            this.tryRestartWithResetOncloseHandler();
                        } else {
                            console.debug("Successfully checked self user", value);
                        }
                    })
                }, pingInterval)
            },
            askUserNameWithRetries(streamId) {
                // request-response with signalLocal and error handling
                this.signalLocal.call("userByStreamId", {streamId: streamId}).then(value => {
                    if (!value.found) {
                        if (!this.closingStarted) {
                            console.log("Rescheduling asking for userName");
                            setTimeout(() => {
                                this.askUserNameWithRetries(streamId);
                            }, askUserNameInterval);
                        }
                    } else {
                        console.debug("Successfully got data by streamId", streamId);
                        const data = value.userDto;
                        if (data) {
                            const component = this.streams[data.streamId];
                            if (component) {
                                component.component.setUserName(data.login);
                                component.component.setAudioMute(data.audioMute);
                            }
                        }
                    }
                })
            },
            getConfig() {
                return axios
                    .get(`/api/video/${this.chatId}/config`)
                    .then(response => response.data)
            },
            notifyWithData() {
                const toSend = {
                    peerId: peerId,
                    streamId: this.$refs.localVideoComponent.getStreamId(),
                    login: this.myUserName,
                    videoMute: this.videoMuted, // from store
                    audioMute: this.audioMuted
                };
                axios.put(`/api/video/${this.chatId}/notify`, toSend).catch(error => {
                    console.log(error.response)
                })
            },
            notifyAboutJoining() {
                if (this.chatId) {
                    this.notifyWithData();
                } else {
                    console.warn("Unable to notify about joining")
                }
            },
            onStartScreenSharing() {
                this.localMedia.unpublish();
                if (this.localMedia) {
                  this.localMedia.getTracks().forEach(t => t.stop());
                }
                this.$refs.localVideoComponent.setSource(null);
                this.localPublisherKey++;
                this.getAndPublishScreen()
                    .catch(reason => {
                      console.error("Error during publishing screen stream, won't restart...", reason);
                      this.$refs.localVideoComponent.setUserName('Error get getDisplayMedia');
                    })
                    .then(value => {
                        this.notifyWithData();
                    });
            },
            onStopScreenSharing() {
                this.localMedia.unpublish();
                if (this.localMedia) {
                  this.localMedia.getTracks().forEach(t => t.stop());
                }
                this.$refs.localVideoComponent.setSource(null);
                this.localPublisherKey++;
                this.getAndPublishCamera()
                    .then(value => {
                        this.notifyWithData();
                    });
            },
            getAndPublishCamera() {
                const resolution = this.getVideoResolution();
                bus.$emit(VIDEO_RESOLUTION_CHANGED, resolution);
                return LocalStream.getUserMedia({
                  resolution: resolution,
                  audio: true,
                }).then((media) => {
                  this.localMedia = media;
                  this.$refs.localVideoComponent.setSource(media);
                  this.$refs.localVideoComponent.setStreamMuted(true);
                  this.$refs.localVideoComponent.setUserName(this.myUserName);
                  this.$refs.localVideoComponent.setAudioMute(this.audioMuted);
                  this.clientLocal.publish(media);
                  this.$store.commit(SET_SHARE_SCREEN, false);
                  this.setMuteDefaults();
                });
            },
            getAndPublishScreen() {
                return LocalStream.getDisplayMedia({
                  audio: true,
                }).then((media) => {
                    this.localMedia = media;
                    //this.localMedia.unmute("audio");
                    this.$refs.localVideoComponent.setSource(media);
                    this.$refs.localVideoComponent.setStreamMuted(true);
                    this.$refs.localVideoComponent.setAudioMute(this.audioMuted);
                    this.$refs.localVideoComponent.setUserName(this.myUserName);
                    this.clientLocal.publish(media);
                    this.$store.commit(SET_SHARE_SCREEN, true);
                    this.setMuteDefaults();
                });
            },
            setMuteDefaults() {
                if (this.audioMuted) {
                    this.localMedia.mute("audio");
                } else {
                    this.localMedia.unmute("audio");
                }
            },
            startVideoProcess() {
                this.getConfig()
                    .catch(reason => {
                      console.error("Error during get config, restarting...")
                      this.tryRestartVideoProcess();
                    })
                    .then(config => {
                    this.joinSession(config);
                })
            },
            tryRestartVideoProcess() {
              setTimeout(() => {
                if (!this.closingStarted) {
                  console.info("Will restart video process after 1 sec");
                  try {
                    this.leaveSession();
                  } catch (e) {
                    console.warn("Some problems during leaving session, ignoring them...")
                  }
                  try {
                    this.startVideoProcess();
                  } catch (e) {
                    console.error("Error during starting video process, restarting...")
                    this.tryRestartVideoProcess();
                  }
                } else {
                  console.info("Will not restart video process because closingStarted");
                }
              }, videoProcessRestartInterval);
            },
            onStartVideoMuting(requestedState) {
                if (requestedState) {
                    this.localMedia.mute("video");
                    this.$store.commit(SET_MUTE_VIDEO, requestedState);
                    this.notifyWithData();
                } else {
                    this.localMedia.unmute("video").then(value => {
                        this.$store.commit(SET_MUTE_VIDEO, requestedState);
                        this.notifyWithData();
                    })
                }
            },
            onStartAudioMuting(requestedState) {
                if (requestedState) {
                    this.localMedia.mute("audio");
                    this.$store.commit(SET_MUTE_AUDIO, requestedState);
                    this.$refs.localVideoComponent.setAudioMute(requestedState);
                    this.notifyWithData();
                } else {
                    this.localMedia.unmute("audio").then(value => {
                        this.$store.commit(SET_MUTE_AUDIO, requestedState);
                        this.$refs.localVideoComponent.setAudioMute(requestedState);
                        this.notifyWithData();
                    })
                }
            },
            onVideoCallChanged(dto) {
                if (dto) {
                    const data = dto.data;
                    if (data) {
                        const component = this.streams[data.streamId];
                        if (component) {
                            component.component.setAudioMute(data.audioMute);
                        }
                    }
                }
            },
            onVideoResolutionChanged(newResolution) {
                this.storeVideoResolution(newResolution);
                this.tryRestartWithResetOncloseHandler();
            },
            getVideoResolution() {
                let got = this.getStoredVideoResolution();
                if (!got) {
                    this.storeVideoResolution('shd');
                    got = this.getStoredVideoResolution();
                }
                return got;
            },
            getStoredVideoResolution() {
                return localStorage.getItem('videoResolution');
            },
            storeVideoResolution(newVideoResolution) {
                localStorage.setItem('videoResolution', newVideoResolution);
            },
        },
        mounted() {
            this.chatId = this.$route.params.id;

            this.closingStarted = false;
            this.$store.commit(SET_SHOW_CALL_BUTTON, false);
            this.$store.commit(SET_SHOW_HANG_BUTTON, true);
            window.addEventListener('beforeunload', this.leaveSession)
            this.startVideoProcess();
        },
        beforeDestroy() {
            this.$store.commit(SET_SHOW_CALL_BUTTON, true);
            this.$store.commit(SET_SHOW_HANG_BUTTON, false);
            this.$store.commit(SET_SHARE_SCREEN, false);
            this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, 0);

            this.closingStarted = true;
            window.removeEventListener('beforeunload', this.leaveSession);
            this.leaveSession();
        },
        created() {
            bus.$on(SHARE_SCREEN_START, this.onStartScreenSharing);
            bus.$on(SHARE_SCREEN_STOP, this.onStopScreenSharing);
            bus.$on(VIDEO_START_MUTING, this.onStartVideoMuting);
            bus.$on(AUDIO_START_MUTING, this.onStartAudioMuting);
            bus.$on(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
            bus.$on(REQUEST_CHANGE_VIDEO_RESOLUTION, this.onVideoResolutionChanged);
        },
        destroyed() {
            bus.$off(SHARE_SCREEN_START, this.onStartScreenSharing);
            bus.$off(SHARE_SCREEN_STOP, this.onStopScreenSharing);
            bus.$off(VIDEO_START_MUTING, this.onStartVideoMuting);
            bus.$off(AUDIO_START_MUTING, this.onStartAudioMuting);
            bus.$off(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
            bus.$off(REQUEST_CHANGE_VIDEO_RESOLUTION, this.onVideoResolutionChanged);
        },
        components: {
            UserVideo
        }
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