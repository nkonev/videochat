<template>
    <v-dialog v-model="show" @click:outside="closeModal()" fullscreen>
        <v-card>
            <v-toolbar
                dark
                color="primary"
            >
                <v-btn
                    icon
                    dark
                    @click="closeModal()"
                >
                    <v-icon>mdi-close</v-icon>
                </v-btn>
                <v-toolbar-title>Editing message</v-toolbar-title>
            </v-toolbar>
            <div class="message-edit-dialog">
                <MessageEdit :chatId="chatId" :canBroadcast="canBroadcast"/>
            </div>
        </v-card>
    </v-dialog>
</template>

<script>
import bus, {OPEN_MESSAGE_DIALOG} from "@/bus";
import MessageEdit from "@/MessageEdit";

    export default {
        data() {
            return {
                show: false,
                canBroadcast: false,
            }
        },
        methods: {
            showModal(canBroadcast) {
                this.show = true;
                this.canBroadcast = canBroadcast;
            },
            closeModal() {
                this.show = false;
                this.canBroadcast = false;
            }
        },
        components: {
            MessageEdit
        },
        computed: {
            chatId() {
                return this.$route.params.id
            },
        },
        created() {
            bus.$on(OPEN_MESSAGE_DIALOG, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_MESSAGE_DIALOG, this.showModal);
        }
    }
</script>

<style lang="stylus">
    .message-edit-dialog {
        // TODO format it
        //display flex
        //align-items: stretch
        //height 100%
        //position: relative;
        //align-self: stretch
    }
</style>