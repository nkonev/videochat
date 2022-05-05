<template>
    <div>Hi its video 2</div>
</template>

<script>
// https://dev.to/hulyakarakaya/how-to-fix-regeneratorruntime-is-not-defined-doj
import 'regenerator-runtime/runtime';
import {
    Room,
    RoomEvent,
    RemoteParticipant,
    RemoteTrackPublication,
    RemoteTrack,
    Participant,
    VideoPresets
} from 'livekit-client';

export default {
    data() {
        return {
            room: null
        }
    },
    methods: {
        handleTrackSubscribed(
            track,
            publication,
            participant,
        ) {
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
            .on(RoomEvent.LocalTrackUnpublished, this.handleLocalTrackUnpublished);

        // connect to room
        await this.room.connect('ws://localhost:8081', "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NTQzODE3NjksImlzcyI6IkFQSXpuSnhXU2hHVzNLdCIsIm5iZiI6MTY1MTc4OTc2OSwic3ViIjoibmlraXRhIiwidmlkZW8iOnsicm9vbSI6ImNoYXQxMDAiLCJyb29tSm9pbiI6dHJ1ZX19.79xvGP8b1BbbufYCvp8xg4NeP7rpx-kaIw7I0UOXkXk", {
            // don't subscribe to other participants automatically
            autoSubscribe: false,
        });
        console.log('connected to room', this.room.name);

        // publish local camera and mic tracks
        await this.room.localParticipant.enableCameraAndMicrophone();
    },
    beforeDestroy() {

    }
}

</script>