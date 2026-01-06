<template>
    <splitpanes ref="splVideo" id="video-splitpanes" class="default-theme" :dbl-click-splitter="false" :horizontal="splitpanesIsHorizontal" @resize="onPanelResized($event)" @pane-add="onPanelAdd($event)" @pane-remove="onPanelRemove($event)">

        <ChatVideoPresenter v-if="!shouldUseReverseOrder() && shouldShowPresenter" :provider="this" ref="presenterRef"/>

        <pane :class="paneVideoContainerClass"  :size="miniaturesPaneSize()">
          <v-col cols="12" class="ma-0 pa-0" id="video-container" :class="videoContainerClass"  @click="onClickFromVideos()"></v-col>
          <VideoButtons v-if="!isMobile() && !shouldShowPresenter" @requestFullScreen="onButtonsFullscreen" v-show="showControls"/>
        </pane>

        <ChatVideoPresenter v-if="shouldUseReverseOrder() && shouldShowPresenter" :provider="this" ref="presenterRef"/>
    </splitpanes>
    <VideoButtons v-if="isMobile()" @requestFullScreen="onButtonsFullscreen" v-show="showControls"/>
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
  getWebsocketUrlPrefix,
  hasLength,
  isFullscreen,
  isMobileBrowser,
  PURPOSE_CALL,
  goToPreservingQuery,
  setSplitter,
  isMicrophoneEnabled
} from "@/utils";
import {
  getStoredAudioDevicePresents, getStoredCallAudioDeviceId, getStoredCallVideoDeviceId,
  getStoredVideoDevicePresents, getStoredVideoMiniatures, isNoLocalTracks, getStoredAutoMicrophoneEnabled,
  NULL_CODEC,
  NULL_SCREEN_RESOLUTION,
  setStoredCallAudioDeviceId,
  setStoredCallVideoDeviceId,
} from "@/store/localStore";
import bus, {
  ADD_SCREEN_SOURCE,
  ADD_VIDEO_SOURCE, CHANGE_VIDEO_SOURCE, PIN_VIDEO,
  REQUEST_CHANGE_VIDEO_PARAMETERS, UN_PIN_VIDEO,
  VIDEO_PARAMETERS_CHANGED
} from "@/bus/bus";
import {chat_name, videochat_name} from "@/router/routes";
import videoServerSettingsMixin from "@/mixins/videoServerSettingsMixin";
import {useChatStore} from "@/store/chatStore";
import {mapStores} from "pinia";
import pinia from "@/store/index";
import videoPositionMixin from "@/mixins/videoPositionMixin";
import { Splitpanes, Pane } from 'splitpanes';
import {largestRect} from "rect-scaler";
import debounce from "lodash/debounce";
import VideoButtons from "./VideoButtons.vue"
import speakingMixin from "@/mixins/speakingMixin.js";
import UserVideoContextMenu from "@/UserVideoContextMenu.vue";
import {SEARCH_MODE_MESSAGES} from "@/mixins/searchString.js";
import ChatVideoPresenter from "@/ChatVideoPresenter.vue";

const first = 'first';
const second = 'second';
const last = 'last';

const classVideoComponentWrapperPositionHorizontal = 'video-component-wrapper-position-horizontal';
const classVideoComponentWrapperPositionVertical = 'video-component-wrapper-position-vertical';
const classVideoComponentWrapperPositionGallery = 'video-component-wrapper-position-gallery';

const panelSizesKey = "videoPanelSizes";

const emptyStoredPanes = () => {
  return {
    presenterPane: 80
  }
}

const videoSplitpanesSelector = "#video-splitpanes";
const videoSplitterDisplayVarName = "--splitter-h-display";

