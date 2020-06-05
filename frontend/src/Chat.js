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

        var sub = centrifuge.subscribe("chat", function(message) {
            drawText(JSON.stringify(message));
        });
        var input = document.getElementById("input");
        input.addEventListener('keyup', function(e) {
            if (e.keyCode == 13) { // ENTER key pressed
                sub.publish(this.value);
                input.value = '';
            }
        });
        // After setting event handlers – initiate actual connection with server.
        centrifuge.connect();


        /* TODO https://www.html5rocks.com/en/tutorials/webrtc/basics/
         * WebRTC applications need to do several things:
            Get streaming audio, video or other data.
            Get network information such as IP addresses and ports, and exchange this with other WebRTC clients (known as peers) to enable connection, even through NATs and firewalls.
            Coordinate signaling communication to report errors and initiate or close sessions.
            Exchange information about media and client capability, such as resolution and codecs.
            Communicate streaming audio, video or data.
         */

        return function cleanup() {
            console.log("Cleaning up");
            sub.unsubscribe();
            centrifuge.disconnect();
        };
    });


    return (
        <div>
            <div>Viewing chat # {id}</div>

            <video id="localVideo" autoPlay playsInline></video>
            <video id="remoteVideo" autoPlay playsInline></video>
            <input type="text" id="input" />
        </div>
    )
}

export default Chat;