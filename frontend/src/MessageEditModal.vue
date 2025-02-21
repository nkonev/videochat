<template>
    <!-- Used only in Mobile.
     eager need in order to invoke mount() in MessageEdit
     in order to set chatStore.isMessageEditing
     -->
    <v-dialog v-model="show" fullscreen :eager="isMobile()">
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
                  <v-toolbar-title>{{ !chatStore.isMessageEditing ? $vuetify.locale.t('$vuetify.message_creating') : $vuetify.locale.t('$vuetify.message_editing')}}</v-toolbar-title>
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
      CLOSE_EDIT_MESSAGE,
      OPEN_EDIT_MESSAGE,
      SET_EDIT_MESSAGE_MODAL,
    } from "./bus/bus";
    import MessageEdit from "@/MessageEdit.vue";
    import heightMixin from "@/mixins/heightMixin";
    import {useChatStore} from "@/store/chatStore";
    import {mapStores} from "pinia";

    export default {
        data() {
            return {
                show: false,
            }
        },
        mixins:[
          heightMixin(),
        ],
        methods: {
            showModal({dto, actionType}) {
                this.show = true;
                this.$nextTick(()=>{
                    bus.emit(SET_EDIT_MESSAGE_MODAL, {dto, actionType});
                });
            },
            closeModal() {
                this.show = false;
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
            ...mapStores(useChatStore),
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
