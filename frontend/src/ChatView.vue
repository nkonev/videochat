<template>
    <v-card>
        <v-card-title>It's a chat #{{chatId}}</v-card-title>
        <div>
            <video id="localVideo" autoPlay playsInline></video>
            <video id="remoteVideo" autoPlay playsInline></video>
        </div>

        <v-card-text>
            <v-list>
                <v-list-item
                        v-for="(item, index) in items"
                        :key="item.id"
                >
                    <v-list-item-content>
                        <v-list-item-title v-html="item.text"></v-list-item-title>
                        <v-list-item-subtitle v-html="item.createDateTime"></v-list-item-subtitle>
                    </v-list-item-content>
                </v-list-item>
            </v-list>
            <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId"></infinite-loading>

            <v-text-field label="Send a message" @keyup.native.enter="sendMessageToChat" v-model="chatMessageText"></v-text-field>
        </v-card-text>

    </v-card>
</template>

<script>
    import Centrifuge from "centrifuge";
    import axios from "axios";
    import infinityListMixin from "./InfinityListMixin";

    const drawText = (text) => {
        var div = document.createElement('div');
        div.innerHTML = text + '<br>';
        document.body.appendChild(div);
    };

    const getData = (message) => {
        return message.data
    };

    const getProperData = (message) => {
        return message.data.payload
    };

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

    let chatUrl;
    const chatUrlFunction = () => {
        return chatUrl
    };
    export default {
        mixins:[infinityListMixin(chatUrlFunction)],
        computed: {
            chatId() {
                return this.$route.params.id
            },
        },
        data() {
            return {
                centrifuge: null,
                clientId: null, // centrifuge session id

                chatSubscription: null,
                signalingSubscription: null,

                pc: null, // peer connection
                isStarted: false, // реально нужен
                localStream: null,
                remoteStream: null,
                turnReady: null,
                remoteDescriptionSet: false,

                localVideo: null,
                remoteVideo: null,

                chatMessageText: "",
            }
        },
        methods: {
            setChatUrl() {
                chatUrl = `/api/chat/`+this.chatId +'/message';
            },
            sendMessageToChat() {
                if (this.chatMessageText && this.chatMessageText !== "") {
                    console.log("Sending", this.chatMessageText);
                    const t = this.chatMessageText;
                    const dtoToPost = {
                        text: t
                    };
                    axios.post(`/api/chat/`+this.chatId+'/message', dtoToPost);
                    this.chatMessageText = "";
                }
            },
            setupCentrifuge () {
                // Create Centrifuge object with Websocket endpoint address set in main.go
                var url = ((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/api/chat/websocket";
                var centrifuge = new Centrifuge(url, {
                    onRefresh: (ctx)=>{
                        console.debug("Dummy refresh");
                    }
                });
                centrifuge.on('connect', (ctx)=>{
                    console.log("Connected response", ctx);
                    this.clientId = ctx.client;
                    console.log('My clientId :', this.clientId);
                });
                centrifuge.on('disconnect', (ctx)=>{
                    console.log("Disconnected response", ctx);
                });
                centrifuge.connect();
                return centrifuge;
            },
            isMyMessage (message) {
                return message.metadata && this.clientId == message.metadata.originatorClientId
            },
            maybeStart(){
                console.log('>>>>>>> maybeStart() ', this.isStarted, this.localStream);
                if (!this.isStarted && this.localStream) {
                    console.log('>>>>>> creating peer connection');
                    this.createPeerConnection();
                    this.pc.addStream(this.localStream);
                    this.isStarted = true;
                    this.doOffer();
                }
            },
            doAnswer (){
                console.log('Sending answer to peer.');
                this.pc.createAnswer().then(
                    this.setLocalAndSendMessage,
                    this.onCreateSessionDescriptionError
                );
            },
            createPeerConnection () {
                try {
                    this.pc = new RTCPeerConnection(null);
                    this.pc.onicecandidate = this.handleIceCandidate;
                    this.pc.onaddstream = this.handleRemoteStreamAdded;
                    this.pc.onremovestream = this.handleRemoteStreamRemoved;
                    console.log('Created RTCPeerConnnection');
                } catch (e) {
                    console.log('Failed to create PeerConnection, exception: ' + e.message);
                    alert('Cannot create RTCPeerConnection object.');
                    return;
                }
            },
            // ex doCall
            doOffer() {
                console.log('Sending offer to peer');
                this.pc.createOffer(this.setLocalAndSendMessage, this.handleCreateOfferError);
            },
            handleRemoteHangup () {
                console.log('Session terminated.');
                this.remoteDescriptionSet = false;
                this.stop();
            },
            handleRemoteStreamAdded (event){
                console.log('Remote stream added.');
                this.remoteStream = event.stream;
                this.remoteVideo.srcObject = this.remoteStream;
            },
            handleRemoteStreamRemoved (event) {
                console.log('Remote stream removed. Event: ', event);
            },
            stop () {
                this.isStarted = false;
                this.pc.close();
                this.pc = null;
            },
            hangup() {
                console.log('Hanging up.');
                this.stop();
                this.sendMessage({type: EVENT_BYE});
            },
            sendMessage(message) {
                console.log('Client sending message: ', message);
                this.signalingSubscription.publish(setProperData(message));
            },
            initDevices() {
                navigator.mediaDevices.getUserMedia({
                    audio: false,
                    video: true
                })
                    .then(this.gotStream)
                    .catch((e) => {
                        alert('getUserMedia() error: ' + e.name);
                    });
            },
            gotStream (stream) {
                console.log('Adding local stream.');
                this.localStream = stream;
                this.localVideo.srcObject = stream;
                this.sendMessage({type: EVENT_GOT_USER_MEDIA});

                this.maybeStart();
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
            handleCreateOfferError (event) {
                console.log('createOffer() error: ', event);
                this.onUnknownErrorReset();
            },

            setLocalAndSendMessage (sessionDescription){
                console.log('setting setLocalDescription', sessionDescription);
                this.pc.setLocalDescription(sessionDescription);
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
            },

            onCreateSessionDescriptionError (error) {
                console.error('Failed to create session description: ' + error.toString());
                this.onUnknownErrorReset();
            },

            onUnknownErrorReset () {
                console.log("Resetting state on error");
                this.isStarted = false;
                this.remoteDescriptionSet = false;
                this.localStream = null;
                this.pc = null;
                this.remoteStream = null;
                this.turnReady = false;

                console.log("Initializing devices again");
                this.initDevices();
            },

        },
        mounted() {
            this.localVideo = document.querySelector('#localVideo');
            this.remoteVideo = document.querySelector('#remoteVideo');

            this.centrifuge = this.setupCentrifuge();


            this.chatSubscription = this.centrifuge.subscribe("chat"+this.chatId, (message) => {
                // we can rely only on data
                // this.items = [...this.items, JSON.stringify(getData(message))];
                this.rerenderItem(getData(message));
            });

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
                } else if (message.type === EVENT_OFFER) {
                    if (this.pc) {
                        if (!this.remoteDescriptionSet) { // checking pc - prevent NPE
                            this.pc.setRemoteDescription(new RTCSessionDescription(message.value));
                            this.remoteDescriptionSet = true;
                        }
                        this.doAnswer();
                    } else {
                        console.warn("Peer connection still not set so I cannot answer on offer");
                    }
                } else if (message.type === EVENT_ANSWER && this.isStarted) {
                    if (!this.remoteDescriptionSet && this.pc) { // checking pc - prevent NPE
                        this.pc.setRemoteDescription(new RTCSessionDescription(message.value));
                        this.remoteDescriptionSet = true;
                    }
                }
                else if (message.type === EVENT_CANDIDATE && this.isStarted) {
                    var candidate = new RTCIceCandidate({
                        sdpMLineIndex: message.label,
                        candidate: message.candidate
                    });
                    this.pc.addIceCandidate(candidate);
                } else if (message.type === EVENT_BYE && this.isStarted) {
                    this.handleRemoteHangup();
                }
            });


            this.initDevices();


            if (location.hostname !== 'localhost') {
                this.requestTurn('https://computeengineondemand.appspot.com/turn?username=41784574&key=4080218913');
            }


            this.setChatUrl();
/////////////////////////////////////////////////////////

        },
        beforeDestroy() {
            console.log("Cleaning up");
            this.hangup();
            this.chatSubscription.unsubscribe();
            this.signalingSubscription.unsubscribe();

            this.centrifuge.disconnect();
        }
    }
</script>