import {SET_SHOW_MICROPHONE_OFF_BUTTON, SET_SHOW_MICROPHONE_ON_BUTTON} from "@/store";

export default () => {
    return {
        methods: {
            refreshLocalMutedInAppBar(muted) {
                if (muted) {
                    this.$store.commit(SET_SHOW_MICROPHONE_ON_BUTTON, false);
                    this.$store.commit(SET_SHOW_MICROPHONE_OFF_BUTTON, true);
                } else {
                    this.$store.commit(SET_SHOW_MICROPHONE_ON_BUTTON, true);
                    this.$store.commit(SET_SHOW_MICROPHONE_OFF_BUTTON, false);
                }
            },
        }
    }
}
