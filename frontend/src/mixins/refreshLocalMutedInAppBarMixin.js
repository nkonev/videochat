
export default () => {
    return {
        methods: {
            refreshLocalMutedInAppBar(muted) {
                if (muted) {
                    this.chatStore.showMicrophoneOnButton = false;
                    this.chatStore.showMicrophoneOffButton = true;
                } else {
                    this.chatStore.showMicrophoneOnButton = true;
                    this.chatStore.showMicrophoneOffButton = false;
                }
            },
        }
    }
}
