<template>
    <v-col cols="12" class="ma-0 pa-0" id="video-container">
        <div>Hi its video 2</div>
    </v-col>
</template>

<script>
import Vue from 'vue';
// https://dev.to/hulyakarakaya/how-to-fix-regeneratorruntime-is-not-defined-doj
import 'regenerator-runtime/runtime';
import {
    Room,
    RoomEvent,
    RemoteParticipant,
    RemoteTrackPublication,
    RemoteTrack,
    Participant,
    VideoPresets,
    Track
} from 'livekit-client';
import UserVideo from "./UserVideo";
import vuetify from "@/plugins/vuetify";
import { v4 as uuidv4 } from 'uuid';
import axios from "axios";

const UserVideoClass = Vue.extend(UserVideo);

export default {
    data() {
        return {
            chatId: null,
            room: null,
            videoContainerDiv: null,
            localElements: []
        }
    },
    methods: {
        getNewId() {
            return uuidv4();
        },
        appendUserVideo(prepend, participant, localVideoProperties) {
            const prefix = localVideoProperties ? 'local-' : 'remote-';
            const videoTagId = prefix + this.getNewId();

            const cameraPub = participant.getTrack(Track.Source.Camera);
            const micPub = participant.getTrack(Track.Source.Microphone);
            const micEnabled = micPub && micPub.isSubscribed && !micPub.isMuted;
            const cameraEnabled = cameraPub && cameraPub.isSubscribed && !cameraPub.isMuted;

            let component;
            // TODO remove from this array somewhere in UserVideo.close()
            const videoCandidates = this.localElements.filter(e => !e.hasVideoStream());
            const audioCandidates = this.localElements.filter(e => !e.hasAudioStream());
            console.log("appendUserVideo", cameraPub, micPub);

            if (videoCandidates.length) {
                component = videoCandidates[0];
            } else {
                component = new UserVideoClass({vuetify: vuetify,
                    propsData: {
                        initialMuted: this.remoteVideoIsMuted,
                        id: videoTagId,
                        localVideoProperties: localVideoProperties
                    }
                });
                component.$mount();
                if (prepend) {
                    this.videoContainerDiv.prepend(component.$el);
                } else {
                    this.videoContainerDiv.appendChild(component.$el);
                }
                this.localElements.push(component);
            }
            component.setAudioStream(micPub, micEnabled);
            component.setVideoStream(cameraPub, cameraEnabled);
            const md = JSON.parse((participant.metadata));
            component.setUserName(md.login);
            return component;
        },
        handleTrackSubscribed(
            track,
            publication,
            participant,
        ) {
            console.log("handleTrackSubscribed");
            if (track.kind === Track.Kind.Video || track.kind === Track.Kind.Audio) {
                // attach it to a new HTMLVideoElement or HTMLAudioElement
                const element = track.attach();
                parentElement.appendChild(element);
            }
        },

        handleTrackUnsubscribed(
            track,
            publication,
            participant,
        ) {
            // remove tracks from all attached elements
            track.detach();
        },

        handleLocalTrackUnpublished(track, participant) {
            // when local tracks are ended, update UI to remove them from rendering
            track.detach();
        },

        handleActiveSpeakerChange(speakers) {
            // show UI indicators when participant is speaking
        },

        handleDisconnect() {
            console.log('disconnected from room');
        }
    },
    async mounted() {
        this.chatId = this.$route.params.id;

        this.videoContainerDiv = document.getElementById("video-container");

        // creates a new room with options
        this.room = new Room({
            // automatically manage subscribed video quality
            adaptiveStream: true,

            // optimize publishing bandwidth and CPU for simulcasted tracks
            dynacast: true,

            // default capture settings
            videoCaptureDefaults: {
                resolution: VideoPresets.hd.resolution,
            },
        });

        // set up event listeners
        this.room
            .on(RoomEvent.TrackSubscribed, this.handleTrackSubscribed)
            .on(RoomEvent.TrackUnsubscribed, this.handleTrackUnsubscribed)
            .on(RoomEvent.ActiveSpeakersChanged, this.handleActiveSpeakerChange)
            .on(RoomEvent.Disconnected, this.handleDisconnect)
            .on(RoomEvent.LocalTrackUnpublished, this.handleLocalTrackUnpublished)
            .on(RoomEvent.LocalTrackPublished, () => {
                console.log("LocalTrackPublished", this.room);
                const localVideoProperties = {}; // todo set local video properties
                this.appendUserVideo(true, this.room.localParticipant, localVideoProperties);
            })
            .on(RoomEvent.LocalTrackUnpublished, () => {
                console.log("LocalTrackUnpublished");
            });
        // connect to room
        // TODO prefix url
        const token = await axios.get(`/api/video/${this.chatId}/token`).then(response => response.data.token);
        console.log("Got video token", token);
        await this.room.connect('ws://localhost:8081', token, {
            // don't subscribe to other participants automatically
            autoSubscribe: true,
        });
        console.log('connected to room', this.room.name);

        // publish local camera and mic tracks
        await this.room.localParticipant.enableCameraAndMicrophone();
    },
    beforeDestroy() {
        this.videoContainerDiv = null;
    }
}

</script>

<style lang="stylus" scoped>
#video-container {
    display: flex;
    flex-direction: row;
    overflow-x: scroll;
    overflow-y: hidden;
    scrollbar-width: none;
    //scroll-snap-align width
    //scroll-padding 0
    height 100%
    width 100%
    //object-fit: contain;
    //box-sizing: border-box
}

</style>