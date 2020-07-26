<template>
    <v-container id="sendButtonContainer">
        <v-row no-gutters dense>
            <v-col cols="12">
                <v-text-field dense label="Send a message" @keyup.native.enter="sendMessageToChat" v-model="editMessageDto.text" :append-outer-icon="'mdi-send'" @click:append-outer="sendMessageToChat"></v-text-field>
            </v-col>
        </v-row>
    </v-container>
</template>

<script>
    import axios from "axios";
    import bus, {MESSAGE_ADD, SET_EDIT_MESSAGE} from "./bus";

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
            }
        },
        methods: {
            sendMessageToChat() {
                if (this.editMessageDto.text && this.editMessageDto.text !== "") {
                    (this.editMessageDto.id ? axios.put(`/api/chat/`+this.chatId+'/message', this.editMessageDto) : axios.post(`/api/chat/`+this.chatId+'/message', this.editMessageDto)).then(response => {
                        console.log("Resetting text input");
                        this.editMessageDto.text = "";
                        this.editMessageDto.id = null;
                    })
                }
            },
            onSetMessage(dto) {
                this.editMessageDto = dto;
            },
        },
        mounted() {
            bus.$on(SET_EDIT_MESSAGE, this.onSetMessage);
        },
        beforeDestroy() {
            bus.$off(SET_EDIT_MESSAGE, this.onSetMessage);
        }
    }
</script>