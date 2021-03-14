<template>
    <v-col cols="12" class="ma-0 pa-0" id="video-container">
        <UserVideo ref="localVideoComponent" :key="localPublisherKey"/>
    </v-col>
</template>

<script>
    import Vue from 'vue';
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import bus, {
        AUDIO_MUTED,
        AUDIO_START_MUTING,
        CHANGE_PHONE_BUTTON,
        SHARE_SCREEN_START, SHARE_SCREEN_STATE_CHANGED,
        SHARE_SCREEN_STOP,
        VIDEO_COMPONENT_DESTROYED,
        VIDEO_LOCAL_ESTABLISHED, VIDEO_MUTED, VIDEO_START_MUTING
    } from "./bus";
    import {phoneFactory} from "./changeTitle";
    import axios from "axios";
    import { Client, LocalStream } from 'ion-sdk-js';
    import { IonSFUJSONRPCSignal } from 'ion-sdk-js/lib/signal/json-rpc-impl';
    import UserVideo from "./UserVideo";
    import {getWebsocketUrlPrefix} from "./utils";
    const ComponentClass = Vue.extend(UserVideo);

    const DATA_EVENT_GET_USERNAME_FOR = "getUserName";
    const DATA_EVENT_RESPOND_USERNAME = "respondUserName";

    const FIELD_TYPE = "type";
    const FIELD_STREAM_ID = "streamId";
    const FIELD_FOR_STREAM_ID = "forStreamId";
    const FIELD_USERNAME = "username";

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
            ...mapGetters({currentUser: GET_USER}),
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
                    this.clientLocal.join(`chat${this.chatId}`).then(()=>{
                        this.dataChannel = this.clientLocal.createDataChannel(`chat${this.chatId}`);
                        this.dataChannel.onmessage = this.receiveFromChannel;
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
                this.notifyAboutLeaving();
            },
            askUserNameWithRetries(streamId) {
                const toSend = {[FIELD_TYPE]: DATA_EVENT_GET_USERNAME_FOR, [FIELD_STREAM_ID]: streamId};
                try {
                    this.sendToChannel(toSend);
                } catch (e) {
                    setTimeout(()=>{
                        console.log("Rescheduling asking for userName");
                        this.askUserNameWithRetries(streamId);
                    }, 1000);
                }
            },
            sendToChannel(obj) {
                const toSend = JSON.stringify(obj);
                console.log("Sending to data channel", toSend)
                this.dataChannel.send(toSend);
            },
            receiveFromChannel(m) {
                const data = JSON.parse(m.data);
                console.log("Received from data channel", m.data);
                if (data[FIELD_TYPE] == DATA_EVENT_GET_USERNAME_FOR && data[FIELD_STREAM_ID] == this.$refs.localVideoComponent.getStreamId()) {
                    this.sendToChannel({[FIELD_TYPE]: DATA_EVENT_RESPOND_USERNAME, [FIELD_USERNAME]: this.myUserName, [FIELD_FOR_STREAM_ID]: data[FIELD_STREAM_ID]});
                } else if (data[FIELD_TYPE] == DATA_EVENT_RESPOND_USERNAME) {
                    const component = this.streams[data[FIELD_FOR_STREAM_ID]];
                    if (component) {
                        component.component.setUserName(data[FIELD_USERNAME]);
                    }
                }
            },
            getConfig() {
                return axios
                    .get(`/api/video/${this.chatId}/config`)
                    .then(response => response.data)
            },

            notifyAboutJoining() {
                if (this.chatId) {
                    axios.put(`/api/video/${this.chatId}/notify`).catch(error => {
                      console.log(error.response)
                    })
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
                  this.$refs.localVideoComponent.setUserName(this.myUserName)
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
                    bus.$emit(VIDEO_MUTED, requestedState);
                } else {
                    this.localMedia.unmute("video").then(value => {
                        bus.$emit(VIDEO_MUTED, requestedState);
                    })
                }
            },
            onStartAudioMuting(requestedState) {
                if (requestedState) {
                    this.localMedia.mute("audio");
                    bus.$emit(AUDIO_MUTED, requestedState);
                } else {
                    this.localMedia.unmute("audio").then(value => {
                        bus.$emit(AUDIO_MUTED, requestedState);
                    })
                }
            }
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
        },
        destroyed() {
            bus.$off(SHARE_SCREEN_START, this.onStartScreenSharing);
            bus.$off(SHARE_SCREEN_STOP, this.onStopScreenSharing);
            bus.$off(VIDEO_START_MUTING, this.onStartVideoMuting);
            bus.$off(AUDIO_START_MUTING, this.onStartAudioMuting);
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