export default {
  mixins: [
    videoServerSettingsMixin(),
    videoPositionMixin(),
    speakingMixin(),
  ],
  props: ['chatId'],
  data() {
    return {
      room: null,
      videoContainerDiv: null,
      userVideoComponents: new Map(),
      inRestarting: false,
      presenterData: null,
      presenterVideoMute: false,
      presenterAudioMute: true,
      showControls: true,

      localTrackDrawn: false,
      localTrackCreatedAndPublished: false,
      finishedConnectingToRoom: false,
    }
  },
  methods: {
    getNewId() {
      return uuidv4();
    },

    setUserVideoWrapperClass(containerEl, videoIsHorizontal, videoIsGallery) {
      if (videoIsHorizontal) { // see also watch chatStore.videoPosition
        containerEl.className = classVideoComponentWrapperPositionHorizontal;
      } else if (videoIsGallery) {
        containerEl.className = classVideoComponentWrapperPositionGallery;
      } else {
        containerEl.className = classVideoComponentWrapperPositionVertical;
      }
    },
    createComponent(userIdentity, position, videoTagId, localVideoProperties) {
      const app = createApp(UserVideo, {
        id: videoTagId,
        localVideoProperties: localVideoProperties,
        loadingName: this.$vuetify.locale.t('$vuetify.user_video_loading'),
      });
      app.config.globalProperties.isMobile = () => {
        return isMobileBrowser()
      }
      app.use(vuetify);
      app.use(pinia);
      const containerEl = document.createElement("div");

      this.setUserVideoWrapperClass(containerEl, this.videoIsHorizontal(), this.videoIsGallery());

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

            const data = this.getDataForPresenter(candidateToAppendVideo);
            this.updatePresenterIfNeed(data, false);
            this.recalculateLayout();

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

            const data = this.getDataForPresenter(candidateToAppendAudio);
            this.updatePresenterIfNeed(data, false); // append an audio to this.presenterData to correct working "muted" on Presenter
            this.recalculateLayout();

            return
          }
        }
        this.setError(participantTrackPublications, "Unable to draw track");
        return
      } finally {
        if (localVideoProperties) {
          this.localTrackDrawn = true;
          this.updateInitializingVideoCall();
        }
      }
    },
    updateInitializingVideoCall() {
      // kinda set "initializing is complete" when all the stages are successful
      // because of possibly 2 local tracks (audio + video) we wait for localTrackCreatedAndPublished too
      this.chatStore.initializingVideoCall = !(((this.localTrackDrawn && this.localTrackCreatedAndPublished) || isNoLocalTracks()) && this.finishedConnectingToRoom);
    },
    setStopInitializingVideoCallOnError() {
      this.chatStore.initializingVideoCall = false;
    },
    getPresenterPriority(pub, isSpeaking) {
      if (!pub) {
        return -1
      }
      if (pub.trackSid === this.chatStore.pinnedTrackSid) {
        return 5
      }
      for (const t of this.room.localParticipant.getTrackPublications().values()) {
        if (t.trackSid === pub.trackSid)  {
          return 0
        }
      }
      switch (pub.source) {
        case "camera":
          return isSpeaking ? 3 : 2;
        case "screen_share":
          return 4
        default:
            return 1
      }
    },
    onPinVideo(trackSid) {
      console.log("pinning", trackSid);
      this.chatStore.pinnedTrackSid = trackSid;
      this.electNewPresenter();
    },
    onUnpinVideo(){
      this.chatStore.pinnedTrackSid = null;
      this.electNewPresenter();
    },
    doUnpinVideo() {
      bus.emit(UN_PIN_VIDEO)
    },
    // TODO think how to reuse the presenter mode with egress
    detachPresenter() {
      if (this.presenterData) {

        // we use "this.$refs.presenterRef?"
        // in order to give time to finish detachPresenter() while ChatVideoPresenter is still on screen
        // else there is a bug, its reproduction:
        // 1. In prod, open video call
        // 2. Open "Messages", enable presenter, switch gallery -> vertical, vertical -> gallery
        // 3. Exit from the video call
        // 4. In the Console the message detachPresenter -> "this.$refs.presenterRef is null" is going to appear
        this.presenterData.videoStream?.videoTrack?.detach(this.$refs.presenterRef?.$refs.presenterVideoRef);
        this.presenterData = null;
      }
    },
    updatePresenter(data) {
      if (data?.videoStream) {
        this.detachPresenter();
        data.videoStream.videoTrack?.attach(this.$refs.presenterRef.$refs.presenterVideoRef);
        this.presenterData = data;
        this.updatePresenterVideoMute();
      }
      if (data?.audioStream) {
        this.updatePresenterAudioMute();
      }
    },
    updatePresenterIfNeed(data, isSpeaking) {
        if (this.chatStore.presenterEnabled && this.canUsePresenter()) {

          // replace presenter
          if (data.videoStream?.trackSid != null && this.presenterData?.videoStream?.trackSid !== data.videoStream.trackSid &&
              this.getPresenterPriority(data.videoStream, isSpeaking) > this.getPresenterPriority(this.presenterData?.videoStream)
          ) {
            this.detachPresenter();
            this.updatePresenter(data);
          }

          // append an audio stream to the existing presenter with only video
          if (this.presenterData?.videoStream != null && this.presenterData.audioStream == null && data.audioStream && this.presenterData?.videoStream?.trackSid === data.videoStream?.trackSid) {
            console.log("Appending an audio stream to the presenter");
            this.presenterData.audioStream = data.audioStream;
          }

          // append a video stream to the existing presenter with only audio
          if (this.presenterData?.audioStream != null && this.presenterData.videoStream == null && data.videoStream && this.presenterData?.audioStream?.trackSid === data.audioStream?.trackSid) {
            console.log("Appending a video stream to the presenter");
            this.presenterData.videoStream = data.videoStream;
          }

          // set is speaking
          if (this.presenterData?.videoStream?.trackSid != null && data.videoStream != null && this.presenterData?.videoStream?.trackSid === data.videoStream.trackSid && isSpeaking) {
            this.setSpeakingWithDefaultTimeout();
          }
        }
    },
    updatePresenterVideoMute() {
      this.presenterVideoMute = this.getPresenterVideoMute();
    },
    getPresenterVideoMute() {
      const p = this.presenterData?.videoStream;
      if (p) {
        const t = p.videoTrack;
        if (t) {
          return t.isMuted
        }
      }
      return true
    },
    updatePresenterAudioMute() {
      this.presenterAudioMute = this.getPresenterAudioMute();
    },
    getPresenterAudioMute() {
      const p = this.presenterData?.audioStream;
      if (p) {
        const t = p.audioTrack;
        if (t) {
          return t.isMuted
        }
      }
      return true
    },
    canUsePresenterPlain(v) {
      return !this.videoIsGalleryPlain(v);
    },
    canUsePresenter() {
      const v = this.chatStore.videoPosition;
      return this.canUsePresenterPlain(v);
    },
    handleTrackUnsubscribed(
      track,
      publication,
      participant,
    ) {
      console.log('handleTrackUnsubscribed', track);
      this.removeComponentIfNeed(participant.identity, track);
    },

    handleLocalTrackUnpublished(trackPublication, participant) {
      const track = trackPublication.track;
      console.log('handleLocalTrackUnpublished sid=', track.sid, "kind=", track.kind);
      console.debug('handleLocalTrackUnpublished', trackPublication, "track", track);

      this.removeComponentIfNeed(participant.identity, track);

      this.refreshLocalMuteAppBarButtons();
    },
    electNewPresenter() {
      const data = this.getAnyPrioritizedVideoData();
      if (data) {
        this.updatePresenter(data);
      }
    },
    removeComponentIfNeed(userIdentity, track) {
      // remove tracks from all attached elements
      track.detach();

      let presenterWasUnpinned = false;
      if (track.sid === this.chatStore.pinnedTrackSid) {
        this.chatStore.pinnedTrackSid = null;
        this.detachPresenter();
        presenterWasUnpinned = true;
      }

      let presenterWasDetached = false;
      for (const componentWrapper of this.getByUser(userIdentity)) {
        const component = componentWrapper.component;
        const app = componentWrapper.app;
        const containerEl = componentWrapper.containerEl;
        console.debug("For removal checking component=", component, "against", track);
        // it's correct regardless weird design
        // testcase:
        // user1 and user2 start a video chat
        // user1 starts screen sharing
        // user1 closes video with screen sharing
        // camera video of user1 should continue to work
        if (component.getVideoStreamId() == track.sid || component.getAudioStreamId() == track.sid) {
            console.log("Setting null video for component=", component.getId());
            if (this.chatStore.presenterEnabled && this.presenterData?.videoStream && this.presenterData.videoStream.trackSid == component.getVideoStreamId()) {
              this.detachPresenter();
              presenterWasDetached = true;
            }
            try {
              console.log("Removing component=", component.getId());
              app.unmount();
              this.videoContainerDiv.removeChild(containerEl);
              this.removeComponentForUser(userIdentity, componentWrapper);

              console.log("Successfully removed component=", component.getId());

              this.recalculateLayout();
              break
            } catch (e) {
              console.debug("Something wrong on removing child", e, component.$el, this.videoContainerDiv);
            }
        }
      }

      if (presenterWasDetached || presenterWasUnpinned) {
        this.electNewPresenter();
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
          if (tracksSids.includes(audioStreamId)) {
            if (speaker.isSpeaking) {
              component.setSpeakingWithDefaultTimeout();
              const data = this.getDataForPresenter(component);
              this.updatePresenterIfNeed(data, true);
            } else {
              component.resetSpeaking();
            }
          }
        }
      }
    },
    getDataForPresenter(component) {
      const id = component.getUserId();
      const userName = component.getUserName();
      const videoPublication = component.getVideoStream();
      const audioPublication = component.getAudioStream();
      const avatar = component.getAvatar();
      const isScreenShare = videoPublication?.source == "screen_share";
      return {videoStream: videoPublication, audioStream: audioPublication, avatar: avatar, userId: id, userName: userName, isScreenShare: isScreenShare}
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
        component.setDisplayVideoMute(trackPublication.isMuted);
        if (component.getVideoStreamId() && this.presenterData?.videoStream && component.getVideoStreamId() == this.presenterData.videoStream.trackSid) {
          this.presenterVideoMute = trackPublication.isMuted;
        }
      }
      for (const component of matchedAudioComponents) {
        component.setDisplayAudioMute(trackPublication.isMuted);
        if (component.getAudioStreamId() && this.presenterData?.audioStream && component.getAudioStreamId() == this.presenterData.audioStream.trackSid) {
          this.presenterAudioMute = trackPublication.isMuted;
        }
      }
    },

    async stopLocalTracks() {
      for (const publication of this.room.localParticipant.getTrackPublications().values()) {
        await this.room.localParticipant.unpublishTrack(publication.track, true);
      }
    },
    async tryRestartVideoDevice() {
      this.inRestarting = true;
      await this.stopLocalTracks();
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

            this.refreshLocalMuteAppBarButtons();
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

      // although we can pass retryNum to Room constructor, actually it doesn't work
      //
      // testcase:
      // setup ssh-vpn socks 5 on Firefox 131
      // connect to the video call
      // it takes 1 or some retries
      //
      // without this retry it's going to just return the error to user
      const maxAttempts = 10;
      const retryOptions = {
        delay: 200,
        maxAttempts: maxAttempts,
      };
      try {
        this.inRestarting = true;
        await retry(async (context) => {
          const attempt = context.attemptNum + 1
          if (attempt > 1) {
            this.setWarning("Connecting to the room, attempt " + attempt + " / " + maxAttempts);
          }

          if (this.room) {
            await this.room.connect(getWebsocketUrlPrefix() + '/api/livekit', token, {
              // subscribe to other participants automatically
              autoSubscribe: true,
            });
            console.log('Connected to room', this.room.name);

            // after a several attempts (Firefox + ssh tunnel)
            // it turns into true
            // and the spinner on red tube button disappears
            // so we can leave the call, and we won't get an error "this.room is null in createLocalMediaTracks()"
            // because of this retry
            this.finishedConnectingToRoom = true;
            this.updateInitializingVideoCall();
            this.closeError();
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

        this.setStopInitializingVideoCallOnError(); // stop the spinner
      }
    },
    getOnScreenPosition(publication) {
      if (publication.source == 'screen_share') {
        return first
      }
      return second
    },
    refreshLocalMuteAppBarButtons() {
      if (this.onlyOneLocalTrackWith(this.room.localParticipant.identity)) {
        this.chatStore.canShowMicrophoneButton = true;
      } else {
        this.chatStore.canShowMicrophoneButton = false;
      }

      if (this.onlyOneLocalTrackWith(this.room.localParticipant.identity, true)) {
        this.chatStore.canShowVideoButton = true;
      } else {
        this.chatStore.canShowVideoButton = false;
      }
    },
    onlyOneLocalTrackWith(userIdentity, video) {
      const userComponents = this.getByUser(userIdentity).map(c => c.component);
      const localComponentsWith = userComponents.filter((component) => {
        if (component.isComponentLocal()) {
          if (video) {
            if (component.getVideoSource() === "screen_share") {
              return false
            }
            return component.getVideoStreamId() != null
          } else {
            return component.getAudioStreamId() != null
          }
        }
        return false
      });
      if (localComponentsWith.length == 1) {
        return localComponentsWith[0]
      } else {
        return null
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

        if (isNoLocalTracks()) {
          console.info("Not able to build local media stream, returning a successful promise");
          return Promise.resolve('No media configured');
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
        this.setStopInitializingVideoCallOnError();
        return Promise.reject("Error during creating local tracks");
      }

      try {
        for (const track of tracks) {
          const normalizedIsScreen = !!isScreen;
          const trackName = "track_" + track.kind + "__screen_" + normalizedIsScreen + "_" + this.getNewId();
          const simulcast = (normalizedIsScreen ? this.screenSimulcast : this.videoSimulcast);
          const normalizedCodec = this.codec === NULL_CODEC ? undefined : this.codec;
          console.log(`Publishing local ${track.kind} screen=${normalizedIsScreen} track with name ${trackName}, simulcast ${simulcast}, codec ${normalizedCodec}`);
          const publication = await this.room.localParticipant.publishTrack(track, {
            name: trackName,
            simulcast: simulcast,
            videoCodec: normalizedCodec,
          });

          if (track.kind == 'audio') {
            const isMicroEnabledPermission = await isMicrophoneEnabled();
            const autoMicrophoneEnabled = getStoredAutoMicrophoneEnabled();
            const isMicroEnabled = isMicroEnabledPermission && autoMicrophoneEnabled;

            if (!isMicroEnabled) {
              await publication.mute();
            } else {
              this.chatStore.localMicrophoneEnabled = true;
            }
          }
          console.info("Published track sid=", track.sid, " kind=", track.kind);
        }
        this.localTrackCreatedAndPublished = true;
        this.updateInitializingVideoCall();
        return Promise.resolve(true);
      } catch (e) {
        this.setError(e, "Error during publishing local tracks");
        this.setStopInitializingVideoCallOnError();
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
    getAnyPrioritizedVideoData() {
      const tmp = [];
      for (const [_, list] of this.userVideoComponents) {
        for (const componentWrapper of list) {
          const data = this.getDataForPresenter(componentWrapper.component);
          if (data.videoStream && data.videoStream.kind == "video") {
            tmp.push(data);
          }
        }
      }

      tmp.sort((a, b) => {
        return this.getPresenterPriority(b.videoStream) - this.getPresenterPriority(a.videoStream);
      });

      if (tmp.length) {
        return tmp[0]
      }

      return null;
    },
    recalculateLayout() {
      const gallery = document.getElementById("video-container");
      if (gallery) {
        const screenWidth = gallery.getBoundingClientRect().width;
        const screenHeight = gallery.getBoundingClientRect().height;
        const videoCount = gallery.getElementsByTagName("video").length;

        if (!!screenWidth && !!screenHeight && !!videoCount) {
          const rectWidth = 16;
          const rectHeight = 9;
          const r = largestRect(
              screenWidth,
              screenHeight,
              videoCount,
              rectWidth,
              rectHeight
          );

          gallery.style.setProperty("--width", r.width + "px");
          gallery.style.setProperty("--height", r.height + "px");
          gallery.style.setProperty("--cols", r.cols + "");
        }
      }
    },
    onButtonsFullscreen() {
      const elem = this.$refs.splVideo?.$el;

      if (elem && isFullscreen()) {
        document.exitFullscreen();
      } else {
        elem.requestFullscreen();
      }
    },
    onMouseEnter() {
      if (!this.isMobile()) {
        this.showControls = true;
      }
    },
    onMouseLeave() {
      if (!this.isMobile()) {
        this.showControls = false;
      }
    },
    getLoadingMessage () {
        if (isNoLocalTracks()) {
          return this.$vuetify.locale.t('$vuetify.user_video_local_viewer_no_media')
        } else {
          return this.$vuetify.locale.t('$vuetify.user_video_loading')
        }
    },
    onClick() {
      this.showControls =! this.showControls
    },
    onClickFromVideos() {
        if (this.shouldShowPresenter) {
          return
        }
        this.onClick()
    },
    presenterPaneSize() {
      if (this.chatStore.videoMiniaturesEnabled) {
        return this.getStored().presenterPane;
      } else {
        return 100
      }
    },
    miniaturesPaneSize() {
      if (this.chatStore.videoMiniaturesEnabled) {
        if (this.shouldShowPresenter) {
          return 100 - this.presenterPaneSize();
        } else {
          return 100;
        }
      } else {
        return 0
      }
    },
    shouldUseReverseOrder() {
      return this.videoIsHorizontal()
    },
    prepareForStore() {
      const ret = this.getStored();

      const paneSizes = this.$refs.splVideo.panes.map(i => i.size);
      if (this.shouldShowPresenter) {
        ret.presenterPane = paneSizes[this.shouldUseReverseOrder() ? paneSizes.length - 1 : 0];
      } else {
        ret.presenterPane = 0;
      }
      return ret
    },
    // returns json with sizes from localstore
    getStored() {
      const mbItem = localStorage.getItem(panelSizesKey);
      if (!mbItem) {
        return emptyStoredPanes();
      } else {
        return JSON.parse(mbItem);
      }
    },
    saveToStored(obj) {
      localStorage.setItem(panelSizesKey, JSON.stringify(obj));
    },
    onPanelResized() {
      this.$nextTick(() => {
        this.saveToStored(this.prepareForStore());
      })
    },
    onPanelAdd() {
      this.$nextTick(() => {
        const stored = this.getStored();
        this.restorePanelsSize(stored);
      })
    },
    onPanelRemove() {
      this.$nextTick(() => {
        const stored = this.getStored();
        this.restorePanelsSize(stored);
      })
    },
    restorePanelsSize(ret) {
      if (this.chatStore.videoMiniaturesEnabled) {
        if (this.shouldShowPresenter) {
          this.$refs.splVideo.panes[this.shouldUseReverseOrder() ? this.$refs.splVideo.panes.length - 1 : 0].size = ret.presenterPane;
        } else {
          this.$refs.splVideo.panes[this.shouldUseReverseOrder() ? this.$refs.splVideo.panes.length - 1 : 0].size = 100;
        }
      } else {
        this.$refs.splVideo.panes[this.shouldUseReverseOrder() ? this.$refs.splVideo.panes.length - 1 : 0].size = 100;
      }
    },
  },
  computed: {
    ...mapStores(useChatStore),
    splitpanesIsHorizontal() {
      return this.videoIsHorizontal() || this.videoIsGallery()
    },
    videoContainerClass() {
      if (this.videoIsHorizontal()) {
        return 'video-container-position-horizontal'
      } else if (this.videoIsGallery()) {
        return 'video-container-position-gallery'
      } else {
        return 'video-container-position-vertical'
      }
    },
    paneVideoContainerClass() {
      if (this.videoIsHorizontal() || this.videoIsGallery()) {
        return 'pane-videos-horizontal'
      } else if (this.videoIsVertical())  {
        return 'pane-videos-vertical'
      } else {
        return null;
      }
    },
    presenterPaneClass() {
      if (this.videoIsHorizontal()) {
        return 'pane-presenter-horizontal'
      } else {
        if (this.isMobile()) {
          return ['pane-presenter-vertical', 'pane-presenter-vertical-mobile']
        } else {
          return 'pane-presenter-vertical'
        }
      }
    },
    shouldShowPresenter() {
      return this.chatStore.presenterEnabled && !this.videoIsGallery()
    },
    presenterAvatarIsSet() {
      return hasLength(this.presenterData?.avatar);
    },
    presenterUserName() {
      return this.presenterData?.userName
    }
  },
  components: {
      UserVideoContextMenu,
      Splitpanes,
      Pane,
      VideoButtons,
      ChatVideoPresenter,
  },
  watch: {
    'chatStore.videoPosition': {
      handler: function (newValue, oldValue) {
        if (this.videoContainerDiv) {
          const videoIsHorizontal = this.videoIsHorizontalPlain(newValue);
          const videoIsGallery = this.videoIsGalleryPlain(newValue);
          for (const containerEl of this.videoContainerDiv.children) {
            this.setUserVideoWrapperClass(containerEl, videoIsHorizontal, videoIsGallery);
          }

          // we added it for the case when user switches from gallery to vertical or horizontal where presenter can be shown
          // test case
          // disable presenter
          // switch vertical and horizontal
          // the local video shouldn't disappear
          // thus because of this this.updatePresenter(data) doesn't have this.detachPresenter()
          if (this.canUsePresenterPlain(newValue) && this.chatStore.presenterEnabled) {
            this.$nextTick(() => {
              this.electNewPresenter();
            })
          }
          if (videoIsGallery) {
            setTimeout(()=>{
              this.recalculateLayout();
            }, 300)
          }
        }
      }
    },
    'chatStore.presenterEnabled': {
      handler: function (newValue, oldValue) {
        if (this.videoContainerDiv) {
          setSplitter(videoSplitpanesSelector, videoSplitterDisplayVarName, this.chatStore.videoMiniaturesEnabled);
          if (newValue) {
            this.$nextTick(()=>{ // needed because videoContainerDiv still not visible for attaching from livekit js
              this.electNewPresenter();
            })
          } else {
            this.detachPresenter();
          }
        }
      }
    },
    'chatStore.showDrawer': {
      handler: function (newValue, oldValue) {
        setTimeout(()=>{
          this.recalculateLayout();
        }, 300)
      }
    },
    'chatStore.localMicrophoneEnabled': {
      handler: function (newValue, oldValue) {
        const onlyOneLocalComponentWithAudio = this.onlyOneLocalTrackWith(this.room.localParticipant.identity)
        if (onlyOneLocalComponentWithAudio) {
          onlyOneLocalComponentWithAudio.doMuteAudio(!newValue, true);
        } else {
          // just for case
          this.chatStore.canShowMicrophoneButton = false;
        }
      },
    },
    'chatStore.localVideoEnabled': {
      handler: function (newValue, oldValue) {
        const onlyOneLocalComponentWithVideo = this.onlyOneLocalTrackWith(this.room.localParticipant.identity, true)
        if (onlyOneLocalComponentWithVideo) {
          onlyOneLocalComponentWithVideo.doMuteVideo(!newValue, true);
        } else {
          // just for case
          this.chatStore.canShowVideoButton = false;
        }
      },
    },
    'chatStore.videoMiniaturesEnabled': {
      handler: function (newValue, oldValue) {
        setSplitter(videoSplitpanesSelector, videoSplitterDisplayVarName, newValue);
      }
    }
  },
  created() {
    this.recalculateLayout = debounce(this.recalculateLayout);
  },
  async mounted() {
    this.initPositionAndPresenter();

    const videoMiniatures = getStoredVideoMiniatures();
    setSplitter(videoSplitpanesSelector, videoSplitterDisplayVarName, videoMiniatures);
    this.chatStore.videoMiniaturesEnabled = videoMiniatures;

    this.chatStore.setCallStateInCall();

    this.chatStore.initializingVideoCall = true;

    if (!this.isMobile()) {
      this.chatStore.showDrawerPrevious = this.chatStore.showDrawer;
      this.chatStore.showDrawer = false;
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
    bus.on(CHANGE_VIDEO_SOURCE, this.onChangeVideoSource);
    bus.on(PIN_VIDEO, this.onPinVideo);
    bus.on(UN_PIN_VIDEO, this.onUnpinVideo);

    this.chatStore.searchType = SEARCH_MODE_MESSAGES;

    window.addEventListener("resize", this.recalculateLayout);

    this.videoContainerDiv = document.getElementById("video-container");

    this.startRoom(enterResponse.data.token);
  },
  beforeUnmount() {
    axios.put(`/api/video/${this.chatId}/dial/exit`, null, {
        params: {
            tokenId: this.chatStore.videoTokenId
        }
    });

    this.detachPresenter();

    this.stopLocalTracks().finally(()=>{
      this.stopRoom();
    })

    console.log("Cleaning videoContainerDiv");
    this.videoContainerDiv = null;
    this.inRestarting = false;

    this.chatStore.canShowMicrophoneButton = false;

    this.localTrackDrawn = false;
    this.localTrackCreatedAndPublished = false;
    this.finishedConnectingToRoom = false;
    this.chatStore.initializingVideoCall = false; // reset

    if (!this.isMobile()) {
      this.chatStore.showDrawer = this.chatStore.showDrawerPrevious;
      this.chatStore.showDrawerPrevious = false;
    }

    this.chatStore.videoChatUsersCount = 0;
    this.chatStore.showRecordStartButton = false;
    this.chatStore.initializingStaringVideoRecord = false;
    this.chatStore.initializingStoppingVideoRecord = false;

    this.chatStore.pinnedTrackSid = null;

    this.chatStore.videoTokenId = null;

    this.chatStore.setCallStateReady();

    window.removeEventListener("resize", this.recalculateLayout);

    bus.off(ADD_VIDEO_SOURCE, this.onAddVideoSource);
    bus.off(ADD_SCREEN_SOURCE, this.onAddScreenSource);
    bus.off(REQUEST_CHANGE_VIDEO_PARAMETERS, this.tryRestartVideoDevice);
    bus.off(CHANGE_VIDEO_SOURCE, this.onChangeVideoSource);
    bus.off(PIN_VIDEO, this.onPinVideo);
    bus.off(UN_PIN_VIDEO, this.onUnpinVideo);
  },
}

</script>

<style lang="stylus" scoped>
#video-container {
  display: flex;
  //scroll-snap-align width
  //scroll-padding 0
  //object-fit: contain;
  //box-sizing: border-box
}

.video-container-position-horizontal {
  height 100%
  width 100%
  flex-direction: row;
  overflow-x: auto;
  overflow-y: hidden;
  // scrollbar-width: none;
  background black
}

.video-container-position-vertical {
  height 100%
  width 100%
  overflow-y: auto;
  background black
  flex-direction: column;
}

.video-container-position-gallery {
  height: 100%;
  width: 100%;

  align-items: center;
  justify-content: center;
  align-content: baseline;
  overflow-y: auto;

  display: flex
  flex-wrap: wrap
  // max-width: calc(var(--width) * var(--cols))
  background-color: black;
}

// need to center the nested video buttons
.pane-videos-horizontal {
  display: flex;
  justify-content: center;
  position: relative // for mobile
}

.pane-videos-vertical {
  display: flex;
  align-items center
  justify-content center
  position relative // for mobile
}

</style>

<style lang="stylus">
// applied from js, so it shouldn't be changed, so without scoped
.video-component-wrapper-position-horizontal {
  display contents
}

// can be both horizontal and vertical depending on the video position
#video-splitpanes .splitpanes__splitter {
  display: var(--splitter-h-display);
}


</style>
