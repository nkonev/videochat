<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="400" persistent>
            <v-card>
                <v-card-title>Upload files</v-card-title>

                <v-card-text>Choose files</v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn class="mr-4" @click="hideModal()">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {OPEN_FILE_UPLOAD_MODAL, CLOSE_FILE_UPLOAD_MODAL} from "./bus";

export default {
    data () {
        return {
            show: false,
        }
    },
    methods: {
        showModal(chatId) {
            console.info("ChatId", chatId);
            this.$data.show = true;
        },
        hideModal() {
            this.$data.show = false;
        },
    },
    created() {
        bus.$on(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.$on(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
    },
    destroyed() {
        bus.$off(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.$off(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
    },
}
</script>