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
                  <v-toolbar-title>{{ !isEditing ? $vuetify.locale.t('$vuetify.message_creating') : $vuetify.locale.t('$vuetify.message_editing')}}</v-toolbar-title>
                </span>
            </v-toolbar>
            <!-- We cannot use it in style tag because it is loading too late and doesn't have an effect -->
            <div class="message-edit-dialog" :style="heightWithoutAppBar">
                <MessageEdit :chatId="chatId" @setEditingTitle="onSetEditingTitle"/>
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

    export default {
        data() {
            return {
                show: false,
                isEditing: false,
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
                this.isEditing = false;
            },
            onSetEditingTitle() {
                this.isEditing = true;
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
