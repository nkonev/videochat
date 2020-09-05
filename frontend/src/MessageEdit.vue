<template>
    <v-container id="sendButtonContainer">
        <v-row no-gutters dense>
            <v-col cols="12">
                <v-col class="mb-0 mt-0 pb-0 pt-0 text--disabled caption" style="height: 1em">
                    <template v-if="writingUsers.length">
                        {{writingUsers.map(v=>v.login).join(', ')}} is writing...
                    </template>
                </v-col>
                <v-textarea solo dense label="Send a message" @keyup.ctrl.enter="sendMessageToChat" @keyup.esc="resetInput" v-model="editMessageDto.text" :append-outer-icon="'mdi-send'" @click:append-outer="sendMessageToChat"></v-textarea>
            </v-col>
        </v-row>
    </v-container>
</template>

<script>
    import axios from "axios";
    import bus, {SET_EDIT_MESSAGE, USER_TYPING} from "./bus";
    import throttle from "lodash/throttle";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";

    const dtoFactory = ()=>{
        return {
            id: null,
            text: "",
        }
    };

    export default {
        props:['chatId'],
        data() {
            return {
                editMessageDto: dtoFactory(),
                writingUsers: []
            }
        },
        methods: {
            sendMessageToChat() {
                if (this.editMessageDto.text && this.editMessageDto.text !== "") {
                    (this.editMessageDto.id ? axios.put(`/api/chat/`+this.chatId+'/message', this.editMessageDto) : axios.post(`/api/chat/`+this.chatId+'/message', this.editMessageDto)).then(response => {
                        this.resetInput();
                    })
                }
            },
            resetInput() {
              console.log("Resetting text input");
              this.editMessageDto.text = "";
              this.editMessageDto.id = null;
            },
            onSetMessage(dto) {
                this.editMessageDto = dto;
            },
            notifyAboutTyping() {
                axios.put(`/api/chat/`+this.chatId+'/typing')
            },
            onUserTyping(data) {
                console.log("OnUserTyping", data);

                if (this.currentUser.id == data.participantId) {
                    console.log("Skipping myself typing notifications");
                    return;
                }

                const idx = this.writingUsers.findIndex(value => value.login === data.login);
                if (idx !== -1) {
                    this.writingUsers[idx].timestamp = + new Date();
                } else {
                    this.writingUsers.push({timestamp: +new Date(), login: data.login})
                }
            },
        },
        computed: {
            ...mapGetters({currentUser: GET_USER})
        },
        mounted() {
            bus.$on(SET_EDIT_MESSAGE, this.onSetMessage);
            setInterval(()=>{
                const curr = + new Date();
                this.writingUsers = this.writingUsers.filter(value => (value.timestamp + 1*1000) > curr);
            }, 500);
            bus.$on(USER_TYPING, this.onUserTyping);
        },
        beforeDestroy() {
            bus.$off(SET_EDIT_MESSAGE, this.onSetMessage);
            bus.$off(USER_TYPING, this.onUserTyping);
        },
        created(){
            this.notifyAboutTyping = throttle(this.notifyAboutTyping, 500);
        },
        watch: {
            'editMessageDto.text': {
                handler: function (newValue, oldValue) {
                    if (newValue && newValue != "")
                    this.notifyAboutTyping();
                },
            }
        },
    }
</script>