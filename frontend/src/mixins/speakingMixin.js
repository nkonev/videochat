export default () => {
    return {
        data() {
            return {
                speaking: false,
                speakingTimer: null,
            }
        },
        methods: {
            setSpeaking(speaking) {
                this.speaking = speaking;
            },
            setSpeakingInternal(timeout) {
                this.speaking = true;
                this.speakingTimer = setTimeout(() => {
                    this.speaking = false;
                    this.speakingTimer = null;
                }, timeout);
            },
            setSpeakingWithTimeout(timeout) {
                if (!this.speakingTimer) {
                    this.setSpeakingInternal(timeout);
                } else {
                    clearTimeout(this.speakingTimer);
                    this.setSpeakingInternal(timeout);
                }
            },
            setSpeakingWithDefaultTimeout() {
                this.setSpeakingWithTimeout(1000);
            },
        },
    }
}
