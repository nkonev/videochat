<template>
    <div class="video-container-element">
        <video :id="id" autoPlay playsInline ref="videoRef" :muted="initialMuted" v-on:dblclick="onDoubleClick"/>
        <p v-bind:class="[speaking ? 'video-container-element-caption-speaking' : '', 'video-container-element-caption']">{{ userName }} <v-icon v-if="audioMute">mdi-microphone-off</v-icon> <v-icon v-if="!audioMute && speaking">mdi-microphone</v-icon></p>
    </div>
</template>

<script>

export default {
	name: 'UserVideo',

    data()  {
        const loadingMessage = this.$vuetify.lang.t('$vuetify.loading');
	    return {
            userName: loadingMessage,
            audioMute: false,
            speaking: false,
            errorDescription: null,
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
        setDisplayAudioMute(b) {
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
        setSpeaking(speaking) {
            this.speaking = speaking;
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

    .video-container-element:nth-child(odd) {
        background #e4efff;
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

    .video-container-element-caption-speaking {
        text-shadow: -2px 0 #9cffa1, 0 2px #9cffa1, 2px 0 #9cffa1, 0 -2px #9cffa1;
    }
</style>