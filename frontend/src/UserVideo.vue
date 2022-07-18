<template>
    <div class="video-container-element" @mouseenter="showControls=true; muteAudioBlink=false" @mouseleave="showControls=false">
        <div class="video-container-element-control" v-show="showControls">
            <v-btn v-if="isLocal" icon @click="doMuteAudio(!audioMute)" :title="audioMute ? $vuetify.lang.t('$vuetify.unmute_audio') : $vuetify.lang.t('$vuetify.mute_audio')"><v-icon large :class="['video-container-element-control-item', muteAudioBlink && audioMute ? 'info-blink' : '']">{{ audioMute ? 'mdi-microphone-off' : 'mdi-microphone' }}</v-icon></v-btn>
            <v-btn v-if="isLocal" icon @click="doMuteVideo(!videoMute)" :title="videoMute ? $vuetify.lang.t('$vuetify.unmute_video') : $vuetify.lang.t('$vuetify.mute_video')"><v-icon large class="video-container-element-control-item">{{ videoMute ? 'mdi-video-off' : 'mdi-video' }} </v-icon></v-btn>
            <v-btn icon @click="onEnterFullscreen" :title="$vuetify.lang.t('$vuetify.fullscreen')"><v-icon large class="video-container-element-control-item">mdi-arrow-expand-all</v-icon></v-btn>
            <v-btn v-if="isLocal" icon @click="onClose()" :title="$vuetify.lang.t('$vuetify.close')"><v-icon large class="video-container-element-control-item">mdi-close</v-icon></v-btn>
        </div>
        <img v-show="avatarIsSet && videoMute" class="video-element" :src="avatar"/>
        <video v-show="!videoMute || !avatarIsSet" class="video-element" :id="id" autoPlay playsInline ref="videoRef"/>
        <p @click="showControls=!showControls" v-bind:class="[speaking ? 'video-container-element-caption-speaking' : '', errored ? 'video-container-element-caption-errored' : '', 'video-container-element-caption']">{{ userName }} <v-icon v-if="audioMute">mdi-microphone-off</v-icon><v-icon v-if="!audioMute && speaking">mdi-microphone</v-icon></p>
    </div>
</template>

<script>

import {hasLength} from "@/utils";

export default {
	name: 'UserVideo',

    data()  {
        const loadingMessage = this.$vuetify.lang.t('$vuetify.loading');
	    return {
            userName: loadingMessage,
            audioMute: true,
            speaking: false,
            errorDescription: null,
            avatar: "",
            videoMute: true,
            userId: null,
            failureCount: 0,
            showControls: false,
            audioPublication: null,
            videoPublication: null,
            speakingTimer: null,
            muteAudioHover: false,
            muteVideoHover: false,
            fullscreenHover: false,
            closeHover: false,
            muteAudioBlink: true,
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

            this.setDisplayAudioMute(!micEnabled);
            this.audioPublication = micPub;
            if (!this.localVideoProperties) { // we don't need to hear own audio
                micPub?.audioTrack?.attach(this.$refs.videoRef);
            }
        },
        setVideoStream(cameraPub, cameraEnabled) {
            console.info("Setting source video for videoRef=", this.$refs.videoRef, " track=", cameraPub, " video tag id=", this.id, ", enabled=", cameraEnabled);
            this.setVideoMute(!cameraEnabled);
            this.videoPublication = cameraPub;

            cameraPub?.videoTrack?.attach(this.$refs.videoRef);
        },
        getVideoStreamId() {
            return this.videoPublication?.trackSid;
        },
        getAudioStreamId() {
            return this.audioPublication?.trackSid;
        },
        getId() {
            return this.$props.id;
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
        setSpeakingWithTimeout(timeout) {
            if (!this.speakingTimer) {
                this.speaking = true;
                this.speakingTimer = setTimeout(() => {
                    this.speaking = false;
                    this.speakingTimer = null;
                }, timeout);
            }
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
        doMuteAudio(requestedState) {
            if (requestedState) {
                this.audioPublication?.mute();
            } else {
                this.audioPublication?.unmute();
            }
            this.setDisplayAudioMute(requestedState);
        },
        doMuteVideo(requestedState) {
            if (requestedState) {
                this.videoPublication?.mute();
            } else {
                this.videoPublication?.unmute();
            }
            this.setVideoMute(requestedState);
        },
        onClose() {
            this.localVideoProperties.localParticipant.unpublishTrack(this.videoPublication?.videoTrack);
            this.localVideoProperties.localParticipant.unpublishTrack(this.audioPublication?.audioTrack);
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
        this.showControls = this.isLocal;
    },
    destroyed() {

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

    .info-blink {
        animation: blink 0.5s;
        animation-iteration-count: 10;
    }

    @keyframes blink {
        50% { opacity: 10% }
    }

</style>