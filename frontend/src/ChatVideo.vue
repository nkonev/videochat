<template>
  <v-col cols="12" class="ma-0 pa-0" id="video-container" :class="videoIsOnTopProperty ? 'video-container-position-top' : 'video-container-position-side'">
  </v-col>
</template>

<script>
import {createApp} from 'vue';
import {
  Room,
  RoomEvent,
  VideoPresets,
  createLocalTracks,
  createLocalScreenTracks,
} from 'livekit-client';
import UserVideo from "./UserVideo";
import vuetify from "@/plugins/vuetify";
import { v4 as uuidv4 } from 'uuid';
import axios from "axios";
import { retry } from '@lifeomic/attempt';
import {
    defaultAudioMute,
    getWebsocketUrlPrefix, isMobileBrowser,
    isMobileFireFox, PURPOSE_CALL
} from "@/utils";
import {
    getStoredAudioDevicePresents, getStoredCallAudioDeviceId, getStoredCallVideoDeviceId,
    getStoredVideoDevicePresents,
    NULL_CODEC,
    NULL_SCREEN_RESOLUTION,
    setStoredCallAudioDeviceId,
    setStoredCallVideoDeviceId,
} from "@/store/localStore";
import bus, {
    ADD_SCREEN_SOURCE,
    ADD_VIDEO_SOURCE, CHANGE_VIDEO_SOURCE,
    REQUEST_CHANGE_VIDEO_PARAMETERS, SET_LOCAL_MICROPHONE_MUTED,
    VIDEO_PARAMETERS_CHANGED
} from "@/bus/bus";
import {chat_name, videochat_name} from "@/router/routes";
import videoServerSettingsMixin from "@/mixins/videoServerSettingsMixin";
import refreshLocalMutedInAppBarMixin from "@/mixins/refreshLocalMutedInAppBarMixin";
import {useChatStore} from "@/store/chatStore";
import {mapStores} from "pinia";
import {goToPreservingQuery} from "@/mixins/searchString";
import pinia from "@/store/index";
import videoPositionMixin from "@/mixins/videoPositionMixin";

const first = 'first';
const second = 'second';
const last = 'last';

