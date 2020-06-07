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

        function getProperData(message) {
            return message.data
        }

        function setProperData(message) {
            return {
                payload: message
            }
        }

        var chatSubscription = centrifuge.subscribe("chat1", function(message) {
            // we can rely only on data
            drawText(JSON.stringify(getProperData(message)));
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

        const EVENT_CANDIDATE = 'candidate';

        function isMyMessage(message) {
            return message.metadata && clientId == message.metadata.originatorClientId
        }

        var signalingSubscription = centrifuge.subscribe("signaling1", function(rawMessage) {
            console.debug("Received raw message", rawMessage);
            // here we will process signaling messages
            const message = getProperData(rawMessage);
            if (isMyMessage(message)) {
                console.debug("Skipping my message", message);
                return
            }

            if (message.type === EVENT_CANDIDATE) {
                console.log('Reacting on candidate as addIceCandidate');
                var candidate = new RTCIceCandidate({
                    sdpMLineIndex: message.label,
                    candidate: message.candidate
                });
                //pc.addIceCandidate(candidate);
            }
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

        // 1. Get streaming audio, video or other data.
        const localVideo = document.getElementById('localVideo');
        const remoteVideo = document.getElementById('remoteVideo1');
        console.log("1. Getting video elements", localVideo, remoteVideo);
        let localStream;
        let remoteStream;
        let localPeerConnection;
        let remotePeerConnection;
        const mediaStreamConstraints = {
            video: true,
        };
        navigator.mediaDevices
            .getUserMedia(mediaStreamConstraints)
            .then(gotLocalMediaStream)
            .catch(handleLocalMediaStreamError)
            .then(establishConnection);

        // 2. Get network information such as IP addresses and ports, and exchange this with other WebRTC clients (known as peers) to enable connection, even through NATs and firewalls.
        function establishConnection() {
            console.log("2. Getting network information");
            const servers = null;  // Allows for RTC server configuration.
            localPeerConnection = new RTCPeerConnection(servers);
            console.debug('Created local peer connection object localPeerConnection.');

            localPeerConnection.addEventListener('icecandidate', handleConnection);
            localPeerConnection.addEventListener('iceconnectionstatechange', handleConnectionChange);

            remotePeerConnection = new RTCPeerConnection(servers);
            console.debug('Created remote peer connection object remotePeerConnection.');

            remotePeerConnection.addEventListener('icecandidate', handleConnection);
            remotePeerConnection.addEventListener('iceconnectionstatechange', handleConnectionChange);
            remotePeerConnection.addEventListener('addstream', gotRemoteMediaStream);

            // Add local stream to connection and create offer to connect.
            localPeerConnection.addStream(localStream);
            console.debug('Added local stream to localPeerConnection.');

            console.debug('localPeerConnection createOffer start.');
            const offerOptions = {
                offerToReceiveVideo: 1,
            };
            localPeerConnection.createOffer(offerOptions)
                .then(createdOffer).catch(setSessionDescriptionError);
        }



        // utility functions

        function gotLocalMediaStream(mediaStream) {
            localVideo.srcObject = mediaStream;
            localStream = mediaStream;
            console.debug('Received local stream.');
        }
        function handleLocalMediaStreamError(error) {
            console.error(`navigator.getUserMedia error: ${error.toString()}.`);
        }

        // Connects with new peer candidate.
        function handleConnection(event) {
            console.debug("Handle connection", event);
            const peerConnection = event.target;
            const iceCandidate = event.candidate;

            if (iceCandidate) {
                /*const newIceCandidate = new RTCIceCandidate(iceCandidate);
                const otherPeer = getOtherPeer(peerConnection);

                otherPeer.addIceCandidate(newIceCandidate)
                    .then(() => {
                        handleConnectionSuccess(peerConnection);
                    }).catch((error) => {
                    handleConnectionFailure(peerConnection, error);
                });*/
                sendMessage({
                    type: EVENT_CANDIDATE,
                    label: event.candidate.sdpMLineIndex,
                    id: event.candidate.sdpMid,
                    candidate: event.candidate.candidate
                });

                console.debug(`${getPeerName(peerConnection)} ICE candidate:\n` +
                    `${event.candidate.candidate}.`);
            }
        }

        function handleConnectionChange(event) {
            const peerConnection = event.target;
            console.log('ICE state change event: ', event);
            console.debug(`${getPeerName(peerConnection)} ICE state: ${peerConnection.iceConnectionState}.`);
        }

        function gotRemoteMediaStream(event) {
            const mediaStream = event.stream;
            remoteVideo.srcObject = mediaStream;
            remoteStream = mediaStream;
            console.debug('Remote peer connection received remote stream.');
        }

        // Logs offer creation and sets peer connection session descriptions.
        function createdOffer(description) {
            console.debug(`Offer from localPeerConnection:\n${description.sdp}`);

            console.debug('localPeerConnection setLocalDescription start.');
            localPeerConnection.setLocalDescription(description)
                .then(() => {
                    setLocalDescriptionSuccess(localPeerConnection);
                }).catch(setSessionDescriptionError);

            console.debug('remotePeerConnection setRemoteDescription start.');
            remotePeerConnection.setRemoteDescription(description)
                .then(() => {
                    setRemoteDescriptionSuccess(remotePeerConnection);
                }).catch(setSessionDescriptionError);

            console.debug('remotePeerConnection createAnswer start.');
            remotePeerConnection.createAnswer()
                .then(createdAnswer)
                .catch(setSessionDescriptionError);
        }
        // Logs error when setting session description fails.
        function setSessionDescriptionError(error) {
            console.debug(`Failed to create session description: ${error.toString()}.`);
        }

        // Logs success when setting session description.
        function setDescriptionSuccess(peerConnection, functionName) {
            const peerName = getPeerName(peerConnection);
            console.debug(`${peerName} ${functionName} complete.`);
        }

        // Logs success when localDescription is set.
        function setLocalDescriptionSuccess(peerConnection) {
            setDescriptionSuccess(peerConnection, 'setLocalDescription');
        }

        // Logs success when remoteDescription is set.
        function setRemoteDescriptionSuccess(peerConnection) {
            setDescriptionSuccess(peerConnection, 'setRemoteDescription');
        }

        // Gets the "other" peer connection.
        function getOtherPeer(peerConnection) {
            return (peerConnection === localPeerConnection) ?
                remotePeerConnection : localPeerConnection;
        }
        // Gets the name of a certain peer connection.
        function getPeerName(peerConnection) {
            return (peerConnection === localPeerConnection) ?
                'localPeerConnection' : 'remotePeerConnection';
        }

        // Logs that the connection failed.
        function handleConnectionFailure(peerConnection, error) {
            console.debug(`${getPeerName(peerConnection)} failed to add ICE Candidate:\n${error.toString()}.`);
        }

        // Logs that the connection succeeded.
        function handleConnectionSuccess(peerConnection) {
            console.debug(`${getPeerName(peerConnection)} addIceCandidate success.`);
        }

        // Logs answer to offer creation and sets peer connection session descriptions.
        function createdAnswer(description) {
            console.debug(`Answer from remotePeerConnection:\n${description.sdp}.`);

            console.debug('remotePeerConnection setLocalDescription start.');
            remotePeerConnection.setLocalDescription(description)
                .then(() => {
                    setLocalDescriptionSuccess(remotePeerConnection);
                }).catch(setSessionDescriptionError);

            console.debug('localPeerConnection setRemoteDescription start.');
            localPeerConnection.setRemoteDescription(description)
                .then(() => {
                    setRemoteDescriptionSuccess(localPeerConnection);
                }).catch(setSessionDescriptionError);
        }

        function sendMessage(message) {
            console.log('Client sending message: ', message);
            signalingSubscription.publish(setProperData(message));
        }


        // cleanup
        return function cleanup() {
            console.log("Cleaning up");
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
            <video id="remoteVideo1" autoPlay playsInline></video>
        </div>
    )
}

export default Chat;