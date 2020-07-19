<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title>Delete chat #{{deleteChatDto.id}}</v-card-title>

                <v-card-text>Are you sure to delete chat '{{deleteChatDto.name}}' ?</v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn color="error" class="mr-4" @click="deleteChat">Delete</v-btn>
                    <v-btn class="mr-4" @click="show=false">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";
    import bus, {OPEN_CHAT_DELETE} from "./bus";

    export default {
        data () {
            return {
                show: false,
                deleteChatDto: {},
            }
        },
        methods: {
            showModal(chat) {
                this.$data.show = true;
                this.deleteChatDto = chat;
            },
            deleteChat() {
                axios.delete(`/api/chat/${this.deleteChatDto.id}`)
                    .then(() => {
                        this.show=false;
                    })
            },
        },
        created() {
            bus.$on(OPEN_CHAT_DELETE, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_CHAT_DELETE, this.showModal);
        },
    }
</script>