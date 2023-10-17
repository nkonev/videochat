<template>
    <div class="video-container-element" :class="videoIsOnTop ? 'video-container-element-position-top' : 'video-container-element-position-side'" ref="containerRef" @mouseenter="onMouseEnter()" @mouseleave="onMouseLeave()">
        <div class="video-container-element-control" v-show="showControls">
            <v-btn large v-if="isLocal && audioPublication != null" icon @click="doMuteAudio(!audioMute)" :title="audioMute ? $vuetify.locale.t('$vuetify.unmute_audio') : $vuetify.locale.t('$vuetify.mute_audio')"><v-icon large :class="['video-container-element-control-item', muteAudioBlink && audioMute ? 'info-blink' : '']">{{ audioMute ? 'mdi-microphone-off' : 'mdi-microphone' }}</v-icon></v-btn>
            <v-btn large v-if="isLocal && videoPublication != null" icon @click="doMuteVideo(!videoMute)" :title="videoMute ? $vuetify.locale.t('$vuetify.unmute_video') : $vuetify.locale.t('$vuetify.mute_video')"><v-icon large class="video-container-element-control-item">{{ videoMute ? 'mdi-video-off' : 'mdi-video' }} </v-icon></v-btn>
            <v-btn large icon @click="onEnterFullscreen" :title="$vuetify.locale.t('$vuetify.fullscreen')"><v-icon large class="video-container-element-control-item">mdi-arrow-expand-all</v-icon></v-btn>
            <v-btn large v-if="isLocal" icon @click="onClose()" :title="$vuetify.locale.t('$vuetify.close')"><v-icon large class="video-container-element-control-item">mdi-close</v-icon></v-btn>
        </div>
        <span v-if="!isLocal && avatarIsSet" class="video-container-element-hint">{{ $vuetify.locale.t('$vuetify.video_is_not_shown') }}</span>
        <img v-show="avatarIsSet && videoMute" @click="showControls=!showControls" class="video-element" :class="videoIsOnTop ? 'video-element-top' : 'video-element-side'" :src="avatar"/>
        <video v-show="!videoMute || !avatarIsSet" @click="showControls=!showControls" class="video-element" :class="videoIsOnTop ? 'video-element-top' : 'video-element-side'" :id="id" autoPlay playsInline ref="videoRef"/>
        <p v-bind:class="[speaking ? 'video-container-element-caption-speaking' : '', errored ? 'video-container-element-caption-errored' : '', 'video-container-element-caption']">{{ userName }} <v-icon v-if="audioMute">mdi-microphone-off</v-icon><v-icon v-if="!audioMute && speaking">mdi-microphone</v-icon></p>
    </div>
</template>

<script>

import {hasLength} from "@/utils";
import refreshLocalMutedInAppBarMixin from "@/mixins/refreshLocalMutedInAppBarMixin";

function isFullscreen(){
    return !!(document.fullscreenElement)
}

export default {
	name: 'UserVideo',

    mixins: [refreshLocalMutedInAppBarMixin()],

    data()  {
      const loadingMessage = this.$vuetify.locale.t('$vuetify.loading');
	    return {
            userName: loadingMessage,
            audioMute: true,
            speaking: false,
            errorDescription: null,
            avatar: "",
            videoMute: true,
            userId: null,
            showControls: false,
            audioPublication: null,
            videoPublication: null,
            speakingTimer: null,
            muteAudioBlink: true,
        }
    },

    props: {
        id: {
            type: String
        },
        localVideoProperties: {
            type: Object
        },
        videoIsOnTop: {
            type: Boolean
        },
        initialShowControls: {
            type: Boolean
        },
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

            if (this.isLocal) {
                this.refreshLocalMutedInAppBar(b);
            }
        },
        onEnterFullscreen(e) {
            const elem = this.$refs.containerRef;

            if (isFullscreen()) {
                document.exitFullscreen();
            } else {
                elem.requestFullscreen();
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
        doMuteAudio(requestedState) {
            if (requestedState) {
                this.audioPublication?.mute();
            } else {
                this.audioPublication?.unmute();
            }
            this.setDisplayAudioMute(requestedState);

            this.muteAudioBlink = false;
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
        isComponentLocal() {
            return this.isLocal
        },
        onMouseEnter() {
            if (!this.isMobile()) {
                this.showControls = true;
            }
        },
        onMouseLeave() {
            if (!this.isMobile()) {
                this.showControls = false;
            }
        },
    },
    computed: {
        avatarIsSet() {
            return hasLength(this.avatar);
        },
        errored() {
            return false;
        },
        isLocal() {
            return !!this.localVideoProperties;
        },
        isChangeable() {
            return this.localVideoProperties && !this.localVideoProperties.screen;
        }
    },
    created(){
        this.showControls = this.initialShowControls;
    },
    destroyed() {

    }
};
</script>

<style lang="stylus" scoped>
    .video-container-element {
        position relative
        display flex
        flex-direction column
        align-items: center;
        //width: fit-content
        //block-size: fit-content
        //box-sizing: content-box
    }

    .video-container-element-position-top {
        height 100%
    }

    .video-container-element-position-side {
        width 100%
        margin-top auto
        margin-bottom auto
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
        height: 100% !important
        min-width: 100% !important
        object-fit: cover;
        z-index 2
    }

    .video-element-side {
        width: 100% !important
    }

    .video-container-element-control {
        width 100%
        z-index 3
        position: absolute
    }

    .video-container-element-control-item {
        z-index 4
        text-shadow: -2px 0 white, 0 2px white, 2px 0 white, 0 -2px white;
    }

    .video-container-element-hint {
        z-index 1
        display inherit
        margin: 0;
        top 2em
        left 0.4em
        text-shadow: -2px 0 white, 0 2px white, 2px 0 white, 0 -2px white;
        position: absolute
        width: 90%;
        //word-wrap: break-word;
        //overflow-wrap: break-all
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .video-container-element-caption {
        z-index 2
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
        animation: blink 0.5s infinite;
    }

    @keyframes blink {
        50% { opacity: 10% }
    }

</style>
