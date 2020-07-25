<template>
    <v-card>
        <v-row dense>
            <v-col cols="12">
                <video id="localVideo" autoPlay playsInline style="height: 220px"></video>
                <video id="remoteVideo" autoPlay playsInline style="height: 220px"></video>
            </v-col>
            <v-col cols="12">
                <div id="messagesScroller" :style="scrollerHeight()">
                    <v-card-text>
                        <v-list>
                            <template v-for="(item, index) in items">
                            <v-list-item
                                    :key="item.id"
                                    dense
                            >
                                <v-list-item-avatar v-if="item.owner && item.owner.avatar">
                                    <v-img :src="item.owner.avatar"></v-img>
                                </v-list-item-avatar>
                                <v-list-item-content>
                                    <v-list-item-subtitle>{{getSubtitle(item)}}</v-list-item-subtitle>
                                    {{item.text}}
                                </v-list-item-content>
                                <v-list-item-action>
                                    <v-btn v-if="item.canEdit" text color="error" @click="deleteMessage(item)"><v-icon dark small>mdi-delete</v-icon></v-btn>
                                    <v-btn v-if="item.canEdit" text color="primary" @click="editMessage(item)"><v-icon dark small>mdi-lead-pencil</v-icon></v-btn>
                                </v-list-item-action>
                            </v-list-item>
                            <v-divider></v-divider>
                            </template>
                        </v-list>
                        <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId" direction="top">
                            <template slot="no-more"><span/></template>
                            <template slot="no-results"><span/></template>
                        </infinite-loading>
                    </v-card-text>
                    </div>
            </v-col>
        </v-row>
        <v-container id="sendButtonContainer">
            <v-row no-gutters dense>
                <v-col cols="12">
                    <v-text-field dense label="Send a message" @keyup.native.enter="sendMessageToChat" v-model="editMessageDto.text" :append-outer-icon="'mdi-send'" @click:append-outer="sendMessageToChat"></v-text-field>
                </v-col>
            </v-row>
        </v-container>
    </v-card>
</template>

