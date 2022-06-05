<template>
    <div class="video-container-element" @mouseenter="showControls=true" @mouseleave="showControls=false">
        <div class="video-container-element-control" v-show="showControls">
            <v-btn icon @click="doMuteAudio(!audioMute)" v-if="isLocal" ><v-icon large class="video-container-element-control-item">{{ audioMute ? 'mdi-microphone-off' : 'mdi-microphone' }}</v-icon></v-btn>
            <v-btn icon @click="doMuteVideo(!videoMute)" v-if="isLocal" ><v-icon large class="video-container-element-control-item">{{ videoMute ? 'mdi-video-off' : 'mdi-video' }} </v-icon></v-btn>
            <v-btn icon @click="onEnterFullscreen"><v-icon large class="video-container-element-control-item">mdi-arrow-expand-all</v-icon></v-btn>
            <v-btn icon @click="onSetupDevice()" v-if="isLocal && isChangeable" ><v-icon large class="video-container-element-control-item">mdi-cog</v-icon></v-btn>
            <v-btn icon v-if="isLocal" @click="onClose()"><v-icon large class="video-container-element-control-item">mdi-close</v-icon></v-btn>
        </div>
        <img v-show="avatarIsSet && videoMute" class="video-element" :src="avatar"/>
        <video v-show="!videoMute || !avatarIsSet" class="video-element" :id="id" autoPlay playsInline ref="videoRef" :muted="audioMute"/>
        <p @click="showControls=!showControls" v-bind:class="[speaking ? 'video-container-element-caption-speaking' : '', errored ? 'video-container-element-caption-errored' : '', 'video-container-element-caption']">{{ userName }} <v-icon v-if="audioMute">mdi-microphone-off</v-icon><v-icon v-if="!audioMute && speaking">mdi-microphone</v-icon></p>
    </div>
</template>

<script>

import {hasLength} from "@/utils";
import bus, {CHANGE_DEVICE, DEVICE_CHANGED, OPEN_DEVICE_SETTINGS, VIDEO_PARAMETERS_CHANGED} from "@/bus";

const PUT_USER_DATA_METHOD = "putUserData";

