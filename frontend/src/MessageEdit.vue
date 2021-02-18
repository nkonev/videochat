<template>
    <v-container id="sendButtonContainer" class="py-0 px-1 d-flex flex-column" fluid style="height: 100%">
            <div class="mb-0 mt-0 pb-0 pt-0 text--disabled caption user-typing">
                <template v-if="writingUsers.length">
                    {{writingUsers.map(v=>v.login).join(', ')}} is writing...
                </template>
            </div>
            <quill-editor
                v-model="editMessageDto.text"
                :options="editorOption"
                @keyup.native.ctrl.enter="sendMessageToChat" @keyup.native.esc="resetInput"
            />
            <div id="custom-toolbar">
                <div class="custom-toolbar-format">
                    <button class="ql-bold"></button>
                    <button class="ql-italic"></button>
                    <button class="ql-underline"></button>
                    <button class="ql-strike"></button>
                    <select class="ql-color"></select>
                    <select class="ql-background"></select>
                    <button class="ql-clean"></button>
                </div>
                <div class="custom-toolbar-send">
                    <v-btn color="primary" @click="sendMessageToChat" small><v-icon color="white">mdi-send</v-icon></v-btn>
                </div>
            </div>

    </v-container>
</template>

<script>
    import axios from "axios";
    import bus, {SET_EDIT_MESSAGE, USER_TYPING} from "./bus";
    import debounce from "lodash/debounce";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import 'quill/dist/quill.core.css'
    import 'quill/dist/quill.snow.css'

    import { quillEditor } from 'vue-quill-editor'

    const dtoFactory = ()=>{
        return {
            id: null,
            text: "",
        }
    };

    let timerId;

    export default {
        props:['chatId'],
        data() {
            return {
                editMessageDto: dtoFactory(),
                writingUsers: [],

                editorOption: {
                    // Some Quill options...
                    modules: {
                        // https://quilljs.com/docs/modules/toolbar/
                        toolbar: '#custom-toolbar',
                    },
                    placeholder: 'Press Ctrl + Enter to send, Esc to clear'
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
            bus.$off(SET_EDIT_MESSAGE, this.onSetMessage);
            bus.$off(USER_TYPING, this.onUserTyping);
            clearInterval(timerId);
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
$mobileWidth = 800px

#sendButtonContainer {
    min-height 25%

    .user-typing {
        height 14px
        max-height 14px
        min-height 14px
        font-size 10px !important
        line-height 14px !important
        padding-left 0.2em
    }
}

.quill-editor {
    height 100%
    overflow-y auto
}
.ql-toolbar.ql-snow {
    padding 4px
}
.ql-snow .ql-picker.ql-expanded .ql-picker-options {
    top: unset
    bottom 100%
}
@media screen and (max-width: $mobileWidth) {
    .ql-editor {
        padding-left 4px
        padding-right 4px
        padding-top 2px
        padding-bottom 2px
    }

    .ql-toolbar.ql-snow {
        padding 2px
    }
}
//.ql-container {
//    height calc(100% - 16px)
//}
.ql-toolbar {
    display: inline-flex;
    //align-items center
}
#custom-toolbar {
    display: flex;
    align-items: center
    justify-content: space-between
    border-top-width: 0
    border-bottom-style dashed
    border-left-style dashed
    border-right-style dashed
}
@media screen and (max-width: $mobileWidth) {
    #custom-toolbar {
        border-width: 0
    }
    //border-left-width: 0
    //border-right-width: 0
}

.custom-toolbar-format {
    display: flex;
    flex-grow: 0
}
.custom-toolbar-send {
    display: flex;
    flex-grow: 10
    justify-content flex-end
}
</style>