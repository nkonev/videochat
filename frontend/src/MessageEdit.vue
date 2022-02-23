<template>
    <v-container id="sendButtonContainer" class="py-0 px-1 pb-1 d-flex flex-column" fluid style="height: 100%">
            <tiptap
                :key="editorKey"
                v-model="editMessageDto.text"
                ref="tipTapRef"
                @keyup.native.ctrl.enter="sendMessageToChat"
                @keyup.native.esc="resetInput"
            />
            <div id="custom-toolbar">
                <div class="custom-toolbar-format" v-if="$refs.tipTapRef != null && $refs.tipTapRef.$data.editor != null">
                    <button
                        :class="{
                          'richText__menu-item': true,
                          active: $refs.tipTapRef.$data.editor.isActive('bold'),
                        }"
                        @click="$refs.tipTapRef.$data.editor.chain().focus().toggleBold().run()">
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'bold' }"></font-awesome-icon>
                    </button>
                    <button
                        :class="{
                          'richText__menu-item': true,
                          active: $refs.tipTapRef.$data.editor.isActive('italic'),
                        }"
                        @click="$refs.tipTapRef.$data.editor.chain().focus().toggleItalic().run()">
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'italic' }"></font-awesome-icon>
                    </button>
                    <button class="richText__menu-item"
                        :class="{
                          'richText__menu-item': true,
                          active: $refs.tipTapRef.$data.editor.isActive('underline')
                        }"
                        @click="$refs.tipTapRef.$data.editor.chain().focus().toggleUnderline().run()"
                    >
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'underline' }"></font-awesome-icon>
                    </button>
                    <button class="richText__menu-item"
                        :class="{
                          'richText__menu-item': true,
                          active: $refs.tipTapRef.$data.editor.isActive('strike'),
                        }"
                        @click="$refs.tipTapRef.$data.editor.chain().focus().toggleStrike().run()"
                    >
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'strikethrough' }"></font-awesome-icon>
                    </button>
                    <select class="ql-color" v-if="false"></select>
                    <select class="ql-background" v-if="false"></select>
                    <button class="richText__menu-item" v-if="false">link</button>
                    <button
                        class="richText__menu-item"
                        @click="$refs.tipTapRef.addImage()"
                    >
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'image' }"></font-awesome-icon>
                    </button>
                </div>
                <div class="custom-toolbar-send">
                    <v-btn v-if="!this.editMessageDto.fileItemUuid" icon tile @click="openFileUpload()"><v-icon color="primary">mdi-file-upload</v-icon></v-btn>
                    <template v-if="this.editMessageDto.fileItemUuid">
                        <v-badge
                            :value="fileCount"
                            :content="fileCount"
                            color="green"
                            overlap
                            left
                        >
                            <v-btn icon tile @click="onFilesClicked()"><v-icon>mdi-file-document-multiple</v-icon></v-btn>
                        </v-badge>
                    </template>
                    <v-btn icon tile class="mr-2" @click="resetInput()"><v-icon>mdi-delete</v-icon></v-btn>
                    <v-switch v-if="canBroadcast && $vuetify.breakpoint.smAndUp" dense hide-details class="ma-0 mr-4" v-model="sendBroadcast"
                        :label="$vuetify.breakpoint.smAndUp ? $vuetify.lang.t('$vuetify.message_broadcast') : null"
                    ></v-switch>
                    <v-btn color="primary" @click="sendMessageToChat" small class="mr-1"><v-icon color="white">mdi-send</v-icon></v-btn>
                </div>
            </div>
            <v-tooltip v-if="writingUsers.length || broadcastMessage" :activator="'#sendButtonContainer'" top v-model="showTooltip" :key="tooltipKey">
                <span v-if="!broadcastMessage">{{writingUsers.map(v=>v.login).join(', ')}} {{ $vuetify.lang.t('$vuetify.user_is_writing') }}</span>
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
    import {GET_USER, SET_TITLE} from "./store";
    import Tiptap from './TipTapEditor.vue'

    const dtoFactory = () => {
        return {
            id: null,
            text: "",
            fileItemUuid: null,
        }
    };

    let timerId;

    export default {
        props:['chatId', 'canBroadcast'],
        data() {
            return {
                editorKey: +new Date(),
                editMessageDto: dtoFactory(),
                writingUsers: [],
                showTooltip: true,
                sendBroadcast: false,
                broadcastMessage: null,
                tooltipKey: 0,
                fileCount: null,
            }
        },
        methods: {
            sendMessageToChat() {
                if (this.messageTextIsPresent()) {
                    (this.editMessageDto.id ? axios.put(`/api/chat/`+this.chatId+'/message', this.editMessageDto) : axios.post(`/api/chat/`+this.chatId+'/message', this.editMessageDto)).then(response => {
                        this.resetInput();
                    })
                }
            },
            resetInput() {
              console.log("Resetting text input");
              this.editMessageDto = dtoFactory();
              this.fileCount = null;
            },
            messageTextIsPresent() {
                return this.editMessageDto.text && this.editMessageDto.text !== "" && this.editMessageDto.text !== '<p><br></p>'
            },
            onSetMessage(dto) {
                this.editMessageDto = dto;
                this.editorKey++;
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
                } else if (this.messageTextIsPresent()) {
                    axios.put(`/api/chat/`+this.chatId+'/broadcast', {text: this.editMessageDto.text});
                }
            },
            notifyAboutTyping() {
                if (this.messageTextIsPresent()) {
                    axios.put(`/api/chat/` + this.chatId + '/typing');
                }
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

                if (!this.sendBroadcast && this.currentUser && this.currentUser.id == data.participantId) {
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
            },
            '$vuetify.lang.current': {
                handler: function (newValue, oldValue) {
                    this.editorKey++;
                },
            }
        },
        components: {
            Tiptap
        }
    }
</script>

<style lang="stylus">
$mobileWidth = 800px

#sendButtonContainer {
    min-height 25%
}

#custom-toolbar {
    display: flex;
    align-items: center
    justify-content: space-between
    border-top-width: 0
    border-bottom-style dashed
    border-left-style dashed
    border-right-style dashed
    border-width 1px
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
    flex-grow: 0;

    .richText__menu-item {
        min-width: 1.75rem;
        color: rgba(0, 0, 0, 0.87);
        border: none;
        background-color: transparent;
        border-radius: 0.4rem;
        padding: 0.25rem;
        margin-right: 0.35rem;
        cursor: pointer;
    }

    .richText__menu-item.active,
    .richText__menu-item:hover {
        color: #fff;
        background-color: rgba(0, 0, 0, 0.87);
    }
}
.custom-toolbar-send {
    display: flex;
    flex-grow: 10
    justify-content flex-end
    align-items center
}
</style>