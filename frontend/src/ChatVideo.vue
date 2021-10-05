<template>
    <v-col cols="12" class="ma-0 pa-0" id="video-container">
        <v-snackbar v-model="showPermissionAsk" color="warning" timeout="-1" :multi-line="true" top>
            Please allow audio autoplay. If not, it will be enabled after unmute.
            <template v-slot:action="{ attrs }">
                <v-btn
                    light
                    v-bind="attrs"
                    @click="onClickPermitted()"
                >
                    Allow
                </v-btn>
                <v-btn icon v-bind="attrs" @click="showPermissionAsk = false"><v-icon color="white">mdi-close-circle</v-icon></v-btn>
            </template>
        </v-snackbar>

        <p v-if="errorDescription" class="error">{{ errorDescription }}</p>

        <UserVideo ref="localVideoComponent" :key="localPublisherKey" :initial-muted="initialMuted"/>
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
      AUDIO_START_MUTING, FORCE_MUTE, REQUEST_CHANGE_VIDEO_PARAMETERS,
      SHARE_SCREEN_START,
      SHARE_SCREEN_STOP, VIDEO_CALL_CHANGED, VIDEO_PARAMETERS_CHANGED,
      VIDEO_START_MUTING
    } from "./bus";
    import axios from "axios";
    import { Client, LocalStream } from 'ion-sdk-js';
    import { IonSFUJSONRPCSignal } from 'ion-sdk-js/lib/signal/json-rpc-impl';
    import UserVideo from "./UserVideo";
    import {
        audioMuteDefault,
        getWebsocketUrlPrefix,
        getVideoResolution,
        getStoredAudioPresents,
        getStoredVideoPresents,
    } from "./utils";
    import { v4 as uuidv4 } from 'uuid';
    import vuetify from './plugins/vuetify'

    const UserVideoClass = Vue.extend(UserVideo);

    let pingTimerId;
    const PUT_USER_DATA_METHOD = "putUserData";
    const USER_BY_STREAM_ID_METHOD = "userByStreamId";
    const pingInterval = 5000;
    const videoProcessRestartInterval = 1000;
    const askUserNameInterval = 1000;
    const MAX_MISSED_FAILURES = 5;

    export default {
        data() {
            return {
                clientLocal: null,
                streams: {},
                remotesDiv: null,
                signalLocal: null,
                localMediaStream: null,
                localPublisherKey: 1,
                chatId: null,
                remoteVideoIsMuted: true,
                peerId: null,

                // this one is about skipping ping checks during changing media stream
                insideSwitchingCameraScreen: false,

                // this two are about restart process
                restartingStarted: false,
                closingStarted: false,

                showPermissionAsk: true,

                errorDescription: null
            }
        },
        props: ['chatDto'],
        computed: {
            ...mapGetters({
                currentUser: GET_USER,
                videoMuted: GET_MUTE_VIDEO,
                audioMuted: GET_MUTE_AUDIO,
            }),
            myUserName() {
                return this.currentUser.login
            },
            initialMuted() {
                return audioMuteDefault;
            },
        },
        methods: {
            onClickPermitted() {
                this.ensureAudioIsEnabledAccordingBrowserPolicies();
                this.showPermissionAsk = false;
            },
            joinSession(configObj) {
                console.info("Used webrtc config", JSON.stringify(configObj));

                this.signalLocal = new IonSFUJSONRPCSignal(
                    getWebsocketUrlPrefix()+`/api/video/${this.chatId}/ws`
                );
                this.remotesDiv = document.getElementById("video-container");

                this.clientLocal = new Client(this.signalLocal, configObj);

                this.clientLocal.onspeaker = (messageEvent) => {
                    console.debug("Speaking event", messageEvent);
                    this.enumerateAllStreams((component, streamId) => {
                        console.debug("Resetting speaking", streamId);
                        component.setSpeaking(false);
                    })
                    for (const speakingStreamId of messageEvent) {
                        console.debug("Setting speaking", speakingStreamId);
                        this.applyCallbackToStreamId(speakingStreamId, (component) => {
                            component.setSpeaking(true);
                        });
                    }
                }

                this.signalLocal.onerror = (e) => { console.error("Error in signal", e); }
                this.signalLocal.onclose = () => {
                  console.info("Signal is closed, something gonna happen");
                  this.tryRestartVideoProcess();
                }

                this.peerId = uuidv4();
                this.signalLocal.onopen = () => {
                    console.info("Signal opened, joining to session...")
                    this.clientLocal.join(`chat${this.chatId}`, this.peerId).then(()=>{
                        console.info("Joined to session, gathering media devices")
                        this.getAndPublishLocalMediaStream({})
                            .then(()=>{
                              this.notifyAboutJoining();
                            }).then(value => {
                                this.startHealthCheckPing();
                            })
                            .catch(reason => {
                              console.error("Error during publishing camera stream, won't restart...", reason);
                              this.errorDescription = reason;
                            });
                    })
                }

                // adding remote tracks
                this.clientLocal.ontrack = (track, stream) => {
                    console.debug("got track", track.id, "for stream", stream.id);
                    track.onunmute = () => {
                        if (!this.streams[stream.id]) {
                            console.log("set track", track.id, "for stream", stream.id);

                            const component = new UserVideoClass({vuetify: vuetify, propsData: { initialMuted: this.remoteVideoIsMuted }});
                            component.$mount();
                            this.remotesDiv.appendChild(component.$el);
                            component.setSource(stream);
                            this.streams[stream.id] = {stream, component, failureCount: 0};

                            stream.onremovetrack = (e) => {
                                console.log("onremovetrack", e);
                                if (e.track) {
                                    this.removeStream(stream.id, component)
                                }
                            };

                            this.askUserNameWithRetries(stream.id);
                        }
                    }
                };
            },
            removeStream(streamId, component) {
              console.log("Removing stream streamId=", streamId);
              try {
                this.remotesDiv.removeChild(component.$el);
                component.$destroy();
              } catch (e) {
                console.debug("Something wrong on removing child", e, component.$el, this.remotesDiv);
              }
              delete this.streams[streamId];
            },
            tryRestartWithResetOncloseHandler() {
                if (this.signalLocal) {
                    this.signalLocal.onclose = null; // remove onclose handler with restart in order to prevent cyclic restarts
                }
                this.tryRestartVideoProcess();
            },
            leaveSession() {
                if (pingTimerId) {
                    clearInterval(pingTimerId);
                }
                for (const streamId in this.streams) {
                    console.log("Cleaning stream " + streamId);
                    const component = this.streams[streamId].component;
                    this.removeStream(streamId, component);
                }
                if (this.localMediaStream) {
                    this.localMediaStream.getTracks().forEach(t => t.stop());
                    this.localMediaStream.unpublish();
                }
                if (this.clientLocal) {
                    this.clientLocal.close();
                }
                this.clientLocal = null;
                this.signalLocal = null;
                this.streams = {};
                this.remotesDiv = null;
                this.localMediaStream = null;
                this.insideSwitchingCameraScreen = false;
                this.peerId = null;

                this.$store.commit(SET_MUTE_VIDEO, false);
                this.$store.commit(SET_MUTE_AUDIO, audioMuteDefault);
            },
            startHealthCheckPing() {
                console.log("Setting up ping every", pingInterval, "ms");
                pingTimerId = setInterval(()=>{
                    if (!this.insideSwitchingCameraScreen) {
                        const localStreamId = this.$refs.localVideoComponent.getStreamId();
                        console.debug("Checking self user", "streamId", localStreamId);
                        this.signalLocal.call(USER_BY_STREAM_ID_METHOD, {streamId: localStreamId, includeOtherStreamIds: true}).then(value => {
                            if (!value.found) {
                                console.warn("Detected absence of self user on server, restarting...", "streamId", localStreamId);
                                this.tryRestartWithResetOncloseHandler();
                            } else {
                                console.debug("Successfully checked self user", "streamId", localStreamId, value);

                                // check other
                                for (const streamId in this.streams) {
                                    console.debug("Checking other streamId", streamId);
                                    const streamHolder = this.streams[streamId];
                                    const component = streamHolder.component;
                                    if (value.otherStreamIds.filter(v => v == streamId).length == 0) {
                                        streamHolder.failureCount++;
                                        console.debug("Other streamId", streamId, "is not present, failureCount icreased to", streamHolder.failureCount);
                                        if (streamHolder.failureCount > MAX_MISSED_FAILURES) {
                                            console.debug("Other streamId", streamId, "subsequently is not present, removing...");
                                            this.removeStream(streamId, component);
                                        }
                                    } else {
                                        console.debug("Other streamId", streamId, "is present, resetting failureCount");
                                        streamHolder.failureCount = 0;
                                    }
                                }
                            }
                        })
                    } else {
                        console.debug("Skipping checking self user because we switch camera to screen sharing or vice versa");
                    }
                }, pingInterval)
            },
            askUserNameWithRetries(streamId) {
                // request-response with signalLocal and error handling
                this.signalLocal.call(USER_BY_STREAM_ID_METHOD, {streamId: streamId}).then(value => {
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
                            const streamHolder = this.streams[data.streamId];
                            if (streamHolder) {
                                streamHolder.component.setUserName(data.login);
                                streamHolder.component.setDisplayAudioMute(data.audioMute);
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
                    peerId: this.peerId,
                    streamId: this.$refs.localVideoComponent.getStreamId(),
                    login: this.myUserName,
                    videoMute: this.videoMuted, // from store
                    audioMute: this.audioMuted
                };
                this.signalLocal.notify(PUT_USER_DATA_METHOD, toSend)
            },
            notifyAboutJoining() {
                if (this.chatId) {
                    this.notifyWithData();
                } else {
                    console.warn("Unable to notify about joining")
                }
            },
            ensureAudioIsEnabledAccordingBrowserPolicies() {
                if (this.remoteVideoIsMuted) {
                    // Unmute all the current videoElements.
                    for (const streamHolder of Object.values(this.streams)) {
                        let { component } = streamHolder;
                        const videoElement = component.getVideoElement();
                        videoElement.pause();
                        videoElement.muted = false;
                        videoElement.play();
                    }
                    // Set remoteVideoIsMuted to false so that all future autoplays work.
                    this.remoteVideoIsMuted = false;
                }
            },
            onStartScreenSharing() {
                return this.onSwitchMediaStream({screen: true});
            },
            onStopScreenSharing() {
                return this.onSwitchMediaStream({screen: false});
            },
            onSwitchMediaStream({screen = false}) {
                if (this.localMediaStream) {
                    this.localMediaStream.unpublish();
                    this.localMediaStream.getTracks().forEach(t => t.stop());
                }
                this.$refs.localVideoComponent.setSource(null);
                this.localPublisherKey++;
                this.getAndPublishLocalMediaStream({screen})
                    .catch(reason => {
                      console.error("Error during publishing screen stream, won't restart...", reason);
                      this.$refs.localVideoComponent.setUserName('Error get getDisplayMedia');
                    })
                    .then(value => {
                        this.notifyWithData();
                    });
            },
            getAndPublishLocalMediaStream({screen = false}) {
                this.insideSwitchingCameraScreen = true;
                const resolution = getVideoResolution();
                bus.$emit(VIDEO_PARAMETERS_CHANGED);

                const audio = getStoredAudioPresents();
                const video = getStoredVideoPresents();

                if (!audio && !video && !screen) {
                    console.info("Not able to build local media stream, returning a successful promise");
                    Vue.nextTick(() => {
                        this.$refs.localVideoComponent.setUserName('No media configured');
                    });

                    if (screen) {
                        this.$store.commit(SET_SHARE_SCREEN, true);
                    } else {
                        this.$store.commit(SET_SHARE_SCREEN, false);
                    }
                    // this.insideSwitchingCameraScreen = false;

                    return Promise.resolve(true);
                }

                const localStream = screen ?
                    LocalStream.getDisplayMedia({
                        audio: audio,
                        video: true
                    }) :
                    LocalStream.getUserMedia({
                        resolution: resolution,
                        audio: audio,
                        video: video
                    });

                return localStream.then((media) => {
                  this.localMediaStream = media;
                  this.$refs.localVideoComponent.setSource(media);
                  this.$refs.localVideoComponent.setStreamMuted(true);
                  this.$refs.localVideoComponent.setUserName(this.myUserName);
                  this.$refs.localVideoComponent.setDisplayAudioMute(this.audioMuted);
                  console.log("Publishing " + (screen ? "screen" : "camera"));
                  this.clientLocal.publish(media);
                  console.log("Successfully published " + (screen ? "screen" : "camera"));
                  if (screen) {
                      this.$store.commit(SET_SHARE_SCREEN, true);
                  } else {
                      this.$store.commit(SET_SHARE_SCREEN, false);
                  }
                  this.setLocalMuteDefaults();
                  this.insideSwitchingCameraScreen = false;
                });
            },
            setLocalMuteDefaults() {
                if (this.audioMuted) {
                    this.localMediaStream.mute("audio");
                } else {
                    this.localMediaStream.unmute("audio");
                }
            },
            startVideoProcess() {
                this.getConfig()
                    .then(config => {
                        console.info("Config fetched, initializing to session...")
                        return this.joinSession(config);
                    })
                    .catch(reason => {
                        console.error("Error during get config, restarting...")
                        this.tryRestartVideoProcess();
                    })
            },
            tryRestartVideoProcess() {
                if (!this.closingStarted && !this.restartingStarted) {
                    this.restartingStarted = true;
                    setTimeout(() => {
                        console.info("Will restart video process after", videoProcessRestartInterval, "ms");
                        try {
                            this.leaveSession();
                        } catch (e) {
                            console.warn("Some problems during leaving session, ignoring them...")
                        }
                        try {
                            this.startVideoProcess();
                            this.restartingStarted = false;
                        } catch (e) {
                            console.error("Error during starting video process, restarting...");
                            this.restartingStarted = false;
                            this.tryRestartVideoProcess();
                        }
                    }, videoProcessRestartInterval);
                } else {
                    console.info("Will not restart video process because", "closingStarted", this.closingStarted, "restartingStarted", this.restartingStarted);
                }
            },
            onStartVideoMuting(requestedState) {
                if (requestedState) {
                    this.localMediaStream.mute("video");
                    this.$store.commit(SET_MUTE_VIDEO, requestedState);
                    this.notifyWithData();
                } else {
                    this.localMediaStream.unmute("video").then(value => {
                        this.$store.commit(SET_MUTE_VIDEO, requestedState);
                        this.notifyWithData();
                    })
                }
            },
            onStartAudioMuting(requestedState) {
                this.ensureAudioIsEnabledAccordingBrowserPolicies();
                if (requestedState) {
                    this.localMediaStream.mute("audio");
                    this.$store.commit(SET_MUTE_AUDIO, requestedState);
                    this.$refs.localVideoComponent.setDisplayAudioMute(requestedState);
                    this.notifyWithData();
                } else {
                    this.localMediaStream.unmute("audio").then(value => {
                        this.$store.commit(SET_MUTE_AUDIO, requestedState);
                        this.$refs.localVideoComponent.setDisplayAudioMute(requestedState);
                        this.notifyWithData();
                    })
                }
            },
            onForceMuteByAdmin(dto) {
                if(dto.chatId == this.chatId) {
                    this.onStartAudioMuting(true);
                }
            },
            onVideoCallChanged(dto) {
                if (dto) {
                    const data = dto.data;
                    if (data) {
                        const streamInfo = this.streams[data.streamId];
                        if (streamInfo) {
                            streamInfo.component.setDisplayAudioMute(data.audioMute);
                        }
                    }
                }
            },
            onVideoParametersChanged() {
                this.tryRestartWithResetOncloseHandler();
            },
            enumerateAllStreams(callback) {
                if (this.localMediaStream && this.$refs.localVideoComponent) {
                    callback(this.$refs.localVideoComponent, this.$refs.localVideoComponent.getStreamId());
                }
                for (const streamId in this.streams) {
                    const streamHolder = this.streams[streamId];
                    if (streamHolder) {
                        callback(streamHolder.component, streamId);
                    }
                }
            },
            applyCallbackToStreamId(streamId, callback) {
                if (this.localMediaStream && this.$refs.localVideoComponent && this.$refs.localVideoComponent.getStreamId() == streamId) {
                    callback(this.$refs.localVideoComponent, this.$refs.localVideoComponent.getStreamId());
                    return;
                }
                const streamHolder = this.streams[streamId];
                if (streamHolder) {
                    callback(streamHolder.component);
                }
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
            bus.$on(REQUEST_CHANGE_VIDEO_PARAMETERS, this.onVideoParametersChanged);
            bus.$on(FORCE_MUTE, this.onForceMuteByAdmin);
        },
        destroyed() {
            bus.$off(SHARE_SCREEN_START, this.onStartScreenSharing);
            bus.$off(SHARE_SCREEN_STOP, this.onStopScreenSharing);
            bus.$off(VIDEO_START_MUTING, this.onStartVideoMuting);
            bus.$off(AUDIO_START_MUTING, this.onStartAudioMuting);
            bus.$off(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
            bus.$off(REQUEST_CHANGE_VIDEO_PARAMETERS, this.onVideoParametersChanged);
            bus.$off(FORCE_MUTE, this.onForceMuteByAdmin);
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