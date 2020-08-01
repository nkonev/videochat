<template>
    <v-col cols="12" class="video-container">
        <video id="localVideo" autoPlay playsInline style="height: 220px"></video>
        <video v-for="(item, index) in getProperParticipantIds()" :key="item" :id="getRemoteVideoId(item)" autoPlay playsInline style="height: 220px"></video>
    </v-col>
</template>

<script>
    import {getData, getProperData} from "./centrifugeConnection";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import bus, { VIDEO_LOCAL_ESTABLISHED } from "./bus";


    const setProperData = (message) => {
        return {
            payload: message
        }
    };

    const EVENT_CANDIDATE = 'candidate';
    const EVENT_GOT_USER_MEDIA = 'got_user_media';
    const EVENT_BYE = 'bye';
    const EVENT_OFFER = 'offer';
    const EVENT_ANSWER = 'answer';

    var pcConfig = {
        'iceServers': [{
            'urls': 'stun:stun.l.google.com:19302'
        }]
    };


    export default {
        data() {
            return {
                signalingSubscription: null,

                isStarted: false, // really needed
                localStream: null,
                turnReady: null,
                localVideo: null,

                remoteConnectionData: [
                    // userId: number
                    // peerConnection: RTCPeerConnection
                    // remoteDescriptionSet: boolean
                    // remoteVideo: html element
                ]
            }
        },
        props: ['participantIds'],
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
                if (!this.currentUser) {
                    return [];
                }
                const ppi = this.participantIds.filter(pi => pi != this.currentUser.id);
                console.log("Participant ids except me:", ppi);
                return ppi;
            },
            initConnections() {
                console.log("Initializing remote videos");
                for (let pi of this.getProperParticipantIds()) {
                    this.remoteConnectionData.push({
                        userId: pi,
                        remoteVideo: document.querySelector('#'+this.getRemoteVideoId(pi))
                    });
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
                this.sendMessage({type: EVENT_GOT_USER_MEDIA});

                bus.$emit(VIDEO_LOCAL_ESTABLISHED);

                this.maybeStart();
            },


            maybeStart(){
                console.log('>>>>>>> maybeStart() ', this.isStarted, this.localStream);
                if (!this.isStarted && this.localStream) {
                    console.log('>>>>>> creating peer connection, localstream=', this.localStream);

                    // save this pc to array
                    for (const rcde of this.remoteConnectionData) {
                        const pc = this.createPeerConnection(rcde.remoteVideo);
                        pc.addStream(this.localStream);
                        rcde.peerConnection = pc;
                        this.doOffer(rcde);
                    }

                    this.isStarted = true;
                }
            },
            createPeerConnection(remoteVideo) {
                try {
                    const pc = new RTCPeerConnection(null);
                    pc.onicecandidate = this.handleIceCandidate;
                    pc.onaddstream = this.fhandleRemoteStreamAdded(remoteVideo);
                    pc.onremovestream = this.handleRemoteStreamRemoved;
                    console.log('Created RTCPeerConnnection');
                    return pc;
                } catch (e) {
                    console.log('Failed to create PeerConnection, exception: ' + e.message);
                    alert('Cannot create RTCPeerConnection object.');
                }
            },

            doAnswer(pcde){
                console.log('Sending answer to peer.');
                const pc = pcde.peerConnection;
                pc.createAnswer().then(
                    this.fsetLocalDescriptionAndSendMessage(pc),
                    this.fonCreateSessionDescriptionError(pcde)
                );
            },
            // ex doCall
            doOffer(pcde) {
                console.log('Sending offer to peer');
                const pc = pcde.peerConnection;
                pc.createOffer(this.fsetLocalDescriptionAndSendMessage(pc), this.fhandleCreateOfferError(pcde));
            },
            handleRemoteHangup (pcde) {
                console.log('Session terminated.');
                pcde.remoteDescriptionSet = false;
                this.stop(pcde);
            },
            fhandleRemoteStreamAdded(remoteVideo) {
                return (event) => {
                    console.log('Remote stream added.', event);
                    remoteVideo.srcObject = event.stream;
                }
            },
            handleRemoteStreamRemoved (event) {
                console.log('Remote stream removed. Event: ', event);
            },
            stop(pcde) {
                this.isStarted = false;
                if (pcde.peerConnection) {
                    pcde.peerConnection.close();
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
            requestTurn (turnURL) {
                var turnExists = false;
                for (var i in pcConfig.iceServers) {
                    if (pcConfig.iceServers[i].urls.substr(0, 5) === 'turn:') {
                        turnExists = true;
                        this.turnReady = true;
                        break;
                    }
                }
                if (!turnExists) {
                    console.log('Getting TURN server from ', turnURL);
                    // No TURN server. Get one from computeengineondemand.appspot.com:
                    var xhr = new XMLHttpRequest();
                    xhr.onreadystatechange = function() {
                        if (xhr.readyState === 4 && xhr.status === 200) {
                            var turnServer = JSON.parse(xhr.responseText);
                            console.log('Got TURN server: ', turnServer);
                            pcConfig.iceServers.push({
                                'urls': 'turn:' + turnServer.username + '@' + turnServer.turn,
                                'credential': turnServer.password
                            });
                            this.turnReady = true;
                        }
                    };
                    xhr.open('GET', turnURL, true);
                    xhr.send();
                }
            },
            handleIceCandidate (event) {
                console.log('icecandidate event: ', event);
                if (event.candidate) {
                    this.sendMessage({
                        type: EVENT_CANDIDATE,
                        label: event.candidate.sdpMLineIndex,
                        id: event.candidate.sdpMid,
                        candidate: event.candidate.candidate
                    });
                } else {
                    console.log('End of candidates.');
                }
            },
            fhandleCreateOfferError(pcde) {
                return (event) => {
                    console.log('createOffer() error: ', event);
                    this.onUnknownErrorReset(pcde);
                }
            },

            fsetLocalDescriptionAndSendMessage(pc) {
                return (sessionDescription) => {
                    console.log('setting setLocalDescription', sessionDescription);
                    pc.setLocalDescription(sessionDescription);
                    const type = sessionDescription.type;
                    if (!type) {
                        console.error("Null type in setLocalAndSendMessage");
                        return
                    }
                    switch (type) {
                        case 'offer':
                            console.log('setLocalAndSendMessage sending message', sessionDescription);
                            this.sendMessage({type: EVENT_OFFER, value: sessionDescription});
                            break;
                        case 'answer':
                            console.log('setLocalAndSendMessage sending message', sessionDescription);
                            this.sendMessage({type: EVENT_ANSWER, value: sessionDescription});
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
                this.isStarted = false;
                this.turnReady = false;
                this.localStream = null;

                pcde.remoteDescriptionSet = false;
                pcde.peerConnection = null;

                console.log("Initializing devices again");
                this.initConnections();
            },

            isMyMessage (message) {
                return message.metadata && this.centrifugeSessionId == message.metadata.originatorClientId
            },
            lookupPeerConnectionData(message) {
                const originatorUserId = message.metadata.originatorUserId;
                for (const pcde of this.remoteConnectionData) {
                    if (pcde.userId == originatorUserId) {
                        return pcde;
                    }
                }
                return null;
            }
        },

        mounted() {
            this.localVideo = document.querySelector('#localVideo');

            /* https://www.html5rocks.com/en/tutorials/webrtc/basics/
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


                console.log('Client received foreign message:', message);
                if (message.type === EVENT_GOT_USER_MEDIA) {
                    this.maybeStart();
                    return;
                }

                const pcde = this.lookupPeerConnectionData(getData(rawMessage));
                if (!pcde){
                    console.warn("Cannot find remot econnection data for ", rawMessage)
                    return;
                }
                const pc = pcde.peerConnection;
                if (message.type === EVENT_OFFER) {
                    if (!pcde.remoteDescriptionSet) { // TODO to array
                        pc.setRemoteDescription(new RTCSessionDescription(message.value));
                        pcde.remoteDescriptionSet = true;
                    }
                    this.doAnswer(pcde);
                } else if (message.type === EVENT_ANSWER && this.isStarted) {
                    if (!pcde.remoteDescriptionSet) {
                        pc.setRemoteDescription(new RTCSessionDescription(message.value));
                        pcde.remoteDescriptionSet = true;
                    }
                } else if (message.type === EVENT_CANDIDATE && this.isStarted) {
                    var candidate = new RTCIceCandidate({
                        sdpMLineIndex: message.label,
                        candidate: message.candidate
                    });
                    pc.addIceCandidate(candidate);
                } else if (message.type === EVENT_BYE && this.isStarted) {
                    this.handleRemoteHangup(pcde);
                }
            });

            if (location.hostname !== 'localhost') {
                this.requestTurn('https://computeengineondemand.appspot.com/turn?username=41784574&key=4080218913');
            }

            this.initConnections();
        },

        beforeDestroy() {
            console.log("Cleaning up");
            this.hangupAll();
            this.signalingSubscription.unsubscribe();
        }
    }
</script>