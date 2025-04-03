<template>
    <div :class="videoContainerElementClass" ref="containerRef" @contextmenu.stop="onShowContextMenu($event, this)">
        <img v-show="avatarIsSet && videoMute" :class="videoElementClass" :src="avatar"/>
        <video v-show="!videoMute || !avatarIsSet" :class="videoElementClass" :id="id" autoPlay playsInline ref="videoRef"/>
        <p v-if="shouldShowCaption()" v-bind:class="[speaking ? 'video-container-element-caption-speaking' : '', 'video-container-element-caption', 'inline-caption-base']">{{ userName }} <v-icon v-if="audioMute">mdi-microphone-off</v-icon></p>

        <UserVideoContextMenu
            ref="contextMenuRef"
            isLocal="isLocal"
            :shouldShowMuteAudio="shouldShowMuteAudio()"
            :shouldShowMuteVideo="shouldShowMuteVideo()"
            :shouldShowClose="shouldShowClose()"
            :shouldShowVideoKick="shouldShowVideoKick()"
            :shouldShowAudioMute="shouldShowAudioMute()"
            :audioMute="audioMute"
            :videoMute="videoMute"
            :userName="getUserName()"
        >
        </UserVideoContextMenu>

    </div>
</template>

<script>

import {hasLength, loadingMessage} from "@/utils";
import axios from "axios";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import videoPositionMixin from "@/mixins/videoPositionMixin.js";
import speakingMixin from "@/mixins/speakingMixin.js";
import UserVideoContextMenu from "@/UserVideoContextMenu.vue";

export default {
	  name: 'UserVideo',

    mixins: [
        videoPositionMixin(),
        speakingMixin(),
    ],

    components: {
      UserVideoContextMenu,
    },

    data()  {
	    return {
            userName: loadingMessage,
            audioMute: true,
            errorDescription: null,
            avatar: "",
            videoMute: true,
            userId: null,
            audioPublication: null,
            videoPublication: null,
      }
    },

    props: {
        id: {
            type: String
        },
        localVideoProperties: {
            type: Object
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
            this.setDisplayVideoMute(!cameraEnabled);
            this.videoPublication = cameraPub;

            cameraPub?.videoTrack?.attach(this.$refs.videoRef);
        },
        getVideoStream() {
            return this.videoPublication
        },
        getAudioStream() {
            return this.audioPublication
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
        getUserName() {
            return this.userName;
        },
        setDisplayAudioMute(newState, skipStoreUpdate) {
            this.audioMute = newState;

            if (this.isLocal && !skipStoreUpdate) { // skipStoreUpdate prevents infinity recursion
                this.chatStore.localMicrophoneEnabled = !newState
            }
        },
        setAvatar(a) {
            this.avatar = a;
        },
        getAvatar() {
            return this.avatar;
        },
        setDisplayVideoMute(newState, skipStoreUpdate) {
            this.videoMute = newState;

            if (this.isLocal && !skipStoreUpdate) { // skipStoreUpdate prevents infinity recursion
              this.chatStore.localVideoEnabled = !newState
            }
        },
        getUserId() {
            return this.userId;
        },
        setUserId(id) {
            this.userId = id;
        },
        doMuteAudio(requestedState, skipStoreUpdate) {
            if (requestedState) {
                this.audioPublication?.mute();
            } else {
                this.audioPublication?.unmute();
            }
            this.setDisplayAudioMute(requestedState, skipStoreUpdate);
        },
        doMuteVideo(requestedState, skipStoreUpdate) {
            if (requestedState) {
                this.videoPublication?.mute();
            } else {
                this.videoPublication?.unmute();
            }
            this.setDisplayVideoMute(requestedState, skipStoreUpdate);
        },
        onLocalClose() {
            this.localVideoProperties.localParticipant.unpublishTrack(this.videoPublication?.videoTrack);
            this.localVideoProperties.localParticipant.unpublishTrack(this.audioPublication?.audioTrack);
        },
        isComponentLocal() {
            return this.isLocal
        },
        getVideoSource() {
          return this.videoPublication?.source
        },
        kickRemote() {
            axios.put(`/api/video/${this.chatStore.chatDto.id}/kick?userId=${this.userId}`)
        },
        forceMuteRemote() {
            axios.put(`/api/video/${this.chatStore.chatDto.id}/mute?userId=${this.userId}`)
        },
        shouldShowMuteAudio() {
            return this.isLocal && this.audioPublication != null
        },
        shouldShowMuteVideo() {
            return this.isLocal && this.videoPublication != null
        },
        shouldShowClose() {
            return this.isLocal
        },
        shouldShowVideoKick() {
            return !this.isLocal && this.canVideoKick
        },
        shouldShowAudioMute() {
            return !this.isLocal && this.canAudioMute
        },

        onShowContextMenu(e, menuableItem) {
          this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
        },
        shouldShowCaption() {
          return !(this.isMobile() && this.chatStore.presenterEnabled)
        },
    },
    computed: {
        ...mapStores(useChatStore),
        avatarIsSet() {
            return hasLength(this.avatar);
        },
        isLocal() {
            return !!this.localVideoProperties;
        },
        canVideoKick() { // only on remote
          return !this.isLocal && this.chatStore.canVideoKickParticipant(this.userId)
        },
        canAudioMute() { // only on remote
          return !this.isLocal && this.chatStore.canAudioMuteParticipant(this.userId)
        },
        videoContainerElementClass() {
          const ret = ['video-container-element'];
          if (this.videoIsHorizontal()) {
            ret.push('video-container-element-position-horizontal');
          } else if (this.videoIsGallery()) {
            ret.push('video-container-element-position-gallery');
          } else {
            ret.push('video-container-element-position-vertical');
          }
          return ret;
        },
        videoElementClass() {
          const ret = ['video-element'];
          if (this.videoIsHorizontal()) {
            ret.push('video-element-horizontal');
          } else if (this.videoIsGallery()) {
            ret.push('video-element-gallery');
          } else {
            ret.push('video-element-vertical');
          }
          return ret;
        },
    },
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

    .video-container-element-position-horizontal {
        height 100%
    }

    .video-container-element-position-vertical {
        width 100%
    }

    .video-container-element-position-gallery {
        width: var(--width);
        height: var(--height);
        background-color: #3a3a3e;
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
        object-fit: cover;
        z-index 2
    }

    .video-element-horizontal {
      height: 100% !important
      min-width: 100% !important
    }

    .video-element-vertical {
      height: 100% !important
      min-width: 100% !important
      width: 100% !important
    }

    .video-element-gallery {
      height: 100%;
      width: 100%;
    }

    .video-container-element-caption {
        max-width: calc(100% - 1em) // still needed for thin (vertical) video on mobile - it prevents bulging
    }

    .video-container-element-caption-speaking {
        text-shadow: -2px 0 #9cffa1, 0 2px #9cffa1, 2px 0 #9cffa1, 0 -2px #9cffa1;
    }

</style>
