import React, {useEffect} from "react";
import Centrifuge from "centrifuge";

import {
    useParams
} from "react-router-dom";

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

        var chatSubscription = centrifuge.subscribe("chat1", function(message) {
            // we can rely only on data
            drawText(JSON.stringify(getProperData(message)));
        });
        var input = document.getElementById("input");
        input.addEventListener('keyup', function(e) {
            if (e.keyCode == 13) { // ENTER key pressed
                chatSubscription.publish({payload: {value: this.value}});
                input.value = '';
            }
        });
        // After setting event handlers – initiate actual connection with server.
        centrifuge.connect();

        var signalingSubscription = centrifuge.subscribe("signaling1", function(message) {
            // here we will process signaling messages
        });




        /* https://www.html5rocks.com/en/tutorials/webrtc/basics/
         * https://codelabs.developers.google.com/codelabs/webrtc-web/#4
         * WebRTC applications need to do several things:
            Get streaming audio, video or other data.
            Get network information such as IP addresses and ports, and exchange this with other WebRTC clients (known as peers) to enable connection, even through NATs and firewalls.
            Coordinate signaling communication to report errors and initiate or close sessions.
            Exchange information about media and client capability, such as resolution and codecs.
            Communicate streaming audio, video or data.
         */


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
            <video id="remoteVideo" autoPlay playsInline></video>
            <div>
                <button id="startButton">Start</button>
                <button id="callButton">Call</button>
                <button id="hangupButton">Hang Up</button>
            </div>
        </div>
    )
}

export default Chat;