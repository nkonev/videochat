import React, {useEffect} from "react";
import Centrifuge from "centrifuge";
import { useParams } from "react-router-dom";

function Chat() {
    let { id } = useParams();

    useEffect(() => {
        console.log("Invoked centrifuge effect");

        // Create Centrifuge object with Websocket endpoint address set in main.go
        var url = ((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/api/chat/websocket";
        var clientId;
        var centrifuge = new Centrifuge(url, {
            onRefresh: function(ctx){
                console.debug("Dummy refresh");
            }
        });
        function drawText(text) {
            var div = document.createElement('div');
            div.innerHTML = text + '<br>';
            document.body.appendChild(div);
        }
        centrifuge.on('connect', function(ctx){
            drawText('Connected over ' + ctx.transport);
            console.log("Connected response", ctx);
            clientId = ctx.client;
            console.log('My clientId :', clientId);
        });
        centrifuge.on('disconnect', function(ctx){
            drawText('Disconnected: ' + ctx.reason);
        });

        function getData(message) {
            return message.data
        }

        function getProperData(message) {
            return message.data.payload
        }

        function setProperData(message) {
            return {
                payload: message
            }
        }

        var chatSubscription = centrifuge.subscribe("chat1", function(message) {
            // we can rely only on data
            drawText(JSON.stringify(getData(message)));
        });
        var input = document.getElementById("input");
        input.addEventListener('keyup', function(e) {
            if (e.keyCode == 13) { // ENTER key pressed
                chatSubscription.publish(setProperData({value: this.value}));
                input.value = '';
            }
        });
        // After setting event handlers – initiate actual connection with server.
        centrifuge.connect();


        function isMyMessage(message) {
            return message.metadata && clientId == message.metadata.originatorClientId
        }



        /* https://www.html5rocks.com/en/tutorials/webrtc/basics/
         * https://codelabs.developers.google.com/codelabs/webrtc-web/#4
         * WebRTC applications need to do several things:
          1.  Get streaming audio, video or other data.
          2.  Get network information such as IP addresses and ports, and exchange this with other WebRTC clients (known as peers) to enable connection, even through NATs and firewalls.
          3.  Coordinate signaling communication to report errors and initiate or close sessions.
          4.  Exchange information about media and client capability, such as resolution and codecs.
          5.  Communicate streaming audio, video or data.
         */
        const EVENT_CANDIDATE = 'candidate';
        const EVENT_GOT_USER_MEDIA = 'got_user_media';
        const EVENT_BYE = 'bye';
        const EVENT_OFFER = 'offer';
        const EVENT_ANSWER = 'answer';

        var signalingSubscription = centrifuge.subscribe("signaling1", function(rawMessage) {
            console.debug("Received raw message", rawMessage);
            // here we will process signaling messages
            if (isMyMessage(getData(rawMessage))) {
                console.debug("Skipping my message", rawMessage);
                return
            }
            const message = getProperData(rawMessage);

            console.log('Client received foreign message:', message);
            if (message.type === EVENT_GOT_USER_MEDIA) {
                maybeStart();
            }
            else if (message.type === EVENT_OFFER) {
                if (!remoteDescriptionSet && pc) { // checking pc - prevent NPE
                    pc.setRemoteDescription(new RTCSessionDescription(message.value));
                    remoteDescriptionSet = true;
                }
                doAnswer();
            } else if (message.type === EVENT_ANSWER && isStarted) {
                if (!remoteDescriptionSet && pc) { // checking pc - prevent NPE
                    pc.setRemoteDescription(new RTCSessionDescription(message.value));
                    remoteDescriptionSet = true;
                }
            }
            else if (message.type === EVENT_CANDIDATE && isStarted) {
                var candidate = new RTCIceCandidate({
                    sdpMLineIndex: message.label,
                    candidate: message.candidate
                });
                pc.addIceCandidate(candidate);
            } else if (message.type === EVENT_BYE && isStarted) {
                handleRemoteHangup();
            }
        });


        var isStarted = false; // реально нужен
        var localStream;
        var pc;
        var remoteStream;
        var turnReady;
        let remoteDescriptionSet = false;

        var pcConfig = {
            'iceServers': [{
                'urls': 'stun:stun.l.google.com:19302'
            }]
        };

////////////////////////////////////////////////

        function sendMessage(message) {
            console.log('Client sending message: ', message);
            signalingSubscription.publish(setProperData(message));
        }

////////////////////////////////////////////////////

        var localVideo = document.querySelector('#localVideo');
        var remoteVideo = document.querySelector('#remoteVideo');

        function initDevices() {
            navigator.mediaDevices.getUserMedia({
                audio: false,
                video: true
            })
                .then(gotStream)
                .catch(function(e) {
                    alert('getUserMedia() error: ' + e.name);
                });
        }

        initDevices();

        function gotStream(stream) {
            console.log('Adding local stream.');
            localStream = stream;
            localVideo.srcObject = stream;
            sendMessage({type: EVENT_GOT_USER_MEDIA});

            maybeStart();
        }

        var constraints = {
            video: true
        };

        console.log('Getting user media with constraints', constraints);

        if (location.hostname !== 'localhost') {
            requestTurn(
                'https://computeengineondemand.appspot.com/turn?username=41784574&key=4080218913'
            );
        }

        function maybeStart() {
            console.log('>>>>>>> maybeStart() ', isStarted, localStream);
            if (!isStarted && localStream) {
                console.log('>>>>>> creating peer connection');
                createPeerConnection();
                pc.addStream(localStream);
                isStarted = true;
                doOffer();
            }
        }


/////////////////////////////////////////////////////////

        function createPeerConnection() {
            try {
                pc = new RTCPeerConnection(null);
                pc.onicecandidate = handleIceCandidate;
                pc.onaddstream = handleRemoteStreamAdded;
                pc.onremovestream = handleRemoteStreamRemoved;
                console.log('Created RTCPeerConnnection');
            } catch (e) {
                console.log('Failed to create PeerConnection, exception: ' + e.message);
                alert('Cannot create RTCPeerConnection object.');
                return;
            }
        }

        function handleIceCandidate(event) {
            console.log('icecandidate event: ', event);
            if (event.candidate) {
                sendMessage({
                    type: EVENT_CANDIDATE,
                    label: event.candidate.sdpMLineIndex,
                    id: event.candidate.sdpMid,
                    candidate: event.candidate.candidate
                });
            } else {
                console.log('End of candidates.');
            }
        }

        function handleCreateOfferError(event) {
            console.log('createOffer() error: ', event);
            onUnknownErrorReset();
        }

        // ex doCall
        function doOffer() {
            console.log('Sending offer to peer');
            pc.createOffer(setLocalAndSendMessage, handleCreateOfferError);
        }

        function doAnswer() {
            console.log('Sending answer to peer.');
            pc.createAnswer().then(
                setLocalAndSendMessage,
                onCreateSessionDescriptionError
            );
        }

        function setLocalAndSendMessage(sessionDescription) {
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
                    sendMessage({type: EVENT_OFFER, value: sessionDescription});
                    break;
                case 'answer':
                    console.log('setLocalAndSendMessage sending message', sessionDescription);
                    sendMessage({type: EVENT_ANSWER, value: sessionDescription});
                    break;
                default:
                    console.error("Unknown type '"+type+"' in setLocalAndSendMessage");
            }
        }

        function onCreateSessionDescriptionError(error) {
            console.error('Failed to create session description: ' + error.toString());
            onUnknownErrorReset();
        }

        function onUnknownErrorReset() {
            console.log("Resetting state on error");
            isStarted = false;
            remoteDescriptionSet = false;
            localStream = null;
            pc = null;
            remoteStream = null;
            turnReady = false;

            console.log("Initializing devices again");
            initDevices();
        }

        function requestTurn(turnURL) {
            var turnExists = false;
            for (var i in pcConfig.iceServers) {
                if (pcConfig.iceServers[i].urls.substr(0, 5) === 'turn:') {
                    turnExists = true;
                    turnReady = true;
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
                        turnReady = true;
                    }
                };
                xhr.open('GET', turnURL, true);
                xhr.send();
            }
        }

        function handleRemoteStreamAdded(event) {
            console.log('Remote stream added.');
            remoteStream = event.stream;
            remoteVideo.srcObject = remoteStream;
        }

        function handleRemoteStreamRemoved(event) {
            console.log('Remote stream removed. Event: ', event);
        }

        function hangup() {
            console.log('Hanging up.');
            stop();
            sendMessage({type: EVENT_BYE});
        }

        function handleRemoteHangup() {
            console.log('Session terminated.');
            remoteDescriptionSet = false;
            stop();
        }

        function stop() {
            isStarted = false;
            pc.close();
            pc = null;
        }


        // cleanup
        return function cleanup() {
            console.log("Cleaning up");
            hangup();
            chatSubscription.unsubscribe();
            signalingSubscription.unsubscribe();
            centrifuge.disconnect();
        };
    });


    return (
        <div>
            <div>Viewing chat # {id}</div>
            <input type="text" id="input" />


            <video id="localVideo" autoPlay playsInline></video>
            <video id="remoteVideo" autoPlay playsInline></video>
        </div>
    )
}

export default Chat;