import {
    getStoredPresenter,
    getStoredVideoPosition,
    VIDEO_POSITION_GALLERY,
    VIDEO_POSITION_HORIZONTAL,
    VIDEO_POSITION_VERTICAL
} from "@/store/localStore";
import {videochat_name} from "@/router/routes";

export default () => {
    return {
        methods: {
            videoIsHorizontalPlain(value) {
                return value === VIDEO_POSITION_HORIZONTAL;
            },
            videoIsHorizontal() {
              const stored = this.chatStore.videoPosition;
              return this.videoIsHorizontalPlain(stored);
            },
            videoIsVertical() {
                return this.chatStore.videoPosition === VIDEO_POSITION_VERTICAL;
            },
            videoIsGalleryPlain(value) {
                return value === VIDEO_POSITION_GALLERY;
            },
            videoIsGallery() {
                const stored = this.chatStore.videoPosition;
                return this.videoIsGalleryPlain(stored);
            },
            isPresenterEnabled() {
                return this.videoIsHorizontal() || this.videoIsVertical()
            },
            isVideoRoute() {
              return this.$route.name == videochat_name
            },

            shouldShowChatList() {
              if (this.isMobile()) {
                return false;
              }
              return !this.isVideoRoute();
            },
            initPositionAndPresenter() {
                this.chatStore.videoPosition = getStoredVideoPosition();
                this.chatStore.presenterEnabled = getStoredPresenter();
            },
        }
    }
}
