<template>
<div v-if="streamManager" class="video-container-element">
	<ov-video :stream-manager="streamManager"/>
	<p class="video-container-element-caption">{{ clientData }}</p>
</div>
</template>

<script>
import OvVideo from './OvVideo';

export default {
	name: 'UserVideo',

	components: {
		OvVideo,
	},

	props: {
		streamManager: Object,
	},

	computed: {
		clientData () {
			const { clientData } = this.getConnectionData();
			return clientData;
		},
	},

	methods: {
		getConnectionData () {
			const { connection } = this.streamManager.stream;
			return JSON.parse(connection.data);
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