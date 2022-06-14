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
    createLocalTracks,
    createLocalScreenTracks,
} from 'livekit-client';
import UserVideo from "./UserVideo";
import vuetify from "@/plugins/vuetify";
import { v4 as uuidv4 } from 'uuid';
import axios from "axios";
import {SET_SHOW_CALL_BUTTON, SET_SHOW_HANG_BUTTON, SET_VIDEO_CHAT_USERS_COUNT} from "@/store";
import {
    defaultAudioMute,
    getCodec,
    getStoredAudioDevicePresents,
    getStoredVideoDevicePresents,
    getVideoResolution,
    getWebsocketUrlPrefix,
    hasLength
} from "@/utils";
import bus, {
    ADD_SCREEN_SOURCE,
    ADD_VIDEO_SOURCE,
    CHANGE_DEVICE,
    REQUEST_CHANGE_VIDEO_PARAMETERS,
    VIDEO_PARAMETERS_CHANGED
} from "@/bus";

const UserVideoClass = Vue.extend(UserVideo);

export default {
    data() {
        return {
            chatId: null,
            room: null,
            videoContainerDiv: null,
            userVideoComponents: {},
            videoResolution: null,
            preferredCodec: null
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
            this.userVideoComponents[videoTagId] = component;
            return component;
        },
        videoPublicationIsPresent (videoStream, userVideoComponents) {
            return !!userVideoComponents.filter(e => e.getVideoStreamId() == videoStream.trackSid).length
        },
        audioPublicationIsPresent (audioStream, userVideoComponents) {
            return !!userVideoComponents.filter(e => e.getAudioStreamId() == audioStream.trackSid).length
        },
        drawNewComponentOrGetExisting(participant, prepend, localVideoProperties) {
            const md = JSON.parse((participant.metadata));
            const prefix = localVideoProperties ? 'local-' : 'remote-';
            const videoTagId = prefix + this.getNewId();

            const participantTracks = participant.getTracks();

            const components = Object.values(this.userVideoComponents);
            const candidatesWithoutVideo = components.filter(e => !e.getVideoStreamId());
            const candidatesWithoutAudio = components.filter(e => !e.getAudioStreamId());

            for (const track of participantTracks) { // track is video, before audio created an element
                if (track.kind == 'video') {
                    console.debug("Processing video track", track);
                    if (this.videoPublicationIsPresent(track, components)) {
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
                    console.log("Video track was set", track.trackSid, "to", candidateToAppendVideo.getId());
                    candidateToAppendVideo.setUserName(md.login);
                    candidateToAppendVideo.setAvatar(md.avatar);
                    candidateToAppendVideo.setUserId(participant.identity);
                    return candidateToAppendVideo
                } else if (track.kind == 'audio') {
                    console.debug("Processing audio track", track);
                    if (this.audioPublicationIsPresent(track, components)) {
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
                    console.log("Audio track was set", track.trackSid, "to", candidateToAppendAudio.getId());
                    candidateToAppendAudio.setUserName(md.login);
                    candidateToAppendAudio.setAvatar(md.avatar);
                    candidateToAppendAudio.setUserId(participant.identity);
                    return candidateToAppendAudio
                }
            }
            console.warn("something wrong");
            return null
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
            this.removeComponent(track);
        },

        handleLocalTrackUnpublished(trackPublication, participant) {
            const track = trackPublication.track;
            console.log('handleLocalTrackUnpublished sid=', track.sid, "kind=", track.kind);
            console.debug('handleLocalTrackUnpublished', trackPublication, "track", track);

            // when local tracks are ended, update UI to remove them from rendering
            track.detach();
            this.removeComponent(track);
        },
        removeComponent(track) {
            for (const componentId in this.userVideoComponents) {
                const component = this.userVideoComponents[componentId];
                console.debug("For removal checking component=", component, "against", track);
                if (component.getVideoStreamId() == track.sid || component.getAudioStreamId() == track.sid) {
                    console.log("Removing component=", component.getId());
                    try {
                        this.videoContainerDiv.removeChild(component.$el);
                        component.$destroy();
                    } catch (e) {
                        console.debug("Something wrong on removing child", e, component.$el, this.videoContainerDiv);
                    }
                    delete this.userVideoComponents[componentId];
                }
            }
        },

        handleActiveSpeakerChange(speakers) {
            console.debug("handleActiveSpeakerChange", speakers);
            this.enumerateAllComponents((component) => {
                component.setSpeaking(false);
            })
            const activeSpeakerIdentities = speakers.filter(s => s.isSpeaking).map(s => s.identity);
            this.enumerateAllComponents((component) => {
                if (activeSpeakerIdentities.includes(component.getUserId())) {
                    component.setSpeaking(true);
                }
            })
        },

        // TODO Optimize. We have participant's sid, participant's id, video track's sid, audio tarck's sid
        //   Seems we just need Map<UserId, []UserVideo>
        enumerateAllComponents(callback) {
            for (const videoTagId in this.userVideoComponents) {
                const component = this.userVideoComponents[videoTagId];
                if (component) {
                    callback(component);
                }
            }
        },

        handleDisconnect() {
            console.log('disconnected from room');
        },

        async setConfig() {
            const configObj = await axios
                .get(`/api/video/${this.chatId}/config`)
                .then(response => response.data)
            if (hasLength(configObj.resolution)) {
                console.log("Server overrided resolution to", configObj.resolution)
                this.videoResolution = configObj.resolution;
            } else {
                this.videoResolution = getVideoResolution();
                console.log("Used resolution from localstorage", this.videoResolution)
            }
            if (hasLength(configObj.codec)) {
                console.log("Server overrided codec to", configObj.codec)
                this.preferredCodec = configObj.codec;
            } else {
                this.preferredCodec = getCodec();
                console.log("Used codec from localstorage", this.preferredCodec)
            }
        },

        async tryRestartVideoProcess() {
            await this.stopRoom();
            await this.startRoom();
        },

        async startRoom() {
            await this.setConfig();
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
                    console.log("LocalTrackPublished to room.name", this.room.name);
                    console.debug("LocalTrackPublished to room", this.room);
                    bus.$emit(VIDEO_PARAMETERS_CHANGED);
                    const localVideoProperties = {
                        localParticipant: this.room.localParticipant
                    };
                    this.drawNewComponentOrGetExisting(this.room.localParticipant, true, localVideoProperties);
                });

            // connect to room
            const token = await axios.get(`/api/video/${this.chatId}/token`).then(response => response.data.token);
            console.log("Got video token", token);
            await this.room.connect(getWebsocketUrlPrefix()+'/api/livekit', token, {
                // don't subscribe to other participants automatically
                autoSubscribe: true,
            });
            console.log('connected to room', this.room.name);

            await this.createLocalMediaTracks(null, null);
        },

        async stopRoom() {
            await this.room.disconnect();
            this.room = null;
        },

        async createLocalMediaTracks(videoId, audioId, isScreen) {
            const preset = VideoPresets[this.videoResolution];
            const resolution = preset.resolution;
            const audioIsPresents = getStoredAudioDevicePresents();
            const videoIsPresents = getStoredVideoDevicePresents();

            if (!audioIsPresents && !videoIsPresents) {
                console.warn("Not able to build local media stream, returning a successful promise");
                bus.$emit(VIDEO_PARAMETERS_CHANGED, {error: 'No media configured'});
                return Promise.reject('No media configured');
            }

            console.info("Creating media tracks", "audioId", audioId, "videoid", videoId, "videoResolution", resolution, "preferredCodec", this.preferredCodec);

            let tracks;
            if (isScreen) {
                tracks = await createLocalScreenTracks({
                    audio: audioIsPresents,
                    resolution: resolution
                });
            } else {
                tracks = await createLocalTracks({
                    audio: audioIsPresents ? {
                        deviceId: audioId,
                        echoCancellation: true,
                        noiseSuppression: true,
                    } : false,
                    video: videoIsPresents ? {
                        deviceId: videoId,
                        resolution: resolution
                    } : false
                })
            }
            for (const track of tracks) {
                const publication = await this.room.localParticipant.publishTrack(track, {
                    name: "track_" + track.kind + "__screen_" + isScreen + "_" + this.getNewId(),
                    videoCodec: this.preferredCodec
                });
                if (track.kind == 'audio' && defaultAudioMute) {
                    await publication.mute();
                }
                console.info("Published track sid=", track.sid, " kind=", track.kind);
            }
        },
        onAddScreenSource() {
            this.createLocalMediaTracks(null, null, true);
        }
    },
    async mounted() {
        this.chatId = this.$route.params.id;

        this.$store.commit(SET_SHOW_CALL_BUTTON, false);
        this.$store.commit(SET_SHOW_HANG_BUTTON, true);

        this.videoContainerDiv = document.getElementById("video-container");

        this.startRoom();
    },
    beforeDestroy() {
        this.stopRoom();

        this.videoContainerDiv = null;

        this.$store.commit(SET_SHOW_CALL_BUTTON, true);
        this.$store.commit(SET_SHOW_HANG_BUTTON, false);
        this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, 0);
    },
    created() {
        bus.$on(ADD_VIDEO_SOURCE, this.createLocalMediaTracks);
        bus.$on(ADD_SCREEN_SOURCE, this.onAddScreenSource);
        bus.$on(REQUEST_CHANGE_VIDEO_PARAMETERS, this.tryRestartVideoProcess);
    },
    destroyed() {
        bus.$off(ADD_VIDEO_SOURCE, this.createLocalMediaTracks);
        bus.$off(ADD_SCREEN_SOURCE, this.onAddScreenSource);
        bus.$off(REQUEST_CHANGE_VIDEO_PARAMETERS, this.tryRestartVideoProcess);
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