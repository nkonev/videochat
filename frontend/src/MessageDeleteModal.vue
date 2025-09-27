<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480" persistent>
            <v-card :title="$vuetify.locale.t('$vuetify.delete_message_title', messageId)" :disabled="loading">
                <v-progress-linear
                  :active="loading"
                  :indeterminate="loading"
                  absolute
                  bottom
                  color="primary"
                ></v-progress-linear>

                <v-card-text v-html="$vuetify.locale.t('$vuetify.delete_message_text')"></v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-spacer></v-spacer>
                    <v-btn color="red" variant="flat" @click="onDelete(true)">{{ $vuetify.locale.t('$vuetify.message_delete_with_attached_files') }}</v-btn>
                    <v-btn color="red" variant="flat" @click="onDelete()">{{ $vuetify.locale.t('$vuetify.delete_btn') }}</v-btn>
                    <v-btn variant="outlined" @click="hideModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_MESSAGE_DELETE_MODAL} from "./bus/bus";
    import axios from "axios";

    export default {
        data () {
            return {
                show: false,
                loading: false,

                messageId: null,
                fileItemUuid: null,
            }
        },
        computed: {
          chatId() {
            return this.$route.params.id
          },
        },
        methods: {
            showModal(newData) {
                this.$data.messageId = newData.messageId;
                this.$data.fileItemUuid = newData.fileItemUuid;

                this.$data.show = true;
            },
            hideModal() {
                this.$data.show = false;
                this.$data.loading = false;

                this.$data.messageId = null;
                this.$data.fileItemUuid = null;
            },
            async onDelete(deleteAttachedFiles) {
              this.loading = true;

              const promises = [];
              if (deleteAttachedFiles) {
                promises.push(axios.delete(`/api/storage/${this.chatId}/file`, {
                  data: {
                    fileItemUuid: this.$data.fileItemUuid
                  }
                }))
              }
              promises.push(axios.delete(`/api/chat/${this.chatId}/message/${this.$data.messageId}`));
              await Promise.all(promises);
              this.hideModal();

              this.loading = false;
            },
        },
        mounted() {
            bus.on(OPEN_MESSAGE_DELETE_MODAL, this.showModal);
        },
        beforeUnmount() {
            bus.off(OPEN_MESSAGE_DELETE_MODAL, this.showModal);
        },
    }
</script>
