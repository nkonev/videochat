import axios from "axios";
import {setProfile} from "./actions";
import Centrifuge from "centrifuge";

export const getProfile = (dispatch) => {
    return axios.get(`/api/profile`)
        .then(value1 => {
            return dispatch(setProfile(value1.data));
        })

};

export function setupCentrifuge() {
    // Create Centrifuge object with Websocket endpoint address set in main.go
    var url = ((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/api/chat/websocket";
    var clientId;
    var centrifuge = new Centrifuge(url, {
        onRefresh: function(ctx){
            console.debug("Dummy refresh");
        }
    });
    centrifuge.on('connect', function(ctx){
        console.log("Connected response", ctx);
        clientId = ctx.client;
        console.log('My clientId :', clientId);
    });
    centrifuge.on('disconnect', function(ctx){
        console.log("Disconnected response", ctx);
    });
    centrifuge.connect();
    return centrifuge;
}