<template>
    <v-container id="sendButtonContainer" class="pa-0 d-flex flex-row" style="height: 100%">
        <v-container class="ma-0 pa-0">
            <div class="mb-0 mt-0 pb-0 pt-0 text--disabled caption" style="height: 2em">
                <template v-if="writingUsers.length">
                    {{writingUsers.map(v=>v.login).join(', ')}} is writing...
                </template>
            </div>
            <quill-editor
                ref="myQuillEditor"
                v-model="editMessageDto.text"
                :options="editorOption"
            />
        </v-container>
        <v-btn class="ml-1 mt-6" color="primary"><v-icon>mdi-send</v-icon></v-btn>
    </v-container>
</template>

<script>
    import axios from "axios";
    import bus, {SET_EDIT_MESSAGE, USER_TYPING} from "./bus";
    import debounce from "lodash/debounce";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import {getHeight} from "./utils"
    import 'quill/dist/quill.core.css'
    import 'quill/dist/quill.snow.css'
    import 'quill/dist/quill.bubble.css'

    import { quillEditor } from 'vue-quill-editor'

    const dtoFactory = ()=>{
        return {
            id: null,
            text: "",
        }
    };

    let timerId;


    // https://quilljs.com/docs/modules/toolbar/
    const toolbarOptions = [
        ['bold', 'italic', 'underline', 'strike'],        // toggled buttons
        [{ 'color': [] }, { 'background': [] }],          // dropdown with defaults from theme
        [{ 'align': [] }],
        ['clean']                                         // remove formatting button
    ];

    export default {
        props:['chatId'],
        data() {
            return {
                editMessageDto: dtoFactory(),
                writingUsers: [],

                editorOption: {
                    // Some Quill options...
                    modules: {
                        toolbar: toolbarOptions,
                    }
                }
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
            calcTextareaHeight() {
                return getHeight("sendButtonContainer", (v) => v - 40 + "px", '100px')
            }
        },
        computed: {
            ...mapGetters({currentUser: GET_USER})
        },
        mounted() {
            bus.$on(SET_EDIT_MESSAGE, this.onSetMessage);
            timerId = setInterval(()=>{
                const curr = + new Date();
                this.writingUsers = this.writingUsers.filter(value => (value.timestamp + 1*1000) > curr);
            }, 500);
            bus.$on(USER_TYPING, this.onUserTyping);
        },
        beforeDestroy() {
            clearInterval(timerId);
            bus.$off(SET_EDIT_MESSAGE, this.onSetMessage);
            bus.$off(USER_TYPING, this.onUserTyping);
        },
        created(){
            this.notifyAboutTyping = debounce(this.notifyAboutTyping, 500, {leading:true, trailing:false});
        },
        watch: {
            'editMessageDto.text': {
                handler: function (newValue, oldValue) {
                    if (newValue && newValue != "")
                    this.notifyAboutTyping();
                },
            }
        },
        components: {
            quillEditor
        }
    }
</script>

<style lang="stylus">
//#sendButtonContainer {
//
//}

.quill-editor {
    height calc(100% - 60px)
}
.ql-container {
    height calc(100% - 10px)
}
</style>