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
    </v-col>
</template>

<script>
    import Vue from 'vue';
    import {mapGetters} from "vuex";
    import {
        GET_USER,
        SET_SHOW_CALL_BUTTON, SET_SHOW_HANG_BUTTON,
        SET_VIDEO_CHAT_USERS_COUNT
    } from "./store";
    import bus, {
        ADD_SCREEN_SOURCE, ADD_VIDEO_SOURCE,
        REQUEST_CHANGE_VIDEO_PARAMETERS,
        USER_PROFILE_CHANGED, VIDEO_CALL_CHANGED, VIDEO_PARAMETERS_CHANGED,
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
    const USER_BY_STREAM_ID_METHOD = "userByStreamId";
    const GET_ALIVE_STREAM_IDS_METHOD = "getAliveStreamIds";

    const KICK_NOTIFICATION = "kick";
    const FORCE_MUTE_NOTIFICATION = "force_mute";

    const pingInterval = 5000;
    const videoProcessRestartInterval = 1000;
    const MAX_MISSED_FAILURES = 5;

    export default {
        data() {
            return {
                clientLocal: null,
                localStreams: {}, // user can have several cameras, or simultaneously translate camera and screen
                remoteStreams: {},
                videoContainerDiv: null,
                signalLocal: null,
                chatId: null,
                remoteVideoIsMuted: true,
                peerId: null,

                // this one is about skipping ping checks during changing media stream
                isChangingLocalStream: false,

                // this two are about restart process
                restartingStarted: false,
                closingStarted: false,

                showPermissionAsk: true,

                errorDescription: null,

                preferredCodec: null,
                simulcast: false,
            }
        },
        props: ['chatDto'],
        computed: {
            ...mapGetters({
                currentUser: GET_USER,
            }),
        },
        methods: {
            onClickPermitted() {
                this.ensureAudioIsEnabledAccordingBrowserPolicies();
                this.showPermissionAsk = false;
            },
            appendUserVideo(prepend, stream, videoTagId, appendTo, localVideoProperties) {
                const component = new UserVideoClass({vuetify: vuetify, propsData: { initialMuted: this.remoteVideoIsMuted, id: videoTagId, localVideoProperties: localVideoProperties }});
                component.$mount();
                if (prepend) {
                    this.videoContainerDiv.prepend(component.$el);
                } else {
                    this.videoContainerDiv.appendChild(component.$el);
                }
                component.setStream(stream);
                appendTo[stream.id] = {stream, component};
                return component;
            },
            joinSession(configObj) {
                console.info("Used webrtc config", JSON.stringify(configObj));

                this.signalLocal = new IonSFUJSONRPCSignal(
                    getWebsocketUrlPrefix()+`/api/video/${this.chatId}/ws`
                );

                this.simulcast = configObj.simulcast;
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
                            const remoteComponent = this.appendUserVideo(false, stream, videoTagId, this.remoteStreams, null);
                            stream.onremovetrack = (e) => {
                                console.log("onremovetrack", e);
                                if (e.track) {
                                    this.removeStream(streamId, remoteComponent, this.remoteStreams)
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
            removeStream(streamId, component, removeFrom) {
              console.log("Removing stream streamId=", streamId);
              try {
                this.videoContainerDiv.removeChild(component.$el);
                component.$destroy();
              } catch (e) {
                console.debug("Something wrong on removing child", e, component.$el, this.videoContainerDiv);
              }
              delete removeFrom[streamId];
            },
            tryRestartWithResetOncloseHandler() {
                if (this.signalLocal) {
                    this.signalLocal.onclose = null; // remove onclose handler with restart in order to prevent cyclic restarts
                }
                this.tryRestartVideoProcess();
            },
            clearLocalMediaStream(localMediaStream) {
                if (localMediaStream) {
                    localMediaStream.getTracks().forEach(t => t.stop());
                    localMediaStream.unpublish();
                }
            },
            leaveSession() { // we won't do restart particular stream in case error, we are gonna restart all
                for (const streamId in this.remoteStreams) {
                    console.log("Cleaning remote stream " + streamId);
                    const component = this.remoteStreams[streamId].component;
                    this.removeStream(streamId, component, this.remoteStreams);
                }

                for (const streamId in this.localStreams) {
                    console.log("Cleaning local stream " + streamId);
                    const streamHolder = this.localStreams[streamId];
                    this.clearLocalMediaStream(streamHolder.stream);
                    this.removeStream(streamId, streamHolder.component, this.localStreams);
                }

                if (this.clientLocal) {
                    this.clientLocal.close(); // also closes signal
                }
                this.clientLocal = null;
                this.signalLocal = null;
                this.remoteStreams = {};
                this.isChangingLocalStream = false;
            },
            startHealthCheckPing() {
                console.log("Setting up ping every", pingInterval, "ms");
                pingTimerId = setInterval(()=>{
                    if (!this.isChangingLocalStream && !this.restartingStarted) {
                        console.debug("Checking local streams");
                        for (const localStreamId in this.localStreams) {
                            console.debug("Checking self user", "streamId", localStreamId);
                            const streamHolder = this.localStreams[localStreamId]
                            if (streamHolder) {
                                this.signalLocal.call(USER_BY_STREAM_ID_METHOD, {
                                    streamId: localStreamId,
                                }).then(value => {
                                    if (!value.found) {
                                        console.warn("Detected absence of self user on server, restarting...", "streamId", localStreamId);
                                        streamHolder.component.incrementFailureCount();
                                        this.tryRestartWithResetOncloseHandler();
                                    } else {
                                        console.debug("Successfully checked self user", "streamId", localStreamId, value);
                                    }
                                });
                            } else {
                                console.warn("streamHolder is not present for", "streamId", localStreamId, value);
                            }
                        }

                        console.debug("Checking remote streams");
                        this.signalLocal.call(GET_ALIVE_STREAM_IDS_METHOD, { }).then((value) => {
                            // check other
                            for (const streamId in this.remoteStreams) {
                                console.debug("Checking remote streamId", streamId);
                                const streamHolder = this.remoteStreams[streamId];
                                const component = streamHolder.component;
                                if (value.aliveStreamIds.filter(v => v == streamId).length == 0) {
                                    component.incrementFailureCount();

                                    console.info("Other streamId", streamId, "is not present, failureCount icreased to", component.getFailureCount());
                                    if (component.getFailureCount() > MAX_MISSED_FAILURES) {
                                        console.debug("Other streamId", streamId, "subsequently is not present, removing...");
                                        this.removeStream(streamId, component, this.remoteStreams);
                                    }
                                } else {
                                    console.debug("Other streamId", streamId, "is present, resetting failureCount");
                                    component.resetFailureCount();
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
            onAddScreenSource() {
                this.getAndPublishLocalMediaStream({screen: true})
                    .catch(reason => {
                        console.error("Error during publishing screen stream, won't restart...", reason);
                        this.errorDescription = reason;
                    });
            },
            onAddVideoSource(videoId, audioId) {
                this.getAndPublishLocalMediaStream({screen: false, videoId, audioId})
                    .catch(reason => {
                        console.error("Error during publishing screen stream, won't restart...", reason);
                        this.errorDescription = reason;
                    });
            },
            getAndPublishLocalMediaStream({screen = false, videoId = null, audioId = null}) {
                this.isChangingLocalStream = true;

                const resolution = getVideoResolution();

                const audio = getStoredAudioDevicePresents();
                const video = getStoredVideoDevicePresents();

                bus.$emit(VIDEO_PARAMETERS_CHANGED);

                if (!audio && !video && !screen) {
                    console.info("Not able to build local media stream, returning a successful promise");
                    // this.isChangingLocalStream = false;
                    return Promise.reject('No media configured');
                }

                const audioConstraints = audioId ? { deviceId: audioId } : audio;
                const videoConstraints = videoId ?  { deviceId: videoId } : video;
                console.info("Selected constraints", "video", videoConstraints, "audio", audioConstraints);

                const localStreamSpec = screen ?
                    LocalStream.getDisplayMedia({
                        audio: audio,
                        video: true,
                        codec: this.preferredCodec,
                        simulcast: this.simulcast,
                    }) :
                    LocalStream.getUserMedia({
                        resolution: resolution,
                        audio: audioConstraints,
                        video: videoConstraints,
                        codec: this.preferredCodec,
                        simulcast: this.simulcast,
                    });

                return localStreamSpec.then((localMediaStream) => {
                  const streamId = localMediaStream.id;
                  const videoTagId = 'local-' + streamId + '-' + this.getNewId();
                  console.info("Setting local stream", streamId, " into video tag id=", videoTagId);
                  const localVideoComponent = this.appendUserVideo(true, localMediaStream, videoTagId, this.localStreams, {
                      peerId: this.peerId,
                      signalLocal: this.signalLocal,
                      parent: this
                  });
                  localVideoComponent.setStream(localMediaStream);
                  localVideoComponent.setStreamMuted(true); // tris is not error - we disable audio in local (own) video tag
                  localVideoComponent.setUserName(this.currentUser.login);
                  localVideoComponent.setAvatar(this.currentUser.avatar);
                  localVideoComponent.setUserId(this.currentUser.id);
                  console.log("Publishing " + (screen ? "screen" : "camera"));
                  this.clientLocal.publish(localMediaStream);
                  console.log("Successfully published " + (screen ? "screen" : "camera") + " streamId=", streamId);

                  // actually during screen sharing there is no audio track - we calculate the actual audio muting state
                  let actualAudioMuted = true;
                  localMediaStream.getTracks().forEach(t => {
                      console.log("localMediaStream track kind=", t.kind, " trackId=", t.id, " local video tag id", localVideoComponent.$props.id, " streamId=", localVideoComponent.getStreamId());
                      if (t.kind === "audio") {
                          actualAudioMuted = t.muted;
                      }
                  });
                  localVideoComponent.setDisplayAudioMute(actualAudioMuted);
                  localVideoComponent.setVideoMute(!videoConstraints);
                  localVideoComponent.resetFailureCount();
                  this.isChangingLocalStream = false;
                  localVideoComponent.notifyOtherParticipants();

                  return Promise.resolve(true)
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
            onForceMuteByAdmin(dto) {
                if(dto.chatId == this.chatId) {
                    for (const streamId in this.localStreams) {
                        const streamHolder = this.localStreams[streamId];
                        if (streamHolder) {
                            streamHolder.component.doMuteAudio(true);
                        }
                    }
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
            onGlobalVideoParametersChanged() {
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
            this.videoContainerDiv = document.getElementById("video-container");
            this.startHealthCheckPing();
            this.startVideoProcess();
        },
        beforeDestroy() {
            this.$store.commit(SET_SHOW_CALL_BUTTON, true);
            this.$store.commit(SET_SHOW_HANG_BUTTON, false);
            this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, 0);

            this.closingStarted = true;
            window.removeEventListener('beforeunload', this.leaveSession);
            if (pingTimerId) {
                console.log("Clearing self healthcheck timer");
                clearInterval(pingTimerId);
            }
            this.leaveSession();
            this.videoContainerDiv = null;
        },
        created() {
            bus.$on(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
            bus.$on(REQUEST_CHANGE_VIDEO_PARAMETERS, this.onGlobalVideoParametersChanged);
            bus.$on(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$on(ADD_SCREEN_SOURCE, this.onAddScreenSource);
            bus.$on(ADD_VIDEO_SOURCE, this.onAddVideoSource);
        },
        destroyed() {
            bus.$off(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
            bus.$off(REQUEST_CHANGE_VIDEO_PARAMETERS, this.onGlobalVideoParametersChanged);
            bus.$off(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$off(ADD_SCREEN_SOURCE, this.onAddScreenSource);
            bus.$off(ADD_VIDEO_SOURCE, this.onAddVideoSource);
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
        overflow-x: scroll;
        overflow-y: hidden;
        scrollbar-width: none;
        //scroll-snap-align width
        //scroll-padding 0
        height 100%
        width 100%
        //object-fit: contain;
        //box-sizing: border-box
    }

</style>