import {getStoredVideoPosition, VIDEO_POSITION_AUTO, VIDEO_POSITION_ON_THE_TOP} from "@/store/localStore";
import {videochat_name} from "@/router/routes";

export default () => {
    return {
        methods: {
            videoIsOnTop() {
              const stored = getStoredVideoPosition();
              if (stored == VIDEO_POSITION_AUTO) {
                return this.isMobile()
              } else {
                return getStoredVideoPosition() == VIDEO_POSITION_ON_THE_TOP;
              }
            },

            videoIsAtSide() {
              return !this.videoIsOnTop();
            },

            isVideoRoute() {
              return this.$route.name == videochat_name
            },

            shouldShowChatList() {
              if (this.isMobile()) {
                return false;
              }
              if (this.isVideoRoute()) {
                if (this.videoIsAtSide()) {
                  return false
                }
              }
              return true;
            },

        }
    }
}
