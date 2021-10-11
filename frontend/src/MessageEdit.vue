<template>
    <v-container id="sendButtonContainer" class="py-0 px-1 pb-1 d-flex flex-column" fluid style="height: 100%">
            <quill-editor
                :editorOptions="editorOption"
                @keyup.native.ctrl.enter="sendMessageToChat"
                @keyup.native.esc="resetInput"
                ref="quillEditorInstance"
                v-model="editMessageDto.text"
            />
            <div id="custom-toolbar">
                <div class="custom-toolbar-format">
                    <button class="ql-bold"></button>
                    <button class="ql-italic"></button>
                    <button class="ql-underline"></button>
                    <button class="ql-strike"></button>
                    <select class="ql-color"></select>
                    <select class="ql-background"></select>
                    <button class="ql-link"></button>
                    <button class="ql-clean"></button>
                </div>
                <div class="custom-toolbar-send">
                    <v-btn v-if="!this.editMessageDto.fileItemUuid" icon tile :class="$vuetify.breakpoint.smAndUp ? 'mr-4' : ''" @click="openFileUpload()"><v-icon color="primary">mdi-file-upload</v-icon></v-btn>
                    <template v-if="this.editMessageDto.fileItemUuid">
                        <v-badge
                            :value="fileCount"
                            :content="fileCount"
                            color="green"
                            overlap
                            left
                        >
                            <v-btn icon tile :class="$vuetify.breakpoint.smAndUp ? 'mr-4' : ''" @click="onFilesClicked()"><v-icon>mdi-file-document-multiple</v-icon></v-btn>
                        </v-badge>
                    </template>
                    <v-btn icon tile :class="$vuetify.breakpoint.smAndUp ? 'mr-4' : 'mr-2'" @click="resetInput()"><v-icon>mdi-delete</v-icon></v-btn>
                    <v-switch v-if="canBroadcast && $vuetify.breakpoint.smAndUp" dense hide-details class="ma-0 mr-4" v-model="sendBroadcast"
                        :label="$vuetify.breakpoint.smAndUp ? `Broadcast` : null"
                    ></v-switch>
                    <v-btn color="primary" @click="sendMessageToChat" small><v-icon color="white">mdi-send</v-icon></v-btn>
                </div>
            </div>
            <v-tooltip v-if="writingUsers.length || broadcastMessage" :activator="'#sendButtonContainer'" top v-model="showTooltip" :key="tooltipKey">
                <span v-if="!broadcastMessage">{{writingUsers.map(v=>v.login).join(', ')}} is writing...</span>
                <span v-else>{{broadcastMessage}}</span>
            </v-tooltip>

    </v-container>
</template>

