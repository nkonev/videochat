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
        CHANGE_PHONE_BUTTON,
        SHARE_SCREEN_START, SHARE_SCREEN_STATE_CHANGED,
        SHARE_SCREEN_STOP,
        VIDEO_CALL_CHANGED,
        VIDEO_LOCAL_ESTABLISHED
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
                localPublisherKey: 1
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
                    iceServers: [
                        {
                            urls: configObj.urls,
                        },
                    ],
                };
                this.signalLocal = new IonSFUJSONRPCSignal(
                    getWebsocketUrlPrefix()+`/api/video/ws?chatId=${this.chatId}`
                );
                this.remotesDiv = document.getElementById("video-container");

                this.clientLocal = new Client(this.signalLocal, config);

                this.signalLocal.onopen = () => {
                    this.clientLocal.join(`chat${this.chatId}`).then(()=>{
                        this.dataChannel = this.clientLocal.createDataChannel(`chat${this.chatId}`);
                        this.dataChannel.onmessage = this.receiveFromChannel;
                        this.getAndPublishCamera()
                            .then(()=>{
                              this.notifyAboutJoining();
                            })
                            .catch(console.error);
                    })
                }
                this.signalLocal.onerror = () => { console.error("Error in signal"); }
                this.signalLocal.onclose = () => { console.info("Signal is closed"); }

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
                                this.streams[stream.id] = null;
                                console.log("removed track", track.id, "for stream", stream.id);
                                try {
                                    this.remotesDiv.removeChild(component.$el);
                                    component.$destroy();
                                } catch (e) {
                                    console.debug("Something wrong on removing child", e, component.$el, this.remotesDiv);
                                }
                            };

                            this.askUserNameWithRetries(stream.id);
                        }
                    }
                };

                window.addEventListener('beforeunload', this.leaveSession)
            },
            leaveSession() {
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

                bus.$emit(VIDEO_CALL_CHANGED, {usersCount: 0}); // restore initial state
                this.notifyAboutLeaving();
                window.removeEventListener('beforeunload', this.leaveSession);
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
                console.log("Sending", toSend)
                this.dataChannel.send(toSend);
            },
            receiveFromChannel(m) {
                const data = JSON.parse(m.data);
                console.log("Received", m.data);
                if (data[FIELD_TYPE] == DATA_EVENT_GET_USERNAME_FOR && data[FIELD_STREAM_ID] == this.$refs.localVideoComponent.getStreamId()) {
                    this.sendToChannel({[FIELD_TYPE]: DATA_EVENT_RESPOND_USERNAME, [FIELD_USERNAME]: this.myUserName, [FIELD_FOR_STREAM_ID]: data[FIELD_STREAM_ID]});
                } else if (data.type == DATA_EVENT_RESPOND_USERNAME) {
                    const component = this.streams[data[FIELD_FOR_STREAM_ID]];
                    if (component) {
                        component.component.setUserName(data[FIELD_USERNAME]);
                    }
                }
            },
            getConfig() {
                return axios
                    .get(`/api/video/config`)
                    .then(response => response.data)
            },

            notifyAboutJoining() {
                if (this.chatId) {
                    axios.put(`/api/video/notify?chatId=${this.chatId}`);
                }
            },
            notifyAboutLeaving() {
                if (this.chatId) {
                    axios.put(`/api/video/notify?chatId=${this.chatId}`);
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
                    .catch(console.error);
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
                  resolution: "vga",
                  audio: true,
                }).then((media) => {
                  this.localMedia = media
                  this.$refs.localVideoComponent.setSource(media);
                  this.$refs.localVideoComponent.setUserName(this.myUserName)
                  this.clientLocal.publish(media);
                  bus.$emit(SHARE_SCREEN_STATE_CHANGED, false);
                });
            },
            getAndPublishScreen() {
                return LocalStream.getDisplayMedia({
                  audio: true,
                }).then((media) => {
                    this.localMedia = media
                    this.$refs.localVideoComponent.setSource(media);
                    this.$refs.localVideoComponent.setUserName(this.myUserName)
                    this.clientLocal.publish(media);
                    bus.$emit(SHARE_SCREEN_STATE_CHANGED, true);
                });
            }

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