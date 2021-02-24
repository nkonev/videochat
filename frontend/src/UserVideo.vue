<template>
<div class="video-container-element">
    <video autoPlay playsInline ref="videoRef" v-on:dblclick="onDoubleClick"/>
	<p class="video-container-element-caption">{{ userName }}</p>
</div>
</template>

<script>

export default {
	name: 'UserVideo',

    data()  {
	    return {
            userName: 'user'
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
        setUserName(u) {
		    this.userName = u;
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
        display flex
        align-items flex-start
        flex-direction column
        // object-fit: scale-down;
        height 100%
        // width 100% !important
    }

    .video-container-element:nth-child(even) {
        background #d5fdd5;
    }

    video {
        //object-fit: scale-down;
        //width 100% !important
        height 100% !important // todo its
    }

    .video-container-element-caption {
        margin: 0;
        top -2.5em
        left 1.2em
        text-shadow: -2px 0 white, 0 2px white, 2px 0 white, 0 -2px white;
        position: relative;
    }
</style>