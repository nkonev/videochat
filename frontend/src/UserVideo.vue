<template>
    <div class="video-container-element" @mouseenter="showControls=true" @mouseleave="showControls=false" @click="showControls=!showControls">
        <div class="video-container-element-control" v-show="showControls" @click="suppress">
            <v-btn icon><v-icon large class="video-container-element-control-item">{{ audioMute ? 'mdi-microphone-off' : 'mdi-microphone' }}</v-icon></v-btn>
            <v-btn icon><v-icon class="video-container-element-control-item">{{ videoMute ? 'mdi-video-off' : 'mdi-video' }} </v-icon></v-btn>
            <v-btn icon><v-icon large class="video-container-element-control-item" @click="onEnterFullscreen">mdi-arrow-expand-all</v-icon></v-btn>
            <v-btn icon><v-icon large class="video-container-element-control-item">mdi-close</v-icon></v-btn>
        </div>
        <img v-show="avatarIsSet && videoMute" class="video-element" :src="avatar"/>
        <video v-show="!videoMute || !avatarIsSet" class="video-element" :id="id" autoPlay playsInline ref="videoRef" :muted="initialMuted"/>
        <p v-bind:class="[speaking ? 'video-container-element-caption-speaking' : '', errored ? 'video-container-element-caption-errored' : '', 'video-container-element-caption']">{{ userName }} <v-icon v-if="audioMute">mdi-microphone-off</v-icon><v-icon v-if="!audioMute && speaking">mdi-microphone</v-icon></p>
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
            audioMute: false,
            speaking: false,
            errorDescription: null,
            avatar: "",
            videoMute: false,
            userId: null,
            failureCount: 0,
            showControls: false
      }
    },

    props: {
        id: {
            type: String
        },
        initialMuted: {
            type: Boolean
        }
    },

    methods: {
        setSource(d) {
            console.log("Setting source for videoRef=", this.$refs.videoRef, " source=", d, " video tag id=", this.id);
            this.$refs.videoRef.srcObject = d;
        },
        getStreamId() {
            return this?.$refs?.videoRef?.srcObject?.id;
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
        suppress(e) {
            e.stopImmediatePropagation();
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
    },
    computed: {
        avatarIsSet() {
            return hasLength(this.avatar);
        },
        errored() {
            return this.failureCount > 0;
        }
    }
};
</script>

<style lang="stylus" scoped>
    .video-container-element {
        height 100%
        width min-content
        overflow-wrap anywhere
    }

    .video-container-element:nth-child(even) {
        background #d5fdd5;
    }

    .video-container-element:nth-child(odd) {
        background #e4efff;
    }

    .video-element {
        height 100% !important
    }

    .video-container-element-control {
        display inherit
        margin: 0;
        position: fixed
    }

    .video-container-element-control-item {
        cursor pointer
        text-shadow: -2px 0 white, 0 2px white, 2px 0 white, 0 -2px white;
    }


    .video-container-element-caption {
        display inherit
        margin: 0;
        top -2em
        right -0.4em
        text-shadow: -2px 0 white, 0 2px white, 2px 0 white, 0 -2px white;
        position: relative
    }

    .video-container-element-caption-speaking {
        text-shadow: -2px 0 #9cffa1, 0 2px #9cffa1, 2px 0 #9cffa1, 0 -2px #9cffa1;
    }

    .video-container-element-caption-errored {
        text-shadow: -2px 0 #ff9c9c, 0 2px #ff9c9c, 2px 0 #ff9c9c, 0 -2px #ff9c9c;
    }
</style>