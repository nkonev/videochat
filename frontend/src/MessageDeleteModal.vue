<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="520" persistent>
            <v-card :title="$vuetify.locale.t('$vuetify.delete_message_title', messageId)" :disabled="loading">
                <v-progress-linear
                  :active="loading"
                  :indeterminate="loading"
                  absolute
                  bottom
                  color="primary"
                ></v-progress-linear>

                <v-card-text class="pb-3">
                  <p v-html="$vuetify.locale.t('$vuetify.delete_message_text')"></p>
                  <div class="message-wrapper pt-2">
                    <div class="message-text">
                      {{ messageText }}
                    </div>
                  </div>

                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-spacer></v-spacer>
                    <v-btn color="red" variant="flat" @click="onDelete()">{{ $vuetify.locale.t('$vuetify.delete_btn') }}</v-btn>
                    <v-btn v-if="hasLength(fileItemUuid)" color="red" variant="outlined" @click="onDelete(true)" :title="$vuetify.locale.t('$vuetify.message_delete_with_attached_files_full')">{{ $vuetify.locale.t('$vuetify.message_delete_with_attached_files') }}</v-btn>
                    <v-btn variant="outlined" @click="hideModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_MESSAGE_DELETE_MODAL} from "./bus/bus";
    import axios from "axios";
    import {hasLength} from "@/utils.js";

    export default {
        data () {
            return {
                show: false,
                loading: false,

                messageId: null,
                fileItemUuid: null,
                messageText: null,
            }
        },
        computed: {
          chatId() {
            return this.$route.params.id
          },
        },
        methods: {
            hasLength,
            showModal(newData) {
                this.$data.messageId = newData.messageId;
                this.$data.fileItemUuid = newData.fileItemUuid;

                axios.put('/api/chat/public/preview-without-html', {text: newData.messageText}).then(({data}) => {
                  this.$data.messageText = data.text;
                })

                this.$data.show = true;
            },
            hideModal() {
                this.$data.show = false;
                this.$data.loading = false;

                this.$data.messageId = null;
                this.$data.fileItemUuid = null;
                this.$data.messageText = null;
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

<style scoped lang="stylus">
.message-wrapper {
  display: block
}
.message-text {
  color: gray;
  white-space: nowrap;
  text-overflow: ellipsis;
  overflow: hidden;
}
.message-text:empty::before {
  content:"\200B";
}
</style>