export default {
	name: 'UserVideo',

    data()  {
        const loadingMessage = this.$vuetify.lang.t('$vuetify.loading');
	    return {
            userName: loadingMessage,
            audioMute: false,
            speaking: false,
            errorDescription: null,
            avatar: "",
            videoMute: false,
            userId: null,
            failureCount: 0,
            showControls: false,
            audioTrack: null,
            videoTrack: null
        }
    },

    props: {
        id: {
            type: String
        },
        localVideoProperties: {
            type: Object
        }
    },

    methods: {
        setAudioStream(micPub, micEnabled) {
            console.info("Setting source audio for videoRef=", this.$refs.videoRef, " track=", micPub, " audio tag id=", this.id, ", enabled=", micEnabled);
            // we don't need to hear own audio
            const realMicEnabled = micEnabled && !this.localVideoProperties;
            this.setDisplayAudioMute(!realMicEnabled);
            this.audioTrack = micPub;
            if (realMicEnabled) {
                micPub?.audioTrack?.attach(this.$refs.videoRef);
            }
        },
        hasAudioStream() {
            return this.audioTrack != null
        },
        setVideoStream(cameraPub, cameraEnabled) {
            console.info("Setting source video for videoRef=", this.$refs.videoRef, " track=", cameraPub, " video tag id=", this.id, ", enabled=", cameraEnabled);
            this.setVideoMute(!cameraEnabled);
            this.videoTrack = cameraPub;

            if (cameraEnabled) {
                cameraPub?.videoTrack?.attach(this.$refs.videoRef);
            }
        },
        hasVideoStream() {
            return this.videoTrack != null
        },

        getVideoStreamId() {
            return this.videoTrack?.trackSid;
        },
        getAudioStreamId() {
            return this.audioTrack?.trackSid;
        },
        getStream() {
            return this.stream;
        },
        getVideoElement() {
            return this?.$refs?.videoRef;
        },
        setUserName(u) {
            this.userName = u;
        },
        setDisplayAudioMute(b) {
            this.audioMute = b;
        },
        setStreamMuted(b) {
            this.$refs.videoRef.muted = b;
        },
        onEnterFullscreen(e) {
            const elem = this.$refs.videoRef;
            if (elem.requestFullscreen) {
                elem.requestFullscreen();
            } else if (elem.webkitRequestFullscreen) { // Safari
                elem.webkitRequestFullscreen();
            }
        },
        setSpeaking(speaking) {
            this.speaking = speaking;
        },
        setAvatar(a) {
            this.avatar = a;
        },
        setVideoMute(newState) {
            this.videoMute = newState;
        },
        getUserId() {
            return this.userId;
        },
        setUserId(id) {
            this.userId = id;
        },
        resetFailureCount() {
            this.failureCount = 0;
        },
        incrementFailureCount() {
            this.failureCount++;
        },
        getFailureCount() {
            return this.failureCount;
        },
        // notifyOtherParticipants() {
        //     // notify another participants, they will receive VIDEO_CALL_CHANGED
        //     const toSend = {
        //         avatar: this.avatarIsSet ? this.avatar : null,
        //         peerId: this.localVideoProperties.peerId,
        //         streamId: this.getStreamId(),
        //         videoMute: this.videoMute,
        //         audioMute: this.audioMute
        //     };
        //     this.localVideoProperties.signalLocal.notify(PUT_USER_DATA_METHOD, toSend);
        // },
        doMuteAudio(requestedState) {
            // TODO find mute and unmute in new api
            // if (requestedState) {
            //     this.getStream().mute("audio");
            //     this.setDisplayAudioMute(requestedState);
            //     this.notifyOtherParticipants();
            // } else {
            //     this.localVideoProperties.parent.ensureAudioIsEnabledAccordingBrowserPolicies();
            //     this.getStream().unmute("audio").then(value => {
            //         this.setDisplayAudioMute(requestedState);
            //         this.notifyOtherParticipants();
            //     })
            // }
        },
        doMuteVideo(requestedState) {
            // TODO find mute and unmute in new api
            // if (requestedState) {
            //     this.getStream().mute("video");
            //     this.setVideoMute(true);
            //     this.notifyOtherParticipants();
            // } else {
            //     this.getStream().unmute("video").then(value => {
            //         this.setVideoMute(false);
            //         this.notifyOtherParticipants();
            //     })
            // }
        },
        onSetupDevice() {
            bus.$emit(OPEN_DEVICE_SETTINGS, this.id);
        },
        onRequestChangeDevice({deviceId, kind, elementIdToProcess}) {
            if (elementIdToProcess != this.id) {
                return
            }
            // TODO implement in new api
            // console.log("Request to change device", deviceId, kind, "stream to change", this.getStream());
            // this.getStream().switchDevice(kind, deviceId).then(()=>{
            //     this.setStream(this.getStream());
            //     bus.$emit(DEVICE_CHANGED, null);
            // }).catch(e => {
            //     console.error("Request to change device failed", deviceId, kind, "stream to change", this.getStream(), e);
            //     bus.$emit(DEVICE_CHANGED, e);
            // });
        },
        onClose() {
            // const streamId = this.getStreamId();
            // // this.localVideoProperties.parent.clearLocalMediaStream(this.getStream());
            // this.localVideoProperties.parent.removeStream(streamId, this, this.localVideoProperties.parent.localStreams);

            // TODO send event to livekit
            // track.stop - может быть не надо
            // track.detach - уже делается . надо unpublish


            // TODO надо вызвать localParticipant.unpublishTrack()
            // delete html element - по идее где-то в родительском
        },
    },
    computed: {
        avatarIsSet() {
            return hasLength(this.avatar);
        },
        errored() {
            return this.failureCount > 0;
        },
        isLocal() {
            return !!this.localVideoProperties;
        },
        isChangeable() {
            return this.localVideoProperties && !this.localVideoProperties.screen;
        }
    },
    created(){
        bus.$on(CHANGE_DEVICE, this.onRequestChangeDevice);
    },
    destroyed() {
        bus.$off(CHANGE_DEVICE, this.onRequestChangeDevice);
    }
};
</script>

<style lang="stylus" scoped>
    .video-container-element {
        height 100%
        position relative
        display flex
        flex-direction column
        align-items: baseline;
        //width: fit-content
        //block-size: fit-content
        //box-sizing: content-box
    }

    .video-container-element:nth-child(even) {
        background #d5fdd5;
    }

    .video-container-element:nth-child(odd) {
        background #e4efff;
    }

    .video-element {
        // object-fit: contain;
        //box-sizing: border-box;
        height 100% !important
    }

    .video-container-element-control {
        z-index 1000
        position: absolute
    }

    .video-container-element-control-item {
        z-index 1001
        text-shadow: -2px 0 white, 0 2px white, 2px 0 white, 0 -2px white;
    }


    .video-container-element-caption {
        display inherit
        margin: 0;
        left 0.4em
        bottom 0.4em
        text-shadow: -2px 0 white, 0 2px white, 2px 0 white, 0 -2px white;
        position: absolute
        width: 90%;
        //word-wrap: break-word;
        //overflow-wrap: break-all
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .video-container-element-caption-speaking {
        text-shadow: -2px 0 #9cffa1, 0 2px #9cffa1, 2px 0 #9cffa1, 0 -2px #9cffa1;
    }

    .video-container-element-caption-errored {
        text-shadow: -2px 0 #ff9c9c, 0 2px #ff9c9c, 2px 0 #ff9c9c, 0 -2px #ff9c9c;
    }
</style>