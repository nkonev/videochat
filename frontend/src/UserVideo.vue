<template>
    <div class="video-container-element">
        <video autoPlay playsInline ref="videoRef" :muted="initialMuted" v-on:dblclick="onDoubleClick"/>
        <p class="video-container-element-caption">{{ userName }} <v-icon v-if="audioMute">mdi-microphone-off</v-icon></p>
    </div>
</template>

<script>

export default {
	name: 'UserVideo',

    data()  {
	    return {
            userName: 'loading...',
            audioMute: false
        }
    },

    props: {
        initialMuted: {
            type: Boolean
        }
    },

    methods: {
        setSource(d) {
            console.log("videoRef=", this.$refs.videoRef);
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
        setAudioMute(b) {
            this.audioMute = b;
        },
        setStreamMuted(b) {
            this.$refs.videoRef.muted = b;
        },
        onDoubleClick(e) {
            const elem = e.target;
            if (elem.requestFullscreen) {
                elem.requestFullscreen();
            } else if (elem.webkitRequestFullscreen) { // Safari
                elem.webkitRequestFullscreen();
            }
        },
    },
};
</script>

<style lang="stylus" scoped>
    .video-container-element {
        height 100%
        width min-content
    }

    .video-container-element:nth-child(even) {
        background #d5fdd5;
    }

    video {
        height 100% !important
    }

    .video-container-element-caption {
        display inherit
        margin: 0;
        top -2.5em
        right -1.2em
        text-shadow: -2px 0 white, 0 2px white, 2px 0 white, 0 -2px white;
        position: relative
        width initial
        white-space nowrap
    }
</style>