<script>
    import axios from "axios";
    import infinityListMixin, {
        findIndex,
        pageSize, replaceInArray
    } from "./InfinityListMixin";
    import {getData, getProperData} from './centrifugeConnection'
    import Vue from 'vue'
    import bus, {CHANGE_TITLE, CHAT_EDITED, MESSAGE_ADD, MESSAGE_DELETED, MESSAGE_EDITED} from "./bus";
    import {titleFactory} from "./changeTitle";

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

    const dtoFactory = ()=>{
        return {
            id: null,
            text: "",
        }
    };


    export default {
        mixins:[infinityListMixin()],
        computed: {
            chatId() {
                return this.$route.params.id
            },
            pageHeight () {
                return document.body.scrollHeight
            },
        },
        data() {
            return {
                chatMessagesSubscription: null,
                signalingSubscription: null,

                pc: null, // peer connection
                isStarted: false, // реально нужен
                localStream: null,
                remoteStream: null,
                turnReady: null,
                remoteDescriptionSet: false,

                localVideo: null,
                remoteVideo: null,

                editMessageDto: dtoFactory(),
            }
        },
        methods: {
            addItem(dto) {
                console.log("Adding item", dto);
                this.items.push(dto);
                this.$forceUpdate();
            },
            changeItem(dto) {
                console.log("Replacing item", dto);
                replaceInArray(this.items, dto);
                this.$forceUpdate();
            },
            removeItem(dto) {
                console.log("Removing item", dto);
                const idxToRemove = findIndex(this.items, dto);
                this.items.splice(idxToRemove, 1);
                this.$forceUpdate();
            },

            deleteMessage(dto){
                axios.delete(`/api/chat/${this.chatId}/message/${dto.id}`)
            },
            editMessage(dto){
                this.editMessageDto = {id: dto.id, text: dto.text};
            },

            scrollerHeight() {
                const maybeScroller = document.getElementById("messagesScroller");
                const maybeSendButton = document.getElementById("sendButtonContainer");

                if (maybeScroller && maybeSendButton) {
                    const topOfScroller = maybeScroller.getBoundingClientRect().top;
                    const sendButtonContainerHeight = maybeSendButton.getBoundingClientRect().height;
                    const availableHeight = window.innerHeight;
                    const newHeight = availableHeight - topOfScroller - sendButtonContainerHeight - 16;
                    if (newHeight > 0) {
                        return `overflow-y: auto; height: ${newHeight}px`
                    }
                }
                return 'overflow-y: auto; height: 240px'
            },
            infiniteHandler($state) {
                axios.get(`/api/chat/${this.chatId}/message`, {
                    params: {
                        page: this.page,
                        size: pageSize,
                        reverse: true
                    },
                }).then(({ data }) => {
                    const list = data;
                    if (list.length) {
                        this.page += 1;
                        // this.items = [...this.items, ...list];
                        this.items.unshift(...list.reverse());
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                });
            },
            getSubtitle(item) {
                return `${item.owner.login} at ${item.createDateTime}`
            },

            sendMessageToChat() {
                if (this.editMessageDto.text && this.editMessageDto.text !== "") {
                    (this.editMessageDto.id ? axios.put(`/api/chat/`+this.chatId+'/message', this.editMessageDto) : axios.post(`/api/chat/`+this.chatId+'/message', this.editMessageDto)).then(response => {
                            console.log("Resetting text input");
                            this.editMessageDto.text = "";
                            this.editMessageDto.id = null;
                        })
                }
            },
            isMyMessage (message) {
                return message.metadata && this.centrifugeSessionId == message.metadata.originatorClientId
            },
            onNewMessage(dto) {
                if (dto.chatId == this.chatId) {
                    this.addItem(dto);
                    this.scrollDown();
                } else {
                    console.log("Skipping", dto)
                }
            },
            onDeleteMessage(dto) {
                if (dto.chatId == this.chatId) {
                    this.removeItem(dto);
                } else {
                    console.log("Skipping", dto)
                }
            },
            onEditMessage(dto) {
                if (dto.chatId == this.chatId) {
                    this.changeItem(dto);
                } else {
                    console.log("Skipping", dto)
                }
            },
            scrollDown() {
                Vue.nextTick(()=>{
                    var myDiv = document.getElementById("messagesScroller");
                    console.log("myDiv.scrollTop", myDiv.scrollTop, "myDiv.scrollHeight", myDiv.scrollHeight);
                    myDiv.scrollTop = myDiv.scrollHeight;
                });
            },
            getInfo() {
                axios.get(`/api/chat/${this.chatId}`).then(({ data }) => {
                    console.log("Got info about chat", data);
                    bus.$emit(CHANGE_TITLE, titleFactory(data.name, false, data.canEdit, data.canEdit ? this.chatId: null));
                });
            },
            onChatChange(dto) {
                if (dto.id == this.chatId) {
                    this.getInfo();
                }
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
                if (this.pc) {
                    this.pc.close();
                }
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
                if (!navigator.mediaDevices) {
                    console.log('There are no media devices');
                    return
                }
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

            this.chatMessagesSubscription = this.centrifuge.subscribe("chatMessages"+this.chatId, (message) => {
                // this.items = [...this.items, JSON.stringify(getData(message))];

                // actually it's global notification, so we just log it
                const data = getData(message);
                console.log("Got global notification", data)
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


            bus.$emit(CHANGE_TITLE, titleFactory(`Chat #${this.chatId}`, false, true));

            this.getInfo();
            bus.$on(MESSAGE_ADD, this.onNewMessage);
            bus.$on(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$on(CHAT_EDITED, this.onChatChange);
            bus.$on(MESSAGE_EDITED, this.onEditMessage);
        },
        beforeDestroy() {
            console.log("Cleaning up");
            this.hangup();
            this.chatMessagesSubscription.unsubscribe();
            this.signalingSubscription.unsubscribe();

            bus.$off(MESSAGE_ADD, this.onNewMessage);
            bus.$off(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$off(CHAT_EDITED, this.onChatChange);
            bus.$off(MESSAGE_EDITED, this.onEditMessage);
        }
    }
</script>