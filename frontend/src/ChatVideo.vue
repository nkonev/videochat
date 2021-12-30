<template>
    <v-col cols="12" class="ma-0 pa-0" id="video-container">
        <v-snackbar v-model="showPermissionAsk" color="warning" timeout="-1" :multi-line="true" top>
            {{ $vuetify.lang.t('$vuetify.please_allow_audio_policy_bypass') }}
            <template v-slot:action="{ attrs }">
                <v-btn
                    light
                    v-bind="attrs"
                    @click="onClickPermitted()"
                >
                    {{ $vuetify.lang.t('$vuetify.allow') }}
                </v-btn>
                <v-btn icon v-bind="attrs" @click="showPermissionAsk = false"><v-icon color="white">mdi-close-circle</v-icon></v-btn>
            </template>
        </v-snackbar>
        <p v-if="errorDescription" class="error">{{ errorDescription }}</p>

        <UserVideo ref="localVideoComponent" :key="localPublisherKey" :initial-muted="true" :id="getNewId()"/>
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
        AUDIO_START_MUTING, REQUEST_CHANGE_VIDEO_PARAMETERS,
        SHARE_SCREEN_START,
        SHARE_SCREEN_STOP, USER_PROFILE_CHANGED, VIDEO_CALL_CHANGED, VIDEO_PARAMETERS_CHANGED,
        VIDEO_START_MUTING
    } from "./bus";
    import axios from "axios";
    import { Client, LocalStream } from 'ion-sdk-js';
    import { IonSFUJSONRPCSignal } from 'ion-sdk-js/lib/signal/json-rpc-impl';
    import UserVideo from "./UserVideo";
    import {
        getWebsocketUrlPrefix,
        getVideoResolution,
        getStoredAudioDevicePresents,
        getStoredVideoDevicePresents, getCodec, hasLength,
    } from "./utils";
    import { v4 as uuidv4 } from 'uuid';
    import vuetify from './plugins/vuetify'
    import {chat_name, videochat_name} from "@/routes";

    const UserVideoClass = Vue.extend(UserVideo);

    let pingTimerId;
    const PUT_USER_DATA_METHOD = "putUserData";
    const USER_BY_STREAM_ID_METHOD = "userByStreamId";
    const KICK_NOTIFICATION = "kick";
    const FORCE_MUTE_NOTIFICATION = "force_mute";

    const pingInterval = 5000;
    const videoProcessRestartInterval = 1000;
    const MAX_MISSED_FAILURES = 5;
    const localAudioMutedInitial = false; // actually works only with camera, screen sharing always started without audio stream

    export default {
        data() {
            return {
                clientLocal: null,
                localStreams: {}, // user can have several cameras, or simultaneously translate camera and screen
                remoteStreams: {},
                remotesDiv: null, // todo rename to video container div (both local and remote)
                signalLocal: null,
                localMediaStream: null,//todo remove
                localPublisherKey: 1,
                chatId: null,
                remoteVideoIsMuted: true,
                peerId: null,

                // this one is about skipping ping checks during changing media stream
                isCnangingLocalStream: false,

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
        },
        methods: {
            onClickPermitted() {
                this.ensureAudioIsEnabledAccordingBrowserPolicies();
                this.showPermissionAsk = false;
            },
            appendUserVideo(stream, videoTagId, appendTo) {
                const component = new UserVideoClass({vuetify: vuetify, propsData: { initialMuted: this.remoteVideoIsMuted, id: videoTagId }});
                component.$mount();
                this.remotesDiv.appendChild(component.$el);
                component.setSource(stream);
                const streamHolder = {stream, component}
                appendTo[stream.id] = streamHolder;
                return component;
            },
            joinSession(configObj) {
                console.info("Used webrtc config", JSON.stringify(configObj));

                this.signalLocal = new IonSFUJSONRPCSignal(
                    getWebsocketUrlPrefix()+`/api/video/${this.chatId}/ws`
                );

                if (hasLength(configObj.codec)) {
                    console.log("Server overrided codec to", configObj.codec)
                    this.preferredCodec = configObj.codec;
                } else {
                    this.preferredCodec = getCodec();
                    console.log("Used codec from localstorage", this.preferredCodec)
                }
                this.clientLocal = new Client(this.signalLocal, {...configObj, codec: this.preferredCodec});

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
                this.signalLocal.on_notify(FORCE_MUTE_NOTIFICATION, dto => {
                    console.log("Got force mute", dto);
                    this.onForceMuteByAdmin(dto);
                });
                this.signalLocal.on_notify(KICK_NOTIFICATION, dto => {
                    console.log("Got kick", dto);
                    this.onVideoCallKicked(dto);
                });

                this.peerId = uuidv4();
                this.signalLocal.onopen = () => {
                    console.info("Signal opened, joining to session...")
                    this.clientLocal.join(`chat${this.chatId}`, this.peerId).then(()=>{
                        console.info("Joined to session, gathering media devices")
                        this.getAndPublishLocalMediaStream({})
                            .then(value => {

                            })
                            .catch(reason => {
                              console.error("Error during publishing camera stream, won't restart...", reason);
                              this.errorDescription = reason;
                            });
                    })
                }

                // adding remote tracks
                this.clientLocal.ontrack = (track, stream) => {
                    console.info("Got track", track, "kind=", track.kind, " for remote stream", stream);
                    track.onunmute = () => {
                        const streamId = stream.id;
                        if (!this.remoteStreams[streamId]) {
                            const videoTagId = 'remote-' + streamId + '-' + this.getNewId();
                            console.info("Setting track", track.id, "for remote stream", streamId, " into video tag id=", videoTagId);
                            const remoteComponent = this.appendUserVideo(stream, videoTagId, this.remoteStreams);
                            stream.onremovetrack = (e) => {
                                console.log("onremovetrack", e);
                                if (e.track) { // todo rewrite here to be aware of simultaneously camera and screen
                                    this.removeStream(streamId, remoteComponent)
                                }
                            };

                            // here we (asynchronously) get metadata by streamId from app server
                            this.signalLocal.call(USER_BY_STREAM_ID_METHOD, {streamId: streamId}).then(value => {
                                if (!value.found || !value.userDto) {
                                    console.error("Metadata by remote streamId=", streamId, " is not found on server");
                                } else {
                                    const data = value.userDto;
                                    console.debug("Successfully got data by streamId", streamId, data);
                                    remoteComponent.setUserName(data.login);
                                    remoteComponent.setDisplayAudioMute(data.audioMute);
                                    remoteComponent.setVideoMute(data.videoMute);
                                    remoteComponent.setAvatar(data.avatar);
                                    remoteComponent.setUserId(data.userId);
                                }
                            })
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
              delete this.remoteStreams[streamId];
            },
            tryRestartWithResetOncloseHandler() {
                if (this.signalLocal) {
                    this.signalLocal.onclose = null; // remove onclose handler with restart in order to prevent cyclic restarts
                }
                this.tryRestartVideoProcess();
            },
            clearLocalMediaStream() {
                if (this.localMediaStream) {
                    this.localMediaStream.getTracks().forEach(t => t.stop());
                    this.localMediaStream.unpublish();
                }
            },
            leaveSession() {
                for (const streamId in this.remoteStreams) {
                    console.log("Cleaning remote stream " + streamId);
                    const component = this.remoteStreams[streamId].component;
                    this.removeStream(streamId, component);
                }
                this.clearLocalMediaStream();
                if (this.clientLocal) {
                    this.clientLocal.close(); // also closes signal
                }
                this.clientLocal = null;
                this.signalLocal = null;
                this.remoteStreams = {};
                this.localMediaStream = null;
                this.isCnangingLocalStream = false;

                this.$store.commit(SET_MUTE_VIDEO, false);
                this.$store.commit(SET_MUTE_AUDIO, localAudioMutedInitial);
            },
            startHealthCheckPing() {
                if (true) {
                    return
                }
                console.log("Setting up ping every", pingInterval, "ms");
                pingTimerId = setInterval(()=>{
                    if (Object.keys(this.localStreams).length && !this.isCnangingLocalStream && !this.restartingStarted) { // TODO rewrite
                        const localStreamId = this.$refs.localVideoComponent.getStreamId();
                        console.debug("Checking self user", "streamId", localStreamId);
                        this.signalLocal.call(USER_BY_STREAM_ID_METHOD, {streamId: localStreamId, includeOtherStreamIds: true}).then(value => {
                            if (!value.found) {
                                console.warn("Detected absence of self user on server, restarting...", "streamId", localStreamId);
                                this.$refs.localVideoComponent.incrementFailureCount();
                                this.tryRestartWithResetOncloseHandler();
                            } else {
                                console.debug("Successfully checked self user", "streamId", localStreamId, value);

                                // check other
                                for (const streamId in this.remoteStreams) {
                                    console.debug("Checking remote streamId", streamId);
                                    const streamHolder = this.remoteStreams[streamId];
                                    const component = streamHolder.component;
                                    if (value.otherStreamIds.filter(v => v == streamId).length == 0) {
                                        component.incrementFailureCount();

                                        console.info("Other streamId", streamId, "is not present, failureCount icreased to", component.getFailureCount());
                                        if (component.getFailureCount() > MAX_MISSED_FAILURES) {
                                            console.debug("Other streamId", streamId, "subsequently is not present, removing...");
                                            this.removeStream(streamId, component);
                                        }
                                    } else {
                                        console.debug("Other streamId", streamId, "is present, resetting failureCount");
                                        component.resetFailureCount();
                                    }
                                }
                            }
                        })
                    } else {
                        console.debug("Skipping checking self user because it isn't ready");
                    }
                }, pingInterval)
            },
            getConfig() {
                return axios
                    .get(`/api/video/${this.chatId}/config`)
                    .then(response => response.data)
            },
            // TODO remove
            notifyWithData() {
                // notify another participants, they will receive VIDEO_CALL_CHANGED
                const toSend = {
                    avatar: this.currentUser.avatar,
                    peerId: this.peerId,
                    streamId: this.$refs.localVideoComponent.getStreamId(),
                    videoMute: this.videoMuted, // from store
                    audioMute: this.audioMuted
                };
                this.signalLocal.notify(PUT_USER_DATA_METHOD, toSend)
            },
            notifyWithData2(streamId) {
                // notify another participants, they will receive VIDEO_CALL_CHANGED
                const toSend = {
                    avatar: this.currentUser.avatar,
                    peerId: this.peerId,
                    streamId: streamId,
                    videoMute: this.videoMuted, // from store
                    audioMute: this.audioMuted
                };
                this.signalLocal.notify(PUT_USER_DATA_METHOD, toSend)
            },
            ensureAudioIsEnabledAccordingBrowserPolicies() {
                if (this.remoteVideoIsMuted) {
                    // Unmute all the current videoElements.
                    for (const streamHolder of Object.values(this.remoteStreams)) {
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
            // TODO rename to something 'addScreenSharingStream' and refactor
            onSwitchMediaStream({screen = false}) {
                this.isCnangingLocalStream = true;
                this.clearLocalMediaStream();
                this.$refs.localVideoComponent.setSource(null);
                this.localPublisherKey++;
                this.getAndPublishLocalMediaStream({screen})
                    .catch(reason => {
                      console.error("Error during publishing screen stream, won't restart...", reason);
                      this.$refs.localVideoComponent.setUserName('Error get getDisplayMedia');
                    });
            },
            getAndPublishLocalMediaStream({screen = false}) {
                this.isCnangingLocalStream = true;

                const resolution = getVideoResolution();

                const audio = getStoredAudioDevicePresents();
                const video = getStoredVideoDevicePresents();

                bus.$emit(VIDEO_PARAMETERS_CHANGED);

                if (!audio && !video && !screen) {
                    console.info("Not able to build local media stream, returning a successful promise");
                    this.$store.commit(SET_SHARE_SCREEN, false);
                    // this.isCnangingLocalStream = false;
                    return Promise.reject('No media configured');
                }

                const localStreamSpec = screen ?
                    LocalStream.getDisplayMedia({
                        audio: audio,
                        video: true,
                        codec: this.preferredCodec,
                    }) :
                    LocalStream.getUserMedia({
                        resolution: resolution,
                        audio: audio,
                        video: video,
                        codec: this.preferredCodec,
                    });

                return localStreamSpec.then((localMediaStream) => {
                  const streamId = localMediaStream.id;
                  const videoTagId = 'local-' + streamId + '-' + this.getNewId();
                  console.info("Setting local stream", streamId, " into video tag id=", videoTagId);
                  const localVideoComponent = this.appendUserVideo(localMediaStream, videoTagId, this.localStreams);
                  localVideoComponent.setSource(localMediaStream);
                  localVideoComponent.setStreamMuted(true); // tris is not error - we disable audio in local (own) video tag
                  localVideoComponent.setUserName(this.currentUser.login);
                  localVideoComponent.setAvatar(this.currentUser.avatar);
                  localVideoComponent.setUserId(this.currentUser.id);
                  console.log("Publishing " + (screen ? "screen" : "camera"));
                  this.clientLocal.publish(localMediaStream);
                  console.log("Successfully published " + (screen ? "screen" : "camera") + " streamId=", streamId);
                  if (screen) {
                      this.$store.commit(SET_SHARE_SCREEN, true);
                  } else {
                      this.$store.commit(SET_SHARE_SCREEN, false);
                  }

                  // actually during screen sharing there is no audio track - we calculate the actual audio muting state
                  let actualAudioMuted = true;
                  localMediaStream.getTracks().forEach(t => {
                      console.log("localMediaStream track kind=", t.kind, " trackId=", t.id, " local video tag id", localVideoComponent.$props.id, " streamId=", this.$refs.localVideoComponent.getStreamId());
                      if (t.kind === "audio") {
                          actualAudioMuted = t.muted;
                      }
                  });
                  this.$store.commit(SET_MUTE_AUDIO, actualAudioMuted);
                  this.$store.commit(SET_MUTE_VIDEO, !video);
                  localVideoComponent.setDisplayAudioMute(actualAudioMuted);
                  localVideoComponent.setVideoMute(!video);
                  localVideoComponent.resetFailureCount();
                  this.isCnangingLocalStream = false;

                  this.notifyWithData2(streamId);

                  return Promise.resolve({
                    component: localVideoComponent // TODO do we really need it because we already added in appendUserVideo()
                  })
                })
            },
            startVideoProcess() {
                this.getConfig()
                    .then(config => {
                        console.info("Config fetched, initializing to session...")
                        return this.joinSession(config);
                    })
                    .catch(reason => {
                        console.error("Error during get config, restarting...", reason)
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
                for (const streamId in this.localStreams) {
                    const streamHolder = this.localStreams[streamId];
                    if (streamHolder) {
                        if (requestedState) {
                            streamHolder.stream.mute("video");
                            streamHolder.component.setVideoMute(true);
                            this.$store.commit(SET_MUTE_VIDEO, requestedState);
                            this.notifyWithData2(streamId);
                        } else {
                            streamHolder.stream.unmute("video").then(value => {
                                streamHolder.component.setVideoMute(false);
                                this.$store.commit(SET_MUTE_VIDEO, requestedState);
                                this.notifyWithData2(streamId);
                            })
                        }
                    }
                }
            },
            onStartAudioMuting(requestedState) {
                this.ensureAudioIsEnabledAccordingBrowserPolicies();

                for (const streamId in this.localStreams) {
                    const streamHolder = this.localStreams[streamId];
                    if (streamHolder) {
                        if (requestedState) {
                            streamHolder.stream.mute("audio");
                            this.$store.commit(SET_MUTE_AUDIO, requestedState);
                            streamHolder.component.setDisplayAudioMute(requestedState);
                            this.notifyWithData2(streamId);
                        } else {
                            streamHolder.stream.unmute("audio").then(value => {
                                this.$store.commit(SET_MUTE_AUDIO, requestedState);
                                streamHolder.component.setDisplayAudioMute(requestedState);
                                this.notifyWithData2(streamId);
                            })
                        }
                    }
                }
            },
            onForceMuteByAdmin(dto) {
                if(dto.chatId == this.chatId) {
                    this.onStartAudioMuting(true);
                }
            },
            onVideoCallChanged(dto) {
                // this method reacts only when data present - in this case it contains changes for particular stream id
                if (dto) {
                    const data = dto.data;
                    if (data) {
                        const streamHolder = this.remoteStreams[data.streamId];
                        if (streamHolder) {
                            streamHolder.component.setDisplayAudioMute(data.audioMute);
                            streamHolder.component.setVideoMute(data.videoMute);
                        }
                    }
                }
            },
            onVideoParametersChanged() {
                this.tryRestartWithResetOncloseHandler();
            },
            enumerateAllStreams(callback) {
                for (const streamId in this.localStreams) {
                    const streamHolder = this.localStreams[streamId];
                    if (streamHolder) {
                        callback(streamHolder.component, streamId);
                    }
                }

                for (const streamId in this.remoteStreams) {
                    const streamHolder = this.remoteStreams[streamId];
                    if (streamHolder) {
                        callback(streamHolder.component, streamId);
                    }
                }
            },
            applyCallbackToStreamId(streamId, callback) {
                let streamHolder = this.localStreams[streamId];
                if (streamHolder) {
                    callback(streamHolder.component);
                    return;
                }

                streamHolder = this.remoteStreams[streamId];
                if (streamHolder) {
                    callback(streamHolder.component);
                }
            },
            getNewId() {
                return uuidv4();
            },
            onVideoCallKicked(e) {
                if (this.$route.name == videochat_name && e.chatId == this.chatId) {
                    console.log("kicked");
                    this.$router.push({name: chat_name});
                }
            },
            onUserProfileChanged(user) {
                this.enumerateAllStreams((component) => {
                    const cid = component.getUserId();
                    if (cid && cid == user.id) {
                        component.setAvatar(user.avatar);
                    }
                })
            },
        },
        mounted() {
            this.chatId = this.$route.params.id;

            this.closingStarted = false;
            this.$store.commit(SET_SHOW_CALL_BUTTON, false);
            this.$store.commit(SET_SHOW_HANG_BUTTON, true);
            window.addEventListener('beforeunload', this.leaveSession);
            this.remotesDiv = document.getElementById("video-container");
            this.startHealthCheckPing();
            this.startVideoProcess();
        },
        beforeDestroy() {
            this.$store.commit(SET_SHOW_CALL_BUTTON, true);
            this.$store.commit(SET_SHOW_HANG_BUTTON, false);
            this.$store.commit(SET_SHARE_SCREEN, false);
            this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, 0);

            this.closingStarted = true;
            window.removeEventListener('beforeunload', this.leaveSession);
            if (pingTimerId) {
                console.log("Clearing self healthcheck timer");
                clearInterval(pingTimerId);
            }
            this.leaveSession();
            this.remotesDiv = null;
        },
        created() {
            bus.$on(SHARE_SCREEN_START, this.onStartScreenSharing);
            bus.$on(SHARE_SCREEN_STOP, this.onStopScreenSharing);
            bus.$on(VIDEO_START_MUTING, this.onStartVideoMuting);
            bus.$on(AUDIO_START_MUTING, this.onStartAudioMuting);
            bus.$on(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
            bus.$on(REQUEST_CHANGE_VIDEO_PARAMETERS, this.onVideoParametersChanged);
            bus.$on(USER_PROFILE_CHANGED, this.onUserProfileChanged);
        },
        destroyed() {
            bus.$off(SHARE_SCREEN_START, this.onStartScreenSharing);
            bus.$off(SHARE_SCREEN_STOP, this.onStopScreenSharing);
            bus.$off(VIDEO_START_MUTING, this.onStartVideoMuting);
            bus.$off(AUDIO_START_MUTING, this.onStartAudioMuting);
            bus.$off(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
            bus.$off(REQUEST_CHANGE_VIDEO_PARAMETERS, this.onVideoParametersChanged);
            bus.$off(USER_PROFILE_CHANGED, this.onUserProfileChanged);
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