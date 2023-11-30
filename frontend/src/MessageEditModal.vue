<template>
    <!-- Used in Mobile Android -->
    <v-dialog v-model="show" fullscreen persistent>
        <v-card>
            <v-toolbar
                dark
                color="indigo"
                height="56"
            >
                <span class="d-flex mx-2">
                    <v-btn
                      icon
                      dark
                      @click="closeModal()"
                    >
                        <v-icon>mdi-close</v-icon>
                    </v-btn>
                </span>
                <span class="d-flex flex-grow-1">
                  <v-toolbar-title>{{ isNew ? $vuetify.locale.t('$vuetify.message_creating') : $vuetify.locale.t('$vuetify.message_editing')}}</v-toolbar-title>
                </span>
                <span class="d-flex mx-2">
                  <v-btn
                    icon
                    dark
                    @click="onPaste()"
                  >
                    <v-icon>mdi-content-paste</v-icon>
                  </v-btn>
                  </span>
            </v-toolbar>
            <!-- We cannot use it in style tag because it is loading too late and doesn't have an effect -->
            <div class="message-edit-dialog" :style="heightWithoutAppBar">
                <MessageEdit :chatId="chatId"/>
            </div>
        </v-card>
    </v-dialog>
</template>

<script>
import bus, {
  ADD_MESSAGE_TEXT,
  CLOSE_EDIT_MESSAGE,
  OPEN_EDIT_MESSAGE,
  SET_EDIT_MESSAGE_MODAL,
} from "./bus/bus";
    import MessageEdit from "@/MessageEdit.vue";
    import heightMixin from "@/mixins/heightMixin";
import {hasLength} from "@/utils";

    export default {
        data() {
            return {
                show: false,
                messageId: null,
                showPaste: false,
            }
        },
        mixins:[
          heightMixin(),
        ],
        methods: {
            showModal(dto) {
                this.show = true;
                this.messageId = dto?.id;
                this.$nextTick(()=>{
                    bus.emit(SET_EDIT_MESSAGE_MODAL, {dto, isNew: this.isNew});
                });
                this.setShowPaste();
            },
            closeModal() {
                this.show = false;
                this.messageId = null;
                this.showPaste = false;
            },
            setShowPaste() {
              navigator.clipboard.readText().then((text)=>{
                const trimmedText = text.trim();
                this.showPaste = hasLength(trimmedText);
              })
            },
            onPaste() {
              this.$nextTick(()=> {
                navigator.clipboard.readText().then((text)=>{
                  bus.emit(ADD_MESSAGE_TEXT, text);
                })
              })
            },
        },
        watch: {
            show(newValue) {
                if (!newValue) {
                    this.closeModal();
                }
            }
        },
        components: {
            MessageEdit,
        },
        computed: {
            chatId() {
                return this.$route.params.id
            },
            isNew() {
                return !this.messageId;
            },
        },
        mounted() {
            bus.on(OPEN_EDIT_MESSAGE, this.showModal);
            bus.on(CLOSE_EDIT_MESSAGE, this.closeModal);
        },
        beforeUnmount() {
            bus.off(OPEN_EDIT_MESSAGE, this.showModal);
            bus.off(CLOSE_EDIT_MESSAGE, this.closeModal);
        },
    }
</script>

<style scoped lang="stylus">
@import "constants.styl"

</style>