<script>
    import axios from "axios";
    import bus, {
        MESSAGE_BROADCAST,
        OPEN_FILE_UPLOAD_MODAL,
        OPEN_VIEW_FILES_DIALOG,
        SET_EDIT_MESSAGE, SET_FILE_ITEM_UUID,
        USER_TYPING
    } from "./bus";
    import debounce from "lodash/debounce";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import 'quill/dist/quill.core.css'
    import 'quill/dist/quill.snow.css'
    import Editor from "./Editor";

    const dtoFactory = ()=>{
        return {
            id: null,
            text: "",
        }
    };

    let timerId;

    export default {
        props:['chatId', 'canBroadcast'],
        data() {
            return {
                editMessageDto: dtoFactory(),
                writingUsers: [],

                editorOption: {
                    theme: 'snow',
                    // Some Quill options...
                    modules: {
                        // https://quilljs.com/docs/modules/toolbar/
                        toolbar: '#custom-toolbar',
                    },
                    placeholder: 'Press Ctrl + Enter to send, Esc to clear'
                },
                showTooltip: true,
                sendBroadcast: false,
                broadcastMessage: null,
                tooltipKey: 0,
                fileCount: null,
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
              this.editMessageDto.fileItemUuid = null;
              this.fileCount = null;
            },
            onSetMessage(dto) {
                this.editMessageDto = dto;
                if (this.editMessageDto.fileItemUuid) {
                    axios.get(`/api/storage/${this.chatId}/file/count/${this.editMessageDto.fileItemUuid}`)
                        .then((response) => {
                            this.onFileItemUuid({fileItemUuid: this.editMessageDto.fileItemUuid, count: response.data.count})
                        });
                }
            },
            notifyAboutBroadcast(clear) {
                if (clear) {
                    axios.put(`/api/chat/`+this.chatId+'/broadcast', {text: null});
                } else {
                    axios.put(`/api/chat/`+this.chatId+'/broadcast', {text: this.editMessageDto.text});
                }
            },
            notifyAboutTyping() {
                axios.put(`/api/chat/` + this.chatId + '/typing');
            },
            sendNotification() {
                if (this.sendBroadcast) {
                    this.notifyAboutBroadcast();
                } else {
                    this.notifyAboutTyping();
                }
            },

            onUserTyping(data) {
                console.debug("OnUserTyping", data);

                if (!this.sendBroadcast && this.currentUser.id == data.participantId) {
                    console.log("Skipping myself typing notifications");
                    return;
                }
                this.showTooltip = true;

                const idx = this.writingUsers.findIndex(value => value.login === data.login);
                if (idx !== -1) {
                    this.writingUsers[idx].timestamp = + new Date();
                } else {
                    this.writingUsers.push({timestamp: +new Date(), login: data.login})
                }
            },
            onUserBroadcast(dto) {
                console.log("onUserBroadcast", dto);
                const stripped = dto.text;
                if (stripped && stripped.length > 0) {
                    this.tooltipKey++;
                    this.showTooltip = true;
                    this.broadcastMessage = dto.text;
                } else {
                    this.broadcastMessage = null;
                }
            },
            updateModel(html) {
                this.editMessageDto.text = html;
            },
            openFileUpload() {
                bus.$emit(OPEN_FILE_UPLOAD_MODAL, this.editMessageDto.fileItemUuid, true, false);
            },
            onFilesClicked() {
                bus.$emit(OPEN_VIEW_FILES_DIALOG, {chatId: this.chatId, fileItemUuid: this.editMessageDto.fileItemUuid, messageEditing: true});
            },
            onFileItemUuid({fileItemUuid, count}) {
                this.editMessageDto.fileItemUuid = fileItemUuid;
                this.fileCount = count;
                if (this.fileCount === 0) {
                    this.editMessageDto.fileItemUuid = null;
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
            bus.$on(MESSAGE_BROADCAST, this.onUserBroadcast);
            bus.$on(SET_FILE_ITEM_UUID, this.onFileItemUuid);
        },
        beforeDestroy() {
            bus.$off(SET_EDIT_MESSAGE, this.onSetMessage);
            bus.$off(USER_TYPING, this.onUserTyping);
            bus.$off(MESSAGE_BROADCAST, this.onUserBroadcast);
            bus.$off(SET_FILE_ITEM_UUID, this.onFileItemUuid);
            clearInterval(timerId);
        },
        created(){
            this.notifyAboutTyping = debounce(this.notifyAboutTyping, 500, {leading:true, trailing:false});
            this.notifyAboutBroadcast = debounce(this.notifyAboutBroadcast, 100, {leading:true, trailing:true});
        },
        watch: {
            'editMessageDto.text': {
                handler: function (newValue, oldValue) {
                    this.sendNotification();
                },
            },
            sendBroadcast: {
                handler: function (newValue, oldValue) {
                    if (!newValue) {
                        this.notifyAboutBroadcast(true);
                    } else {
                        this.notifyAboutBroadcast();
                    }
                }
            }
        },
        components: {
            quillEditor: Editor
        }
    }
</script>

<style lang="stylus">
$mobileWidth = 800px

#sendButtonContainer {
    min-height 25%
}

.quillWrapper {
    height 100%
    overflow-y auto
}

.ql-editor {
  padding 10px 8px
}

.ql-toolbar.ql-snow {
    padding 4px
}
.ql-snow .ql-picker.ql-expanded .ql-picker-options {
    top: unset
    bottom 100%
}

.ql-snow .ql-picker svg {
    position: absolute;
    margin-top: -9px;
    right: 0;
    top: 50%;
    width: 18px;
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
.ql-snow .ql-tooltip {
    left 0 !important
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