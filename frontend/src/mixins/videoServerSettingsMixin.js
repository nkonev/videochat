import { hasLength, isSet } from "@/utils";
import {
    getScreenResolution, getStoredCodec,
    getStoredRoomAdaptiveStream,
    getStoredRoomDynacast,
    getStoredScreenSimulcast,
    getStoredVideoSimulcast, getVideoResolution
} from "@/store/localStore"
import axios from "axios";

export default () => {
    return  {
        data() {
            return {
                serverPreferredVideoResolution: false,
                serverPreferredScreenResolution: false,
                videoResolution: null,
                screenResolution: null,

                serverPreferredVideoSimulcast: false,
                serverPreferredScreenSimulcast: false,
                videoSimulcast: true,
                screenSimulcast: true,

                serverPreferredRoomDynacast: false,
                roomDynacast: true,

                serverPreferredRoomAdaptiveStream: false,
                roomAdaptiveStream: true,

                serverPreferredCodec: false,
                codec: null
            }
        },
        methods: {
            initServerData() {
                this.videoResolution = getVideoResolution();
                this.screenResolution = getScreenResolution();
                this.serverPreferredVideoResolution = false;
                this.serverPreferredScreenResolution = false;

                this.videoSimulcast = getStoredVideoSimulcast();
                this.screenSimulcast = getStoredScreenSimulcast()
                this.serverPreferredVideoSimulcast = false;
                this.serverPreferredScreenSimulcast = false;

                this.roomDynacast = getStoredRoomDynacast();
                this.serverPreferredRoomDynacast = false;

                this.roomAdaptiveStream = getStoredRoomAdaptiveStream();
                this.serverPreferredRoomAdaptiveStream = false;

                this.codec = getStoredCodec();
                this.serverPreferredCodec = false;

                return axios
                    .get(`/api/video/config`)
                    .then(response => response.data)
                    .then(respData => {
                        if (hasLength(respData.videoResolution)) {
                            this.serverPreferredVideoResolution = true;
                            this.videoResolution = respData.videoResolution;
                            console.log("Server overrided videoResolution to", this.videoResolution)
                        }
                        if (hasLength(respData.screenResolution)) {
                            this.serverPreferredScreenResolution = true;
                            this.screenResolution = respData.screenResolution;
                            console.log("Server overrided screenResolution to", this.screenResolution)
                        }
                        if (isSet(respData.videoSimulcast)) {
                            this.serverPreferredVideoSimulcast = true;
                            this.videoSimulcast = respData.videoSimulcast;
                            console.log("Server overrided videoSimulcast to", this.videoSimulcast)
                        }
                        if (isSet(respData.screenSimulcast)) {
                            this.serverPreferredScreenSimulcast = true;
                            this.screenSimulcast = respData.screenSimulcast;
                            console.log("Server overrided screenSimulcast to", this.screenSimulcast)
                        }
                        if (isSet(respData.roomDynacast)) {
                            this.serverPreferredRoomDynacast = true;
                            this.roomDynacast = respData.roomDynacast;
                            console.log("Server overrided roomDynacast to", this.roomDynacast)
                        }
                        if (isSet(respData.roomAdaptiveStream)) {
                            this.serverPreferredRoomAdaptiveStream = true;
                            this.roomAdaptiveStream = respData.roomAdaptiveStream;
                            console.log("Server overrided roomAdaptiveStream to", this.roomAdaptiveStream)
                        }

                        if (isSet(respData.codec)) {
                            this.serverPreferredCodec = true;
                            this.codec = respData.codec;
                            console.log("Server overrided codec to", this.codec)
                        }
                    })
            }
        }
    }
}
