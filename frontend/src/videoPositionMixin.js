import {getStoredVideoPosition, VIDEO_POSITION_AUTO, VIDEO_POSITION_ON_THE_TOP} from "@/localStore";

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

        }
    }
}
