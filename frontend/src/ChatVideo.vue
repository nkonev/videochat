<template>
    <v-col cols="12" class="video-container">
        <video id="localVideo" autoPlay playsInline style="height: 220px"></video>
        <video v-for="(item, index) in getProperParticipantIds()" :key="item" :id="getRemoteVideoId(item)" autoPlay playsInline style="height: 220px"></video>
    </v-col>
</template>

<script>
    import {getData, getProperData, setProperData} from "./centrifugeConnection";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import bus, {CHANGE_PHONE_BUTTON, CHAT_PARTICIPANTS_CHANGED, VIDEO_LOCAL_ESTABLISHED} from "./bus";
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
                signalingSubscription: null,

                pcConfig: null,

                localStream: null,
                localVideo: null,

                remoteConnectionData: [
                    // userId: number
                    // peerConnection: RTCPeerConnection
                    // remoteVideo: html element
                ]
            }
        },
        props: ['chatDto'],
        computed: {
            chatId() {
                return this.$route.params.id
            },
            ...mapGetters({currentUser: GET_USER})
        },
        methods: {
            getRemoteVideoId(participantId) {
                return 'remoteVideo'+participantId;
            },
            getProperParticipantIds() {
                const ppi = this.chatDto.participantIds.filter(pi => pi != this.currentUser.id);
                console.log("Participant ids except me:", ppi);
                return ppi;
            },
            onChangeParticipants(addedParticipantIds, deletedParticipantIds) {
                /*console.log("Added participantIds ", addedParticipantIds, " deleted participantIds ", deletedParticipantIds);

                // bypass reactive effect of rerender remote participants
                Vue.nextTick(()=>{
                    // template already changed, so we need initialize news
                    for (const participantId of addedParticipantIds) {
                        this.createAndAddNewRemoteConnectionElement(participantId);
                        const rcde = this.lookupPeerConnectionDataByUserId(participantId);
                        if (!rcde) {
                            console.warn("Can't lookupPeerConnectionDataByUserId ", participantId);
                            continue;
                        }
                        this.initializeRemoteConnectionElement(rcde);
                    }

                    // close olds
                    for (const participantId of deletedParticipantIds) {
                        const rcde = this.lookupPeerConnectionDataByUserId(participantId);
                        if (!rcde) {
                            console.warn("Can't lookupPeerConnectionDataByUserId ", participantId);
                            continue;
                        }
                        this.stop(rcde);

                        // delete from page
                        const html = this.getRemoteVideoHtml(participantId);
                        console.log("Got remote video el", html);
                        html.parentElement.removeChild(html);
                    }
                    this.$forceUpdate();
                });*/
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
                for (let pi of this.getProperParticipantIds()) {
                    this.createAndAddNewRemoteConnectionElement(pi);
                }

                this.initDevices();
            },
            initDevices() {
                if (!navigator.mediaDevices) {
                    console.log('There are no media devices');
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
                console.log('Adding local stream.');
                this.localStream = stream;
                this.localVideo.srcObject = stream;

                bus.$emit(VIDEO_LOCAL_ESTABLISHED);
                bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, false))

                this.initConnections();
            },

            initializeRemoteConnectionElement(rcde) {
                console.log('>>>>>> creating peer connection, localstream=', this.localStream, "from me to", rcde.userId);
                const pc = this.createPeerConnection(rcde);
                pc.addStream(this.localStream);
                rcde.peerConnection = pc;
            },
            initConnections(){
                console.log('Initializing connections from local stream', this.localStream);
                if (this.localStream) {
                    // save this pc to array
                    for (const rcde of this.remoteConnectionData) {
                        this.initializeRemoteConnectionElement(rcde);
                    }
                    this.sendMessage({type: EVENT_HELLO});
                } else {
                    // TODO maybe retry
                    console.warn("localStream still not set -> we unable to initialize connections");
                }
            },

            maybeStart(rcde){
                if (this.localStream) {
                    console.log('>>>>>> starting peer connection for', rcde.userId);

                    this.stop(rcde);
                    const pc = this.createPeerConnection(rcde);
                    console.log('Created RTCPeerConnnection me -> user '+rcde.userId);
                    pc.addStream(this.localStream);
                    rcde.peerConnection = pc;
                    this.doOffer(rcde);
                } else {
                    // TODO maybe retry
                    console.warn("localStream still not set  -> we unable to send offer");
                }
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

                    // if ("ontrack" in pcde.peerConnection) {
                    //     pcde.peerConnection.ontrack = null;
                    // } else {
                    //     pcde.peerConnection.onaddstream = null;
                    // }
                    // pcde.peerConnection.onremovestream = null;
                    // this.stopStreamedVideo(pcde.remoteVideo);

                    pcde.peerConnection.close();
                } else {
                    console.log("Didn't stopped peer connection to user " + pcde.userId);
                }
                pcde.peerConnection = null;
            },
            // stopAndRemove(pcde) {
            //     this.stop(pcde);
            //
            //     const foundIndex = this.chatDto.participantIds.findIndex(value => value === pcde.userId);
            //     if (foundIndex == -1) {
            //         console.warn("Can't find index to remove from participantIds", pcde.userId);
            //         return
            //     }
            //     this.chatDto.participantIds.splice(foundIndex, 1);
            // },
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
                        console.log('End of candidates.');
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
            // TODO check in chrome
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
            }
        },

        mounted() {
            bus.$on(CHAT_PARTICIPANTS_CHANGED, this.onChangeParticipants);

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
            bus.$off(CHAT_PARTICIPANTS_CHANGED, this.onChangeParticipants);
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
            }
        },
    }
</script>