<template>
    <v-col cols="12" class="ma-0 pa-0" id="video-container">
        <UserVideo ref="localVideoComponent" :key="localPublisherKey"/>
    </v-col>
</template>

<script>
    import Vue from 'vue';
    import {mapGetters} from "vuex";
    import {GET_MUTE_AUDIO, GET_MUTE_VIDEO, GET_USER, SET_MUTE_AUDIO, SET_MUTE_VIDEO} from "./store";
    import bus, {
        AUDIO_START_MUTING,
        CHANGE_PHONE_BUTTON,
        SHARE_SCREEN_START, SHARE_SCREEN_STATE_CHANGED,
        SHARE_SCREEN_STOP, VIDEO_CALL_CHANGED,
        VIDEO_COMPONENT_DESTROYED,
        VIDEO_LOCAL_ESTABLISHED, VIDEO_START_MUTING
    } from "./bus";
    import {phoneFactory} from "./changeTitle";
    import axios from "axios";
    import { Client, LocalStream } from 'ion-sdk-js';
    import { IonSFUJSONRPCSignal } from 'ion-sdk-js/lib/signal/json-rpc-impl';
    import UserVideo from "./UserVideo";
    import {getWebsocketUrlPrefix} from "./utils";
    import { v4 as uuidv4 } from 'uuid';

    const ComponentClass = Vue.extend(UserVideo);

    const peerId = uuidv4();

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
                closingStarted: false
            }
        },
        props: ['chatDto'],
        computed: {
            chatId() {
                return this.$route.params.id
            },
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

                this.signalLocal.onerror = () => { console.error("Error in signal"); }
                this.signalLocal.onclose = () => {
                  console.info("Signal is closed, something gonna happen");
                  this.tryRestartVideoProcess();
                }

                this.signalLocal.onopen = () => {
                    this.clientLocal.join(`chat${this.chatId}`, peerId).then(()=>{
                        this.getAndPublishCamera()
                            .then(()=>{
                              this.notifyAboutJoining();
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
                    if (track.kind === "video") {
                        if (!this.streams[stream.id]) {
                            console.log("set track", track.id, "for stream", stream.id);

                            const component = new ComponentClass();
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
            leaveSession() {
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

                bus.$emit(VIDEO_COMPONENT_DESTROYED); // restore initial state in App.vue
                this.$store.commit(SET_MUTE_VIDEO, false);
                this.$store.commit(SET_MUTE_AUDIO, false);

                this.notifyAboutLeaving();
            },
            askUserNameWithRetries(streamId) {
                // request-response with axios and error handling
                axios.get(`/api/video/${this.chatId}/user?streamId=${streamId}`)
                .then(value => {
                    if (value.status == 204) {
                        if (!this.closingStarted) {
                            console.log("Rescheduling asking for userName");
                            setTimeout(() => {
                              this.askUserNameWithRetries(streamId);
                            }, 1000);
                        }
                    } else {
                        const data = value.data;
                        if (data) {
                            const component = this.streams[data.streamId];
                            if (component) {
                                component.component.setUserName(data.login);
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
            notifyAboutLeaving() {
                if (this.chatId) {
                    axios.put(`/api/video/${this.chatId}/notify`).catch(error => {
                      console.log(error.response)
                    });
                } else {
                    console.warn("Unable to notify about leaving")
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
                    });
            },
            onStopScreenSharing() {
                this.localMedia.unpublish();
                if (this.localMedia) {
                  this.localMedia.getTracks().forEach(t => t.stop());
                }
                this.$refs.localVideoComponent.setSource(null);
                this.localPublisherKey++;
                this.getAndPublishCamera();
            },
            getAndPublishCamera() {
                return LocalStream.getUserMedia({
                  resolution: "hd",
                  audio: true,
                }).then((media) => {
                  this.localMedia = media
                  this.$refs.localVideoComponent.setSource(media);
                  this.$refs.localVideoComponent.setMuted(true);
                  this.$refs.localVideoComponent.setUserName(this.myUserName);
                  this.clientLocal.publish(media);
                  bus.$emit(SHARE_SCREEN_STATE_CHANGED, false);
                });
            },
            getAndPublishScreen() {
                return LocalStream.getDisplayMedia({
                  audio: true,
                }).then((media) => {
                    this.localMedia = media;
                    this.localMedia.unmute("audio");
                    this.$refs.localVideoComponent.setSource(media);
                    this.$refs.localVideoComponent.setMuted(true);
                    this.$refs.localVideoComponent.setUserName(this.myUserName)
                    this.clientLocal.publish(media);
                    bus.$emit(SHARE_SCREEN_STATE_CHANGED, true);
                });
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
              }, 1000);
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
        },
        mounted() {
            this.closingStarted = false;
            bus.$emit(VIDEO_LOCAL_ESTABLISHED);
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, false));
            window.addEventListener('beforeunload', this.leaveSession)
            this.startVideoProcess();
        },
        beforeDestroy() {
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, true));
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
        },
        destroyed() {
            bus.$off(SHARE_SCREEN_START, this.onStartScreenSharing);
            bus.$off(SHARE_SCREEN_STOP, this.onStopScreenSharing);
            bus.$off(VIDEO_START_MUTING, this.onStartVideoMuting);
            bus.$off(AUDIO_START_MUTING, this.onStartAudioMuting);
            bus.$on(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
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