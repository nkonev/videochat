<template>
    <div class="pr-1 mr-1 pl-4 mt-4 message-item-root" :id="'message-' + item.id">
        <router-link :to="{ name: 'profileUser', params: { id: item.owner.id }}" class="user-link">
            <v-list-item-avatar v-if="item.owner && item.owner.avatar" class="pr-0 mr-3">
                <v-img :src="item.owner.avatar"></v-img>
            </v-list-item-avatar>
        </router-link>

        <div class="message-item-with-buttons-wrapper">
            <v-container class="ma-0 pa-0 d-flex list-item-head">
                <router-link :to="{ name: 'profileUser', params: { id: item.owner.id }}">{{getOwner(item)}}</router-link><span class="with-space"> {{$vuetify.lang.t('$vuetify.time_at')}} </span>{{getDate(item)}}
                <v-icon class="mx-1 ml-2" v-if="item.fileItemUuid" @click="onFilesClicked(item.fileItemUuid)" small :title="$vuetify.lang.t('$vuetify.attached_message_files')">mdi-file-download</v-icon>
                <v-icon class="mx-1" v-if="item.canDelete" color="error" @click="deleteMessage(item)" dark small :title="$vuetify.lang.t('$vuetify.delete_btn')">mdi-delete</v-icon>
                <v-icon class="mx-1" v-if="item.canEdit" color="primary" @click="editMessage(item)" dark small :title="$vuetify.lang.t('$vuetify.edit')">mdi-lead-pencil</v-icon>
                <a class="mx-2 hash" :href="require('./routes').chat + '/' + chatId + require('./routes').messageIdHashPrefix + item.id" :title="$vuetify.lang.t('$vuetify.link')">#</a>
                <v-icon class="mx-1" small :title="$vuetify.lang.t('$vuetify.reply')">mdi-reply</v-icon>
                <v-icon class="mx-1" small :title="$vuetify.lang.t('$vuetify.share')">mdi-share</v-icon>
            </v-container>
            <div @click="onMessageClick(item)" @mousemove="onMessageMouseMove(item)" class="pa-0 ma-0 mt-1 message-item-wrapper" :class="{ my: my, highlight: highlight }" >
                <div v-if="item.embedMessage" class="embedded-message">
                    <div class="list-item-head">{{item.embedMessage.ownerId}}</div>
                    <div class="ma-0 message-item-text" v-html="item.embedMessage.text"></div>
                </div>
                <v-container v-html="item.text" class="ma-0 message-item-text" :style="item.embedMessage ? 'padding-top: 0.5em': ''"></v-container>
            </div>
        </div>
    </div>
</template>

<script>
    import axios from "axios";
    import bus, {
        CLOSE_SIMPLE_MODAL,
        OPEN_SIMPLE_MODAL,
        OPEN_VIEW_FILES_DIALOG,
        OPEN_EDIT_MESSAGE, SET_EDIT_MESSAGE
    } from "./bus";
    import debounce from "lodash/debounce";
    import {getHumanReadableDate, setIcon} from "@/utils";
    import "./message.styl";

    export default {
        props: ['item', 'chatId', 'my', 'highlight'],
        methods: {
            onMessageClick(dto) {
                axios.put(`/api/chat/${this.chatId}/message/read/${dto.id}`)
            },
            onMessageMouseMove(item) {
                this.onMessageClick(item);
            },
            deleteMessage(dto){
                bus.$emit(OPEN_SIMPLE_MODAL, {
                    buttonName: this.$vuetify.lang.t('$vuetify.delete_btn'),
                    title: this.$vuetify.lang.t('$vuetify.delete_message_title', dto.id),
                    text:  this.$vuetify.lang.t('$vuetify.delete_message_text'),
                    actionFunction: ()=> {
                        axios.delete(`/api/chat/${this.chatId}/message/${dto.id}`)
                            .then(() => {
                                bus.$emit(CLOSE_SIMPLE_MODAL);
                            })
                    }
                });
            },
            editMessage(dto){
                const editMessageDto = {id: dto.id, text: dto.text, fileItemUuid: dto.fileItemUuid};
                if (!this.isMobile()) {
                    bus.$emit(SET_EDIT_MESSAGE, editMessageDto);
                } else {
                    bus.$emit(OPEN_EDIT_MESSAGE, editMessageDto);
                }
            },
            getOwner(item) {
                return item.owner.login
            },
            getDate(item) {
                return getHumanReadableDate(item.createDateTime)
            },
            onFilesClicked(itemId) {
                bus.$emit(OPEN_VIEW_FILES_DIALOG, {chatId: this.chatId, fileItemUuid :itemId});
            }
        },
        created() {
            this.onMessageMouseMove = debounce(this.onMessageMouseMove, 1000, {leading:true, trailing:false});
        },
    }
</script>

<style lang="stylus">
  @import "common.styl"

  .embedded-message {
      background: #f9f9f9;
      border-radius 0 10px 10px 0
      border-left: 4px solid #ccc;
      margin: 0.5em 0.5em 0 0.5em;
      padding: 0.3em 0.5em 0.5em 0.5em;
      quotes: "\201C""\201D""\2018""\2019";
  }

  .user-link {
    height 100%
  }

  .message-item-root {
      // align-items: center;
      display: flex;
      flex: 1 1 100%;
      letter-spacing: normal;
      min-height: 48px;
      outline: none;
      padding: 0 16px;
      padding-right: 16px;
      padding-left: 16px;
      position: relative;
      text-decoration: none;
  }
  .message-item-with-buttons-wrapper {
      flex 1 1
  }
  .message-item-wrapper {
      border-radius 10px
      background #efefef
      display inline-block
      word-wrap break-word
      overflow-wrap break-word
  }
  .message-item-text {
      line-height: 1.1;
      -ms-word-break: break-all;
      /* This is the dangerous one in WebKit, as it breaks things wherever */
      word-break: break-all;
      /* Instead use this non-standard one: */
      word-break: break-word;

      white-space: pre-wrap

      /* Adds a hyphen where the word breaks, if supported (No Blink) */
      -ms-hyphens: auto;
      -moz-hyphens: auto;
      -webkit-hyphens: auto;
      hyphens: auto;
      p {
          margin-bottom unset
      }
      p:empty:after {
          content: '\200b';
      }
  }
  .with-space {
      white-space: pre;
  }
  .my {
      background $messageSelectedBackground
  }

  .hash {
      align-items: center;
      display: inline-flex;
  }

  .highlight {
      animation: anothercolor 10s;
  }

  @keyframes anothercolor {
      0% { background: yellow }
  }

</style>
