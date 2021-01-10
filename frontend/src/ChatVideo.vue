<template>
    <v-col cols="12" class="ma-0 pa-0" id="video-container">
        <user-video class="video-container-element-my" :stream-manager="publisher"/>
        <user-video v-for="sub in subscribers" :key="sub.stream.connection.connectionId" :stream-manager="sub"/>
    </v-col>
</template>

<script>
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import bus, {
        CHANGE_PHONE_BUTTON,
        VIDEO_LOCAL_ESTABLISHED
    } from "./bus";
    import {phoneFactory} from "./changeTitle";
    import axios from "axios";
    import {getWebsocketUrlPrefix} from "./utils";

    import { OpenVidu } from 'openvidu-browser';
    import UserVideo from './UserVideo';

    const OPENVIDU_SERVER_URL = "/api/video";
    const OPENVIDU_SERVER_SECRET = "MY_SECRET";

    export default {
        components: {
            UserVideo,
        },
        data() {
            return {
                OV: undefined,
                session: undefined,
                publisher: undefined,
                subscribers: []
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
            joinSession() {
                // --- Get an OpenVidu object ---
                this.OV = new OpenVidu();
                this.OV.setAdvancedConfiguration({forceMediaReconnectionAfterNetworkDrop: true})

                // --- Init a session ---
                const sess = this.OV.initSession();

                const oldProcessTokenFunction = sess.constructor.prototype.processToken;
                sess.constructor.prototype.processToken = function (token) {
                    oldProcessTokenFunction.call(this, token);

                    this.openvidu.wsUri = getWebsocketUrlPrefix()+'/api/video/openvidu';
                    this.openvidu.httpUri = '/api/video';
                };

                this.session = sess;

                // --- Specify the actions when events take place in the session ---

                // On every new Stream received...
                this.session.on('streamCreated', ({ stream }) => {
                    const subscriber = this.session.subscribe(stream);
                    this.subscribers.push(subscriber);
                });

                // On every Stream destroyed...
                this.session.on('streamDestroyed', ({ stream }) => {
                    const index = this.subscribers.indexOf(stream.streamManager, 0);
                    if (index >= 0) {
                        this.subscribers.splice(index, 1);
                    }
                });

                // --- Connect to the session with a valid user token ---

                // 'getToken' method is simulating what your server-side should do.
                // 'token' parameter should be retrieved and returned by your own backend
                this.getToken().then(token => {
                    this.session.connect(token, { clientData: this.myUserName })
                        .then(() => {

                            // --- Get your own camera stream with the desired properties ---

                            let publisher = this.OV.initPublisher(undefined, {
                                audioSource: undefined, // The source of audio. If undefined default microphone
                                videoSource: undefined, // The source of video. If undefined default webcam
                                publishAudio: true,  	// Whether you want to start publishing with your audio unmuted or not
                                publishVideo: true,  	// Whether you want to start publishing with your video enabled or not
                                resolution: '640x480',  // The resolution of your video
                                frameRate: 30,			// The frame rate of your video
                                insertMode: 'APPEND',	// How the video is inserted in the target element 'video-container'
                                mirror: false       	// Whether to mirror your local video or not
                            });

                            this.publisher = publisher;

                            // --- Publish your stream ---

                            this.session.publish(this.publisher);
                        })
                        .catch(error => {
                            console.log('There was an error connecting to the session:', error.code, error.message);
                        });
                });

                window.addEventListener('beforeunload', this.leaveSession)
            },

            leaveSession() {
                // --- Leave the session by calling 'disconnect' method over the Session object ---
                if (this.session) this.session.disconnect();

                this.session = undefined;
                this.publisher = undefined;
                this.subscribers = [];
                this.OV = undefined;

                window.removeEventListener('beforeunload', this.leaveSession);
            },
            getToken() {
                return new Promise((resolve, reject) => {
                    axios
                        .post(`/api/chat/${this.chatId}/token`)
                        .then(response => response.data)
                        .then(data => resolve(data.token))
                        .catch(error => reject(error.response));
                });
            },
        },
        mounted() {
            bus.$emit(VIDEO_LOCAL_ESTABLISHED);
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, false));

            this.joinSession();
        },

        beforeDestroy() {
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, true));

            this.leaveSession();
        },
    }
</script>

<style lang="stylus">
    #video-container {
        display: flex;
        flex-direction: row;
        overflow-x: auto;
        overflow-y: hidden;
        height 100%
    }

    .video-container-element {
        display flex
        flex-direction column
        object-fit: scale-down;
        height 100% !important
        width 100% !important
    }

    .video-container-element-my {
        background #b3e7ff
    }

    .video-container-element:nth-child(even) {
        background #d5fdd5;
    }

    video {
        //object-fit: scale-down;
        //width 100% !important
        height 100% !important // todo its
    }

    .video-container-element-caption {
        top -1.8em
        left 2em
        text-shadow: -2px 0 white, 0 2px white, 2px 0 white, 0 -2px white;
        position: relative;
    }
</style>