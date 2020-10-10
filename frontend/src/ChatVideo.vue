<template>
    <v-col cols="12" class="ma-0 pa-0" id="video-container">
        <div class="video-container-element video-container-element-my">
            <video id="localVideo" autoPlay playsInline></video>
            <p class="video-container-element-caption">{{ currentUser.login }}</p>
        </div>
        <div class="video-container-element" v-for="(item, index) in properParticipants" :key="item.id">
            <video :id="getRemoteVideoId(item.id)" autoPlay playsInline :class="otherParticipantsClass" :poster="getAvatar(item)"></video>
            <p class="video-container-element-caption">{{ getLogin(item) }}</p>
        </div>
    </v-col>
</template>

<script>
    import {getData, getProperData, setProperData} from "./centrifugeConnection";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import bus, {
        CHANGE_PHONE_BUTTON,
        VIDEO_LOCAL_ESTABLISHED
    } from "./bus";
    import {phoneFactory} from "./changeTitle";
    import axios from "axios";
    import Vue from 'vue'

    const EVENT_CANDIDATE = 'candidate';
    const EVENT_HELLO = 'hello';
    const EVENT_BYE = 'bye';
    const EVENT_OFFER = 'offer';
    const EVENT_ANSWER = 'answer';

    export default {
        data() {
            return {
                prevVideoPaneSize: null,

                signalingSubscription: null,

                pcConfig: null,

                localStream: null,
                localVideo: null,

                remoteConnectionData: [
                    // userId: number
                    // peerConnection: RTCPeerConnection
                    // remoteVideo: html element
                ],
            }
        },
        props: ['chatDto'],
        computed: {
            chatId() {
                return this.$route.params.id
            },
            ...mapGetters({currentUser: GET_USER}),
            otherParticipantsClass() {
                if (!this.localStream) {
                    return "order-first"
                } else {
                    return ""
                }
            },
            properParticipants() {
                const ppi = this.chatDto.participants.filter(pi => pi.id != this.currentUser.id);
                console.log("Participant ids except me:", ppi);
                return ppi;
            },
        },
        methods: {
            getRemoteVideoId(participantId) {
                return 'remoteVideo'+participantId;
            },
            getWebRtcConfiguration() {
                const localPcConfig = {
                    iceServers: []
                };
                axios.get("/api/chat/public/webrtc/config").then(({data}) => {
                    for (const srv of data) {
                        localPcConfig.iceServers.push({
                            'urls': srv
                        });
                        this.pcConfig = localPcConfig;
                        console.log("Configured WebRTC servers", this.pcConfig);
                    }
                    this.initRemoteStructures();
                })

            },
            getRemoteVideoHtml(participantId) {
                return document.querySelector('#'+this.getRemoteVideoId(participantId));
            },
            createAndAddNewRemoteConnectionElement(participantId) {
                this.remoteConnectionData.push({
                    userId: participantId,
                    remoteVideo: this.getRemoteVideoHtml(participantId)
                });
            },
            initRemoteStructures() {
                console.log("Initializing remote videos");
                for (let pi of this.properParticipants) {
                    this.createAndAddNewRemoteConnectionElement(pi.id);
                }

                this.initDevices();
            },
            initDevices() {
                if (!navigator.mediaDevices) {
                    alert('There are no media devices');
                    this.gotLocalStream(null);
                    return
                }
                navigator.mediaDevices.getUserMedia({
                    audio: true,
                    video: true
                })
                    .then(this.gotLocalStream)
                    .catch((e) => {
                        alert('getUserMedia() error: ' + e.name);
                    });
            },
            gotLocalStream(stream) {
                if (stream) {
                    console.log('Adding local stream.');
                    this.localStream = stream;
                    this.localVideo.srcObject = stream;
                }

                bus.$emit(VIDEO_LOCAL_ESTABLISHED);
                bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, false))

                this.initConnections();
            },

            initializeRemoteConnectionElement(rcde) {
                console.log('>>>>>> creating peer connection, localstream=', this.localStream, "from me to", rcde.userId);
                const pc = this.createPeerConnection(rcde);

                if (this.localStream) {
                    pc.addStream(this.localStream);
                }
                rcde.peerConnection = pc;
            },
            initConnections(){
                if (!this.localStream) {
                    // TODO maybe retry
                    console.warn("localStream still not set -> we unable to initialize connections");
                } else {
                    console.log('Initializing connections from local stream', this.localStream);
                }
                // save this pc to array
                for (const rcde of this.remoteConnectionData) {
                    this.initializeRemoteConnectionElement(rcde);
                }
                this.sendMessage({type: EVENT_HELLO});
            },

            maybeStart(rcde){
                console.log('>>>>>> starting peer connection for', rcde.userId);

                this.stop(rcde);
                const pc = this.createPeerConnection(rcde);
                console.log('Created RTCPeerConnnection me -> user '+rcde.userId);
                if (this.localStream) {
                    // TODO maybe retry
                    pc.addStream(this.localStream);
                }
                rcde.peerConnection = pc;
                this.doOffer(rcde);
            },
            createPeerConnection(rcde) {
                const remoteVideo = rcde.remoteVideo;
                try {
                    const pc = new RTCPeerConnection(this.pcConfig);
                    pc.onicecandidate = this.fhandleIceCandidate(rcde);
                    if ("ontrack" in pc) {
                        pc.ontrack = this.fhandleRemoteTrackAdded(remoteVideo);
                    } else {
                        pc.onaddstream = this.fhandleRemoteStreamAdded(remoteVideo);
                    }
                    pc.onremovestream = this.handleRemoteStreamRemoved;
                    return pc;
                } catch (e) {
                    console.log('Failed to create PeerConnection, exception: ' + e.message);
                    alert('Cannot create RTCPeerConnection object.');
                }
            },

            doAnswer(pcde){
                console.log('Sending answer to peer ' + pcde.userId);
                const pc = pcde.peerConnection;
                pc.createAnswer().then(
                    this.fsetLocalDescriptionAndSendMessage(pcde),
                    this.fonCreateSessionDescriptionError(pcde)
                );
            },
            // ex doCall
            doOffer(pcde) {
                console.log('Sending offer to peer ' + pcde.userId);
                const pc = pcde.peerConnection;
                pc.createOffer(this.fsetLocalDescriptionAndSendMessage(pcde), this.fhandleCreateOfferError(pcde));
            },
            handleRemoteHangup(pcde) {
                console.log('Session terminated for ' + pcde.userId);
                this.stop(pcde);
            },
            fhandleRemoteStreamAdded(remoteVideo) {
                return (event) => {
                    console.log('Remote stream added.', event);
                    remoteVideo.srcObject = event.stream;
                }
            },
            fhandleRemoteTrackAdded(remoteVideo) {
                return (event) => {
                    console.log('Remote track added.', event);
                    remoteVideo.srcObject = event.streams[0];
                }
            },
            handleRemoteStreamRemoved (event) {
                console.log('Remote stream removed. Event: ', event);
            },
            stop(pcde) {
                if (pcde.peerConnection) {
                    console.log("Stopping peer connection to user " + pcde.userId);
                    pcde.peerConnection.close();
                } else {
                    console.log("Didn't stopped peer connection to user " + pcde.userId);
                }
                pcde.peerConnection = null;
            },
            hangupAll() {
                console.log('Hanging up.');
                for (const pcde of this.remoteConnectionData) {
                    this.stop(pcde);
                }
                this.sendMessage({type: EVENT_BYE});
            },
            sendMessage(message) {
                console.log('Client sending message: ', message);
                this.signalingSubscription.publish(setProperData(message));
            },
            fhandleIceCandidate(pcde) {
                const toUserId = pcde.userId;
                return (event) => {
                    console.log('icecandidate event: ', event);
                    if (event.candidate) {
                        this.sendMessage({
                            type: EVENT_CANDIDATE,
                            label: event.candidate.sdpMLineIndex,
                            id: event.candidate.sdpMid,
                            candidate: event.candidate.candidate,
                            toUserId: toUserId
                        });
                    } else {
                        console.log('End of candidates.', event);
                    }
                }
            },
            fhandleCreateOfferError(pcde) {
                return (event) => {
                    console.log('createOffer() error: ', event);
                    this.onUnknownErrorReset(pcde);
                }
            },

            fsetLocalDescriptionAndSendMessage(pcde) {
                return (sessionDescription) => {
                    console.log('setting setLocalDescription and sending it', sessionDescription);
                    const pc = pcde.peerConnection;
                    pc.setLocalDescription(sessionDescription);
                    const toUserId = pcde.userId;

                    const type = sessionDescription.type;
                    if (!type) {
                        console.error("Null type in setLocalAndSendMessage");
                        return
                    }
                    switch (type) {
                        case 'offer':
                            this.sendMessage({type: EVENT_OFFER, value: sessionDescription, toUserId: toUserId});
                            break;
                        case 'answer':
                            this.sendMessage({type: EVENT_ANSWER, value: sessionDescription, toUserId: toUserId});
                            break;
                        default:
                            console.error("Unknown type '"+type+"' in setLocalAndSendMessage");
                    }
                }
            },

            fonCreateSessionDescriptionError(pcde) {
                return (error) => {
                    console.error('Failed to create session description: ' + error.toString());
                    this.onUnknownErrorReset(pcde);
                }
            },

            onUnknownErrorReset(pcde) {
                console.log("Resetting state on error");
                this.localStream = null;

                pcde.peerConnection = null;

                console.log("Initializing devices again");
                this.initRemoteStructures();
            },

            isMyMessage (message) {
                return message.metadata && this.centrifugeSessionId == message.metadata.originatorClientId
            },
            shouldSkipNonMineMessage(message) {
                return message.toUserId != this.currentUser.id;
            },
            lookupPeerConnectionDataByUserId(userId) {
                console.log("Using remoteConnectionData", this.remoteConnectionData);
                for (const pcde of this.remoteConnectionData) {
                    if (pcde.userId == userId) {
                        return pcde;
                    }
                }
                return null;
            },
            lookupPeerConnectionData(message) {
                const originatorUserId = message.metadata.originatorUserId;
                return this.lookupPeerConnectionDataByUserId(originatorUserId);
            },

            stopStreamedVideo(videoElem) {
                if (!videoElem) {
                    console.warn("Didn't stopped html tracks because videoElem is null")
                    return
                }

                console.log("Stopping html tracks");
                const stream = videoElem.srcObject;
                if (!stream) {
                    console.warn("Didn't stopped html tracks because stream is null")
                    return
                }

                const tracks = stream.getTracks();

                tracks.forEach(function(track) {
                    track.stop();
                });

                videoElem.srcObject = null;
            },

            getLogin(participant) {
                return participant.login;
            },
            getAvatar(participant) {
                return participant.avatar;
            },
        },

        mounted() {
            this.localVideo = document.querySelector('#localVideo');

            /*
             * https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Connectivity
             * https://www.html5rocks.com/en/tutorials/webrtc/basics/
             * https://codelabs.developers.google.com/codelabs/webrtc-web/#4
             * WebRTC applications need to do several things:
              1.  Get streaming audio, video or other data.
              2.  Get network information such as IP addresses and ports, and exchange this with other WebRTC clients (known as peers) to enable connection, even through NATs and firewalls.
              3.  Coordinate signaling communication to report errors and initiate or close sessions.
              4.  Exchange information about media and client capability, such as resolution and codecs.
              5.  Communicate streaming audio, video or data.
             */

            this.signalingSubscription = this.centrifuge.subscribe("signaling"+this.chatId, (rawMessage) => {
                console.debug("Received raw message", rawMessage);
                // here we will process signaling messages
                if (this.isMyMessage(getData(rawMessage))) {
                    console.debug("Skipping my message", rawMessage);
                    return
                }
                const message = getProperData(rawMessage);

                console.log('Client received foreign presonal message:', message);

                const pcde = this.lookupPeerConnectionData(getData(rawMessage));

                if (!pcde){
                    console.warn("Cannot find remote connection data for ", rawMessage, " among ", this.remoteConnectionData)
                    return;
                }
                const pc = pcde.peerConnection;


                // handle broadcast messages
                if (message.type === EVENT_HELLO) {
                    this.maybeStart(pcde);
                    return;
                } else if (message.type === EVENT_BYE) {
                    this.handleRemoteHangup(pcde);
                    return;
                }


                // handle personal messages
                if (this.shouldSkipNonMineMessage(message)) {
                    console.debug("Skipping message not for me but for", message.toUserId);
                    return;
                }
                if (message.type === EVENT_OFFER && pc) {
                    pc.setRemoteDescription(new RTCSessionDescription(message.value));
                    this.doAnswer(pcde);
                } else if (message.type === EVENT_ANSWER && pc) {
                    console.debug("setting RemoteDescription");
                    pc.setRemoteDescription(new RTCSessionDescription(message.value));
                } else if (message.type === EVENT_CANDIDATE && pc) {
                    console.log("Handling remote ICE candidate for ", pcde.userId);
                    var candidate = new RTCIceCandidate({
                        sdpMLineIndex: message.label,
                        candidate: message.candidate
                    });
                    pc.addIceCandidate(candidate);
                }
            });

            this.getWebRtcConfiguration();
        },

        beforeDestroy() {
            console.log("Cleaning up");
            this.hangupAll();
            this.signalingSubscription.unsubscribe();
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, true));
        },

        watch: {
            'chatDto.participantIds': {
                handler: function (val, oldVal) {
                    const addedParticipantIds = val.filter(n => !oldVal.includes(n));
                    const deletedParticipantIds = oldVal.filter(n => !val.includes(n))
                    console.info("Added participantIds ", addedParticipantIds, " deleted participantIds ", deletedParticipantIds);

                    // close olds
                    for (const participantId of deletedParticipantIds) {
                        const rcde = this.lookupPeerConnectionDataByUserId(participantId);
                        if (!rcde) {
                            console.warn("Can't lookup peer connection data by userId ", participantId);
                            continue;
                        }
                        const html = this.getRemoteVideoHtml(rcde.userId);
                        console.log("Got remote video el", html);
                        this.stopStreamedVideo(html);

                        this.stop(rcde);

                        // remove it from array
                        const foundIndex = this.remoteConnectionData.findIndex(value => value.userId === rcde.userId);
                        if (foundIndex === -1) {
                            console.warn("Can't find index to remove from participantIds", rcde.userId);
                            return
                        }
                        this.remoteConnectionData.splice(foundIndex, 1);
                        console.info("Successfully removed PeerConnectionData for user", rcde.userId, this.remoteConnectionData);

                        // delete from page
                        html.parentElement.removeChild(html);
                    }
                    this.$forceUpdate();

                    // bypass reactive effect of rerender remote participants
                    Vue.nextTick(()=>{
                        // template already changed, so we need initialize news
                        for (const participantId of addedParticipantIds) {
                            this.createAndAddNewRemoteConnectionElement(participantId);
                            const rcde = this.lookupPeerConnectionDataByUserId(participantId);
                            if (!rcde) {
                                console.warn("Can't lookup peer connection data by userId ", participantId);
                                continue;
                            }
                            this.initializeRemoteConnectionElement(rcde);
                        }
                    });

                },
                deep: true
            },

        },
    }
</script>

<style scoped lang="stylus">
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