export default {
  mixins: [
    videoServerSettingsMixin(),
    refreshLocalMutedInAppBarMixin(),
    videoPositionMixin(),
  ],
  props: ['videoIsOnTopProperty'],
  data() {
    return {
      room: null,
      videoContainerDiv: null,
      userVideoComponents: new Map(),
      inRestarting: false,
      chatId: null,
    }
  },
  methods: {
    getNewId() {
      return uuidv4();
    },

    createComponent(userIdentity, position, videoTagId, localVideoProperties) {
      const app = createApp(UserVideo, {
        id: videoTagId,
        localVideoProperties: localVideoProperties,
        videoIsOnTop: this.videoIsOnTopProperty,
        initialShowControls: localVideoProperties != null && this.isMobile()
      });
      app.config.globalProperties.isMobile = () => {
        return isMobileBrowser()
      }
      app.use(vuetify);
      app.use(pinia);
      const containerEl = document.createElement("div");
      if (this.videoIsOnTopProperty) {
        containerEl.className = 'video-component-wrapper-position-top';
      } else {
        containerEl.className = 'video-component-wrapper-position-side';
      }

      if (position == first) {
        this.insertChildAtIndex(this.videoContainerDiv, containerEl, 0);
      } else if (position == last) {
        this.videoContainerDiv.append(containerEl);
      } else if (position == second) {
        this.insertChildAtIndex(this.videoContainerDiv, containerEl, 1);
      }
      const instance = app.mount(containerEl);

      this.addComponentForUser(userIdentity, {component: instance, app: app, containerEl: containerEl});
      return instance;
    },
    insertChildAtIndex(element, child, index) {
      if (!index) index = 0
      if (index >= element.children.length) {
        element.appendChild(child)
      } else {
        element.insertBefore(child, element.children[index])
      }
    },
    videoPublicationIsPresent (videoStream, userVideoComponents) {
      return !!userVideoComponents.filter(e => e.getVideoStreamId() == videoStream.trackSid).length
    },
    audioPublicationIsPresent (audioStream, userVideoComponents) {
      return !!userVideoComponents.filter(e => e.getAudioStreamId() == audioStream.trackSid).length
    },
    drawNewComponentOrInsertIntoExisting(participant, participantTrackPublications, position, localVideoProperties) {
      try {
        const md = JSON.parse((participant.metadata));
        const prefix = localVideoProperties ? 'local-' : 'remote-';
        const videoTagId = prefix + this.getNewId();

        const participantIdentityString = participant.identity;
        const components = this.getByUser(participantIdentityString).map(c => c.component);
        const candidatesWithoutVideo = components.filter(e => !e.getVideoStreamId());
        const candidatesWithoutAudio = components.filter(e => !e.getAudioStreamId());

        for (const track of participantTrackPublications) {
          if (track.kind == 'video') {
            console.debug("Processing video track", track);
            if (this.videoPublicationIsPresent(track, components)) {
              console.debug("Skipping video", track);
              continue;
            }
            let candidateToAppendVideo = candidatesWithoutVideo.length ? candidatesWithoutVideo[0] : null;
            console.debug("candidatesWithoutVideo", candidatesWithoutVideo, "candidateToAppendVideo", candidateToAppendVideo);
            if (!candidateToAppendVideo) {
              candidateToAppendVideo = this.createComponent(participantIdentityString, position, videoTagId, localVideoProperties);
            }
            const cameraEnabled = track && !track.isMuted;
            if (!track.isSubscribed) {
              console.warn("Video track is not subscribed");
            }
            candidateToAppendVideo.setVideoStream(track, cameraEnabled);
            console.log("Video track was set", track.trackSid, "to", candidateToAppendVideo.getId());
            candidateToAppendVideo.setUserName(md.login);
            candidateToAppendVideo.setAvatar(md.avatar);
            candidateToAppendVideo.setUserId(md.userId);
            return
          } else if (track.kind == 'audio') {
            console.debug("Processing audio track", track);
            if (this.audioPublicationIsPresent(track, components)) {
              console.debug("Skipping audio", track);
              continue;
            }
            let candidateToAppendAudio = candidatesWithoutAudio.length ? candidatesWithoutAudio[0] : null;
            console.debug("candidatesWithoutAudio", candidatesWithoutAudio, "candidateToAppendAudio", candidateToAppendAudio);
            if (!candidateToAppendAudio) {
              candidateToAppendAudio = this.createComponent(participantIdentityString, position, videoTagId, localVideoProperties);
            }
            const micEnabled = track && !track.isMuted;
            if (!track.isSubscribed) {
              console.warn("Audio track is not subscribed");
            }
            candidateToAppendAudio.setAudioStream(track, micEnabled);
            console.log("Audio track was set", track.trackSid, "to", candidateToAppendAudio.getId());
            candidateToAppendAudio.setUserName(md.login);
            candidateToAppendAudio.setAvatar(md.avatar);
            candidateToAppendAudio.setUserId(md.userId);
            return
          }
        }
        this.setError(participantTrackPublications, "Unable to draw track");
        return
      } finally {
        if (localVideoProperties) {
          this.chatStore.initializingVideoCall = false;
        }
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
      this.removeComponent(participant.identity, track);
    },

    handleLocalTrackUnpublished(trackPublication, participant) {
      const track = trackPublication.track;
      console.log('handleLocalTrackUnpublished sid=', track.sid, "kind=", track.kind);
      console.debug('handleLocalTrackUnpublished', trackPublication, "track", track);

      // when local tracks are ended, update UI to remove them from rendering
      track.detach();
      this.removeComponent(participant.identity, track);

      this.refreshLocalMicrophoneAppBarButtons();
    },
    removeComponent(userIdentity, track) {
      for (const componentWrapper of this.getByUser(userIdentity)) {
        const component = componentWrapper.component;
        const app = componentWrapper.app;
        const containerEl = componentWrapper.containerEl;
        console.debug("For removal checking component=", component, "against", track);
        if (component.getVideoStreamId() == track.sid || component.getAudioStreamId() == track.sid) {
          console.log("Removing component=", component.getId());
          try {
            app.unmount();
            this.videoContainerDiv.removeChild(containerEl);
          } catch (e) {
            console.debug("Something wrong on removing child", e, component.$el, this.videoContainerDiv);
          }
          this.removeComponentForUser(userIdentity, componentWrapper);
        }
      }
    },

    handleActiveSpeakerChange(speakers) {
      console.debug("handleActiveSpeakerChange", speakers);

      for (const speaker of speakers) {
        const userIdentity = speaker.identity;
        const tracksSids = [...speaker.audioTrackPublications.keys()];
        const userComponents = this.getByUser(userIdentity).map(c => c.component);
        for (const component of userComponents) {
          const audioStreamId = component.getAudioStreamId();
          console.debug("Track sids", tracksSids, " component audio stream id", audioStreamId);
          if (tracksSids.includes(component.getAudioStreamId())) {
            component.setSpeakingWithTimeout(1000);
          }
        }
      }
    },

    handleDisconnect() {
      console.log('disconnected from room');

      // handles kick
      if (this.$route.name == videochat_name && !this.inRestarting) {
        console.log('Handling kick');

        this.chatStore.leavingVideoAcceptableParam = true;
        const routerNewState = { name: chat_name };
        goToPreservingQuery(this.$route, this.$router, routerNewState);
      }
    },

    async setConfig() {
      await this.initServerData()
    },

    handleTrackMuted(trackPublication, participant) {
      const participantIdentityString = participant.identity;
      const components = this.getByUser(participantIdentityString).map(c => c.component);
      const matchedVideoComponents = components.filter(e => trackPublication.trackSid == e.getVideoStreamId());
      const matchedAudioComponents = components.filter(e => trackPublication.trackSid == e.getAudioStreamId());
      for (const component of matchedVideoComponents) {
        component.setVideoMute(trackPublication.isMuted);
      }
      for (const component of matchedAudioComponents) {
        component.setDisplayAudioMute(trackPublication.isMuted);
      }
    },

    async tryRestartVideoDevice() {
      this.inRestarting = true;
      for (const publication of this.room.localParticipant.getTrackPublications().values()) {
        await this.room.localParticipant.unpublishTrack(publication.track, true);
      }
      await this.createLocalMediaTracks(getStoredCallVideoDeviceId(), getStoredCallAudioDeviceId());
      bus.emit(VIDEO_PARAMETERS_CHANGED);
      this.inRestarting = false;
    },

    async startRoom(token) {
      try {
        await this.setConfig();
      } catch (e) {
        this.setError(e, "Error during fetching config");
      }

      console.log("Creating room with dynacast", this.roomDynacast, "adaptiveStream", this.roomAdaptiveStream);

      // creates a new room with options
      this.room = new Room({
        // automatically manage subscribed video quality
        adaptiveStream: this.roomAdaptiveStream,

        // optimize publishing bandwidth and CPU for simulcasted tracks
        dynacast: this.roomDynacast,
      });

      // set up event listeners
      this.room
        .on(RoomEvent.TrackSubscribed, (track, publication, participant) => {
          try {
            console.log("TrackPublished to room.name", this.room.name);
            console.debug("TrackPublished to room", this.room);
            this.drawNewComponentOrInsertIntoExisting(participant, [publication], this.getOnScreenPosition(publication), null);
          } catch (e) {
            this.setError(e, "Error during reacting on remote track published");
          }
        })
        .on(RoomEvent.TrackUnsubscribed, this.handleTrackUnsubscribed)
        .on(RoomEvent.ActiveSpeakersChanged, this.handleActiveSpeakerChange)
        .on(RoomEvent.LocalTrackUnpublished, this.handleLocalTrackUnpublished)
        .on(RoomEvent.LocalTrackPublished, () => {
          try {
            console.log("LocalTrackPublished to room.name", this.room.name);
            console.debug("LocalTrackPublished to room", this.room);

            const localVideoProperties = {
              localParticipant: this.room.localParticipant
            };
            const participantTracks = this.room.localParticipant.getTrackPublications();
            this.drawNewComponentOrInsertIntoExisting(this.room.localParticipant, participantTracks, first, localVideoProperties);

            this.refreshLocalMicrophoneAppBarButtons();
          } catch (e) {
            this.setError(e, "Error during reacting on local track published");
          }
        })
        .on(RoomEvent.TrackMuted, this.handleTrackMuted)
        .on(RoomEvent.TrackUnmuted, this.handleTrackMuted)
        .on(RoomEvent.Reconnecting, () => {
          this.setWarning("Reconnecting to video server")
        })
        .on(RoomEvent.Reconnected, () => {
          this.setOk(this.$vuetify.locale.t('$vuetify.video_successfully_reconnected'))
        })
        .on(RoomEvent.Disconnected, this.handleDisconnect)
        .on(RoomEvent.SignalConnected, () => {
          this.createLocalMediaTracks(getStoredCallVideoDeviceId(), getStoredCallAudioDeviceId());
        })
      ;

      const retryOptions = {
        delay: 200,
        maxAttempts: 5,
      };
      try {
        this.inRestarting = true;
        await retry(async (context) => {
          if (this.room) {
            await this.room.connect(getWebsocketUrlPrefix() + '/api/livekit', token, {
              // subscribe to other participants automatically
              autoSubscribe: true,
            });
            console.log('Connected to room', this.room.name);
          } else {
            console.warn("Didn't connect to room because it's null. It is ok when user leaves very fast.");
          }
          return
        }, retryOptions);
        this.inRestarting = false;
      } catch (e) {
        // If the max number of attempts was exceeded then `err`
        // will be the last error that was thrown.
        //
        // If error is due to timeout then `err.code` will be the
        // string `ATTEMPT_TIMEOUT`.
        this.setError(e, "Error during connecting to room");
      }
    },
    getOnScreenPosition(publication) {
      if (publication.source == 'screen_share') {
        return first
      }
      return second
    },
    refreshLocalMicrophoneAppBarButtons() {
      const onlyOneLocalComponentWithAudio = this.onlyOneLocalTrackWithMicrophone(this.room.localParticipant.identity);
      if (onlyOneLocalComponentWithAudio) {
        this.chatStore.canShowMicrophoneButton = true;
      } else {
        this.chatStore.canShowMicrophoneButton = false;
      }
    },
    onlyOneLocalTrackWithMicrophone(userIdentity) {
      const userComponents = this.getByUser(userIdentity).map(c => c.component);
      const localComponentsWithAudio = userComponents.filter((component) => component.isComponentLocal() && component.getAudioStreamId() != null)
      if (localComponentsWithAudio.length == 1) {
        return localComponentsWithAudio[0]
      } else {
        return null
      }
    },
    onLocalMicrophoneMutedByAppBarButton(value) {
      const onlyOneLocalComponentWithAudio = this.onlyOneLocalTrackWithMicrophone(this.room.localParticipant.identity)
      if (onlyOneLocalComponentWithAudio) {
        onlyOneLocalComponentWithAudio.doMuteAudio(value);
        const muted = onlyOneLocalComponentWithAudio.audioMute;
        this.refreshLocalMutedInAppBar(muted);
      } else {
        // just for case
        this.chatStore.canShowMicrophoneButton = false;
      }
    },

    async stopRoom() {
      console.log('Stopping room');
      await this.room.disconnect();
      this.room = null;
    },
    onAddVideoSource({videoId, audioId, isScreen}) {
      this.createLocalMediaTracks(videoId, audioId, isScreen)
    },
    async createLocalMediaTracks(videoId, audioId, isScreen) {
      let tracks = [];

      try {
        const videoResolution = VideoPresets[this.videoResolution].resolution;
        const normalizedScreenResolution = this.screenResolution === NULL_SCREEN_RESOLUTION ? undefined : VideoPresets[this.screenResolution].resolution;
        const audioIsPresents = getStoredAudioDevicePresents();
        const videoIsPresents = getStoredVideoDevicePresents();

        if (!audioIsPresents && !videoIsPresents) {
          console.warn("Not able to build local media stream, returning a successful promise");
          bus.emit(VIDEO_PARAMETERS_CHANGED, {error: 'No media configured'});
          return Promise.reject('No media configured');
        }

        console.info(
          "Creating media tracks", "isScreen", isScreen, "audioId", audioId, "videoid", videoId,
          "videoResolution", videoResolution, "screenResolution", normalizedScreenResolution,
        );

        if (isScreen) {
          tracks = await createLocalScreenTracks({
            audio: audioIsPresents,
            resolution: normalizedScreenResolution
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
              resolution: videoResolution
            } : false
          })
        }
      } catch (e) {
        this.setError(e, "Error during creating local tracks");
        this.chatStore.initializingVideoCall = false;
        return Promise.reject("Error during creating local tracks");
      }

      try {
        const isMobileFirefox = isMobileFireFox();
        console.debug("isMobileFirefox = ", isMobileFirefox, " in case Mobile Firefox simulcast for video tracks will be disabled");
        for (const track of tracks) {
          const normalizedIsScreen = !!isScreen;
          const trackName = "track_" + track.kind + "__screen_" + normalizedIsScreen + "_" + this.getNewId();
          const simulcast = !isMobileFirefox && (normalizedIsScreen ? this.screenSimulcast : this.videoSimulcast);
          const normalizedCodec = this.codec === NULL_CODEC ? undefined : this.codec;
          console.log(`Publishing local ${track.kind} screen=${normalizedIsScreen} track with name ${trackName}, simulcast ${simulcast}, codec ${normalizedCodec}`);
          const publication = await this.room.localParticipant.publishTrack(track, {
            name: trackName,
            simulcast: simulcast,
            videoCodec: normalizedCodec,
          });
          if (track.kind == 'audio' && defaultAudioMute) {
            await publication.mute();
          }
          console.info("Published track sid=", track.sid, " kind=", track.kind);
        }
      } catch (e) {
        this.setError(e, "Error during publishing local tracks");
        this.chatStore.initializingVideoCall = false;
        return Promise.reject("Error during publishing local tracks");
      }
    },
    onAddScreenSource() {
      this.createLocalMediaTracks(null, null, true);
    },
    onChangeVideoSource({videoId, audioId, purpose}) {
        if (purpose === PURPOSE_CALL) {
            setStoredCallVideoDeviceId(videoId);
            setStoredCallAudioDeviceId(audioId);
            this.tryRestartVideoDevice();
        }
    },
    addComponentForUser(userIdentity, componentWrapper) {
      let existingList = this.userVideoComponents.get(userIdentity);
      if (!existingList) {
        this.userVideoComponents.set(userIdentity, []);
        existingList = this.userVideoComponents.get(userIdentity);
      }
      existingList.push(componentWrapper);
    },

    removeComponentForUser(userIdentity, componentWrapper) {
      let existingList = this.userVideoComponents.get(userIdentity);
      if (existingList) {
        for(let i = 0; i < existingList.length; i++){
          if (existingList[i].component.getId() == componentWrapper.component.getId()) {
            existingList.splice(i, 1);
          }
        }
      }
      if (existingList.length == 0) {
        this.userVideoComponents.delete(userIdentity);
      }
    },

    getByUser(userIdentity) {
      let existingList = this.userVideoComponents.get(userIdentity);
      if (!existingList) {
        this.userVideoComponents.set(userIdentity, []);
        existingList = this.userVideoComponents.get(userIdentity);
      }
      return existingList;
    },
  },
  computed: {
    ...mapStores(useChatStore),
  },
  async mounted() {
    this.chatStore.setCallStateInCall();

    this.chatStore.initializingVideoCall = true;

    this.chatId = this.chatStore.chatDto.id;

    if (!this.isMobile() && this.videoIsAtSide()) {
      this.chatStore.showDrawerPrevious = this.chatStore.showDrawer;
      this.chatStore.showDrawer = this.shouldShowChatList();
    }

    // creates the userCallState and assigns sessionId (as part of primary key)
    // and puts this tokenId to metadata
    const enterResponse = await axios.put(`/api/video/${this.chatId}/dial/enter`, null, {
        params: {
            // in case we earlier got the token from /invite
            tokenId: this.chatStore.videoTokenId
        }
    });
    this.chatStore.videoTokenId = enterResponse.data.tokenId;

    if (!this.chatStore.showRecordStopButton && this.chatStore.canMakeRecord) {
      this.chatStore.showRecordStartButton = true;
      this.chatStore.showRecordStopButton = false;
    }

    bus.on(ADD_VIDEO_SOURCE, this.onAddVideoSource);
    bus.on(ADD_SCREEN_SOURCE, this.onAddScreenSource);
    bus.on(REQUEST_CHANGE_VIDEO_PARAMETERS, this.tryRestartVideoDevice);
    bus.on(SET_LOCAL_MICROPHONE_MUTED, this.onLocalMicrophoneMutedByAppBarButton);
    bus.on(CHANGE_VIDEO_SOURCE, this.onChangeVideoSource);

    this.videoContainerDiv = document.getElementById("video-container");

    this.startRoom(enterResponse.data.token);
  },
  beforeUnmount() {
    axios.put(`/api/video/${this.chatId}/dial/exit`, null, {
        params: {
            tokenId: this.chatStore.videoTokenId
        }
    });
    this.stopRoom().then(()=>{
      console.log("Cleaning videoContainerDiv");
      this.videoContainerDiv = null;
      this.inRestarting = false;
    });

    this.chatStore.canShowMicrophoneButton = false;

    if (!this.isMobile() && this.videoIsAtSide()) {
      this.chatStore.showDrawer = this.chatStore.showDrawerPrevious;
    }

    this.chatStore.videoChatUsersCount = 0;
    this.chatStore.showRecordStartButton = false;
    this.chatStore.initializingStaringVideoRecord = false;
    this.chatStore.initializingStoppingVideoRecord = false;

    this.chatStore.videoTokenId = null;

    this.chatStore.setCallStateReady();

    bus.off(ADD_VIDEO_SOURCE, this.onAddVideoSource);
    bus.off(ADD_SCREEN_SOURCE, this.onAddScreenSource);
    bus.off(REQUEST_CHANGE_VIDEO_PARAMETERS, this.tryRestartVideoDevice);
    bus.off(SET_LOCAL_MICROPHONE_MUTED, this.onLocalMicrophoneMutedByAppBarButton);
    bus.off(CHANGE_VIDEO_SOURCE, this.onChangeVideoSource);
  },
}

</script>

<style lang="stylus" scoped>
#video-container {
  display: flex;
  //scroll-snap-align width
  //scroll-padding 0
  height 100%
  width 100%
  //object-fit: contain;
  //box-sizing: border-box
}

.video-container-position-top {
  flex-direction: row;
  overflow-x: scroll;
  overflow-y: hidden;
  scrollbar-width: none;
}

.video-container-position-side {
  overflow-y: scroll;
  scrollbar-width: auto;
  background black
  flex-direction: column;
}


</style>

<style lang="stylus">
.video-component-wrapper-position-top {
  display contents
}

</style>
