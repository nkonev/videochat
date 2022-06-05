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
        appendUserVideo2(prepend, participant, localVideoProperties) {
            const prefix = localVideoProperties ? 'local-' : 'remote-';
            const videoTagId = prefix + this.getNewId();

            const allTracks = participant.getTracks();

            const cameraPubs = allTracks.filter(track => track.kind == "video");
            const micPubs = allTracks.filter(track => track.kind == "audio");
            const cameraPub = cameraPubs.length ? cameraPubs[0] : null;
            const micPub = micPubs.length ? micPubs[0] : null;

            const videoPublicationIsSet = (videoStream, userVideoComponents) => {
                return !!userVideoComponents.filter(e => e.getVideoStreamId() == videoStream.trackSid).length
            }

            const audioPublicationIsSet = (audioStream, userVideoComponents) => {
                return !!userVideoComponents.filter(e => e.getAudioStreamId() == audioStream.trackSid).length
            }

            let component;
            // this.userVideoComponents.filter(e => e.getVideoStreamId())
            if (!videoPublicationIsSet(cameraPub, this.userVideoComponents)) {
                const localVideoCandidates = this.userVideoComponents.filter(e => !e.hasVideoStream());
                if (localVideoCandidates.length) {
                    component = localVideoCandidates[0];
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
                    // TODO remove from this array somewhere in UserVideo.close()
                    this.userVideoComponents.push(component);
                }
            }
            if (!component && !audioPublicationIsSet(micPub, this.userVideoComponents)) {
                const localVideoCandidates = this.userVideoComponents.filter(e => !e.hasAudioStream());
                if (localVideoCandidates.length) {
                    component = localVideoCandidates[0];
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
                    // TODO remove from this array somewhere in UserVideo.close()
                    this.userVideoComponents.push(component);
                }
            }
            console.log("appenqdUserVideo", cameraPub, micPub);

            const micEnabled = micPub && micPub.isSubscribed && !micPub.isMuted;
            const cameraEnabled = cameraPub && cameraPub.isSubscribed && !cameraPub.isMuted;
            component.setAudioStream(micPub, micEnabled);
            component.setVideoStream(cameraPub, cameraEnabled);
            const md = JSON.parse((participant.metadata));
            component.setUserName(md.login);
            return component;
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
            this.userVideoComponents.push(component);
            return component;
        },
        drawNewComponentOrGetExisting(participantTracks, videoTagId, prepend, localVideoProperties) {
            const candidatesWithoutVideo = this.userVideoComponents.filter(e => !e.hasVideoStream());
            const candidatesWithoutAudio = this.userVideoComponents.filter(e => !e.hasAudioStream());

            const videoPublicationIsPresent = (videoStream, userVideoComponents) => {
                return !!userVideoComponents.filter(e => e.getVideoStreamId() == videoStream.trackSid).length
            }

            const audioPublicationIsPresent = (audioStream, userVideoComponents) => {
                return !!userVideoComponents.filter(e => e.getAudioStreamId() == audioStream.trackSid).length
            }

            for (const track of participantTracks) { // track is video, before audio created an element
                if (track.kind == 'video') {
                    console.log("Processing video track", track);
                    if (videoPublicationIsPresent(track, this.userVideoComponents)) {
                        console.log("Skipping video", track);
                        continue;
                    }
                    //let candidateToAppendVideo = candidatesWithoutVideo.find(e => e.getVideoStreamId() == track.trackSid);
                    let candidateToAppendVideo = candidatesWithoutVideo.length ? candidatesWithoutVideo[0] : null;
                    console.log("candidatesWithoutVideo", candidatesWithoutVideo, "candidateToAppendVideo", candidateToAppendVideo);
                    if (!candidateToAppendVideo) {
                        candidateToAppendVideo = this.createComponent(prepend, videoTagId, localVideoProperties);
                    }
                    const cameraEnabled = track && track.isSubscribed && !track.isMuted;
                    candidateToAppendVideo.setVideoStream(track, cameraEnabled);
                    console.log("Video track was set");
                    return candidateToAppendVideo
                } else if (track.kind == 'audio') {
                    console.log("Processing audio track", track);
                    if (audioPublicationIsPresent(track, this.userVideoComponents)) {
                        console.log("Skipping audio", track);
                        continue;
                    }
                    // let candidateToAppendAudio = candidatesWithoutAudio.find(e => e.getAudioStreamId() == track.trackSid);
                    let candidateToAppendAudio = candidatesWithoutAudio.length ? candidatesWithoutAudio[0] : null;
                    console.log("candidatesWithoutAudio", candidatesWithoutAudio, "candidateToAppendAudio", candidateToAppendAudio);
                    if (!candidateToAppendAudio) {
                        candidateToAppendAudio = this.createComponent(prepend, videoTagId, localVideoProperties);
                    }
                    const micEnabled = track && track.isSubscribed && !track.isMuted;
                    candidateToAppendAudio.setAudioStream(track, micEnabled);
                    console.log("Audio track was set");
                    return candidateToAppendAudio
                }
            }
            return null
        },
        appendUserVideo(prepend, participant, localVideoProperties) {
            const prefix = localVideoProperties ? 'local-' : 'remote-';
            const videoTagId = prefix + this.getNewId();

            const allTracks = participant.getTracks();
            console.log("appendingUserVideo", participant);

            const component = this.drawNewComponentOrGetExisting(allTracks, videoTagId, prepend, localVideoProperties);
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
            console.log('handleLocalTrackUnpublished', track);
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
            /*const onlyVideo = audioId == null;
            const onlyAudio = videoId == null;
            const tracks = await this.room.localParticipant.createTracks({
                audio: {deviceId: onlyVideo ? false : audioId},
                video: {deviceId: onlyAudio ? false : videoId }
            });
            if (!tracks.length) {
                console.warn("No tracks found");
            } else {
                console.info("Found tracks", tracks);
                if (onlyVideo) {
                    const filteredTracks = tracks.filter(track => track.kind == "video");
                    for (const track of filteredTracks) {
                        console.info("1 Publishing track", track);
                        this.room.localParticipant.publishTrack(track);
                    }
                } else if (onlyAudio) {
                    const filteredTracks = tracks.filter(track => track.kind == "audio");
                    for (const track of filteredTracks) {
                        console.info("2 Publishing track", track);
                        this.room.localParticipant.publishTrack(track);
                    }
                } else {
                    for (const track of tracks) {
                        console.info("3 Publishing track", track);
                        this.room.localParticipant.publishTrack(track);
                    }
                }
            }*/
            /*const videoTrack = await createLocalVideoTrack({
                deviceId: videoId
            })
            const audioTrack = await createLocalAudioTrack({
                deviceId: audioId,
                echoCancellation: true,
                noiseSuppression: true,
            })
                        const videoPublication = await this.room.localParticipant.publishTrack(videoTrack);
            const audioPublication = await this.room.localParticipant.publishTrack(audioTrack);
            */

            const tracks = await createLocalTracks({
                audio: {
                    deviceId: audioId,
                    echoCancellation: true,
                    noiseSuppression: true,
                },
                video: {
                    deviceId: videoId,
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

            // default capture settings
            // videoCaptureDefaults: {
            //     resolution: VideoPresets.h720.resolution,
            // },
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
        const token = await axios.get(`/api/video/${this.chatId}/token`).then(response => response.data.token);
        console.log("Got video token", token);
        await this.room.connect(getWebsocketUrlPrefix()+'/api/livekit', token, {
            // don't subscribe to other participants automatically
            autoSubscribe: true,
        });
        console.log('connected to room', this.room.name);

        // publish local camera and mic tracks
        // await this.room.localParticipant.enableCameraAndMicrophone();


        const videoTrack = await createLocalVideoTrack({
            facingMode: "user",
            // preset resolutions
            resolution: VideoPresets.h720
        })
        const audioTrack = await createLocalAudioTrack({
            echoCancellation: true,
            noiseSuppression: true,
        })
        const videoPublication = await this.room.localParticipant.publishTrack(videoTrack, {name: "initialvideo"+this.getNewId()});
        const audioPublication = await this.room.localParticipant.publishTrack(audioTrack, {name: "initialaudio"+this.getNewId()});

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