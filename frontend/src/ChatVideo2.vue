<template>
    <v-col cols="12" class="ma-0 pa-0" id="video-container">
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
    Track,
    createLocalVideoTrack,
    createLocalAudioTrack,
    createLocalTracks

} from 'livekit-client';
import UserVideo from "./UserVideo";
import vuetify from "@/plugins/vuetify";
import { v4 as uuidv4 } from 'uuid';
import axios from "axios";
import {SET_SHOW_CALL_BUTTON, SET_SHOW_HANG_BUTTON, SET_VIDEO_CHAT_USERS_COUNT} from "@/store";
import {getWebsocketUrlPrefix} from "@/utils";
import bus, {ADD_VIDEO_SOURCE} from "@/bus";

const UserVideoClass = Vue.extend(UserVideo);

export default {
    data() {
        return {
            chatId: null,
            room: null,
            videoContainerDiv: null,
            userVideoComponents: []
        }
    },
    methods: {
        getNewId() {
            return uuidv4();
        },

        createComponent(prepend, videoTagId, localVideoProperties) {
            const component = new UserVideoClass({vuetify: vuetify,
                propsData: {
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
            this.userVideoComponents.push(component); // TODO remove somewhere in some close
            return component;
        },
        drawNewComponentOrGetExisting(participant, prepend, localVideoProperties) {
            const prefix = localVideoProperties ? 'local-' : 'remote-';
            const videoTagId = prefix + this.getNewId();

            const participantTracks = participant.getTracks();

            const candidatesWithoutVideo = this.userVideoComponents.filter(e => !e.getVideoStreamId());
            const candidatesWithoutAudio = this.userVideoComponents.filter(e => !e.getAudioStreamId());

            const videoPublicationIsPresent = (videoStream, userVideoComponents) => {
                return !!userVideoComponents.filter(e => e.getVideoStreamId() == videoStream.trackSid).length
            }

            const audioPublicationIsPresent = (audioStream, userVideoComponents) => {
                return !!userVideoComponents.filter(e => e.getAudioStreamId() == audioStream.trackSid).length
            }

            for (const track of participantTracks) { // track is video, before audio created an element
                if (track.kind == 'video') {
                    console.debug("Processing video track", track);
                    if (videoPublicationIsPresent(track, this.userVideoComponents)) {
                        console.debug("Skipping video", track);
                        continue;
                    }
                    let candidateToAppendVideo = candidatesWithoutVideo.length ? candidatesWithoutVideo[0] : null;
                    console.debug("candidatesWithoutVideo", candidatesWithoutVideo, "candidateToAppendVideo", candidateToAppendVideo);
                    if (!candidateToAppendVideo) {
                        candidateToAppendVideo = this.createComponent(prepend, videoTagId, localVideoProperties);
                    }
                    const cameraEnabled = track && track.isSubscribed && !track.isMuted;
                    candidateToAppendVideo.setVideoStream(track, cameraEnabled);
                    console.log("Video track was set", track, "to", candidateToAppendVideo);
                    return candidateToAppendVideo
                } else if (track.kind == 'audio') {
                    console.debug("Processing audio track", track);
                    if (audioPublicationIsPresent(track, this.userVideoComponents)) {
                        console.debug("Skipping audio", track);
                        continue;
                    }
                    let candidateToAppendAudio = candidatesWithoutAudio.length ? candidatesWithoutAudio[0] : null;
                    console.debug("candidatesWithoutAudio", candidatesWithoutAudio, "candidateToAppendAudio", candidateToAppendAudio);
                    if (!candidateToAppendAudio) {
                        candidateToAppendAudio = this.createComponent(prepend, videoTagId, localVideoProperties);
                    }
                    const micEnabled = track && track.isSubscribed && !track.isMuted;
                    candidateToAppendAudio.setAudioStream(track, micEnabled);
                    console.log("Audio track was set", track, "to", candidateToAppendAudio);
                    return candidateToAppendAudio
                }
            }
            return null
        },
        renderUserVideo(prepend, participant, localVideoProperties) {
            console.log("appendingUserVideo", participant);

            const component = this.drawNewComponentOrGetExisting(participant, prepend, localVideoProperties);
            if (!component) {
                console.warn("something wrong");
                return
            }
            const md = JSON.parse((participant.metadata));
            component.setUserName(md.login);
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
            console.log('handleTrackUnsubscribed', track);
            // remove tracks from all attached elements
            track.detach();
        },

        handleLocalTrackUnpublished(trackPublication, participant) {
            const track = trackPublication.track;
            console.log('handleLocalTrackUnpublished', trackPublication);
            // when local tracks are ended, update UI to remove them from rendering
            track.detach();
        },

        handleActiveSpeakerChange(speakers) {
            // show UI indicators when participant is speaking
        },

        handleDisconnect() {
            console.log('disconnected from room');
        },

        async onAddVideoSource(videoId, audioId) {
            console.info("onAddVideoSource", "audioId", audioId, "videoid", videoId);

            const tracks = await createLocalTracks({
                audio: {
                    deviceId: audioId,
                    echoCancellation: true,
                    noiseSuppression: true,
                },
                video: {
                    deviceId: videoId,
                    resolution: VideoPresets.h720
                }
            })
            for (const track of tracks) {
                console.info("Publishing track", track);
                this.room.localParticipant.publishTrack(track, {name: "appended" + this.getNewId()});
            }
        }
    },
    async mounted() {
        this.chatId = this.$route.params.id;

        this.$store.commit(SET_SHOW_CALL_BUTTON, false);
        this.$store.commit(SET_SHOW_HANG_BUTTON, true);

        this.videoContainerDiv = document.getElementById("video-container");

        // creates a new room with options
        this.room = new Room({
            // automatically manage subscribed video quality
            adaptiveStream: true,

            // optimize publishing bandwidth and CPU for simulcasted tracks
            dynacast: true,

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
                const localVideoProperties = {
                    localParticipant: this.room.localParticipant
                };
                this.renderUserVideo(true, this.room.localParticipant, localVideoProperties);
            })
            .on(RoomEvent.LocalTrackUnpublished, () => {
                console.log("LocalTrackUnpublished");
            });
        // connect to room
        const token = await axios.get(`/api/video/${this.chatId}/token`).then(response => response.data.token);
        console.log("Got video token", token);
        await this.room.connect(getWebsocketUrlPrefix()+'/api/livekit', token, {
            // don't subscribe to other participants automatically
            autoSubscribe: true,
        });
        console.log('connected to room', this.room.name);

        await this.onAddVideoSource(null, null);
    },
    beforeDestroy() {
        for(const component of this.userVideoComponents) {
            component.onClose();
        }
        this.room.disconnect();

        this.room = null;
        this.videoContainerDiv = null;

        this.$store.commit(SET_SHOW_CALL_BUTTON, true);
        this.$store.commit(SET_SHOW_HANG_BUTTON, false);
        this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, 0);
    },
    created() {
        bus.$on(ADD_VIDEO_SOURCE, this.onAddVideoSource);
    },
    destroyed() {
        bus.$off(ADD_VIDEO_SOURCE, this.onAddVideoSource);
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