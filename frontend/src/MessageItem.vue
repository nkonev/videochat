<template>
    <div class="pr-1 mr-1 pl-4 mt-4 message-item-root" :id="'message-' + item.id">
        <router-link :to="{ name: 'profileUser', params: { id: item.owner.id }}" class="user-link">
            ava
        </router-link>

        <div class="message-item-with-buttons-wrapper">
            <v-container class="ma-0 pa-0 d-flex list-item-head">
                <router-link :to="{ name: 'profileUser', params: { id: item.owner.id }}">{{getOwner(item.owner)}}</router-link><span class="with-space"> {{$vuetify.lang.t('$vuetify.time_at')}} </span>{{getDate(item)}}
                <v-icon class="mx-1 ml-2" v-if="item.fileItemUuid" @click="onFilesClicked(item)" small :title="$vuetify.lang.t('$vuetify.attached_message_files')">mdi-file-download</v-icon>
                <template v-if="!isMobile()">
                    <v-icon class="mx-1" v-if="item.canDelete" color="error" @click="deleteMessage(item)" dark small :title="$vuetify.lang.t('$vuetify.delete_btn')">mdi-delete</v-icon>
                    <v-icon class="mx-1" v-if="item.canEdit" color="primary" @click="editMessage(item)" dark small :title="$vuetify.lang.t('$vuetify.edit')">mdi-lead-pencil</v-icon>
                    <router-link class="mx-2 hash" :to="getMessageLink(item)" :title="$vuetify.lang.t('$vuetify.link')">#</router-link>
                    <v-icon class="mx-1" small :title="$vuetify.lang.t('$vuetify.reply')" @click="replyOnMessage(item)">mdi-reply</v-icon>
                    <v-icon v-if="canResend" class="mx-1" small :title="$vuetify.lang.t('$vuetify.share')" @click="shareMessage(item)">mdi-share</v-icon>
                </template>
            </v-container>
            <div class="pa-0 ma-0 mt-1 message-item-wrapper" :class="{ my: my, highlight: highlight }" @click="onMessageClick(item)" @mousemove="onMessageMouseMove(item)" @contextmenu="onShowContextMenu($event, item)">
                <div v-if="item.embedMessage" class="embedded-message">
                    <template v-if="item.embedMessage.chatName">
                        <router-link class="list-item-head" :to="getEmbedLinkTo(item)">{{getEmbedHead(item)}}</router-link>
                    </template>
                    <template v-else>
                        <div class="list-item-head">
                            {{getEmbedHeadLite(item)}}
                        </div>
                    </template>
                    <div class="message-item-text" v-html="item.embedMessage.text"></div>
                </div>
                <v-container v-if="shouldShowMainContainer(item)" v-html="item.text" class="message-item-text ml-0" :class="item.embedMessage  ? 'after-embed': ''"></v-container>
            </div>
        </div>
    </div>
</template>

<script>
    import axios from "axios";
    import debounce from "lodash/debounce";
    import {
        embed_message_reply,
        embed_message_resend,
        getHumanReadableDate,
    } from "@/utils";
    import "./message.styl";
    import {chat_name, messageIdHashPrefix} from "./routes"

    export default {
        props: ['item', 'chatId', 'my', 'highlight', 'canResend'],
        methods: {
            onMessageClick(dto) {
                axios.put(`/api/chat/${this.chatId}/message/read/${dto.id}`)
            },
            onMessageMouseMove(item) {
                this.onMessageClick(item);
            },
            deleteMessage(dto){
                this.$emit('deleteMessage', dto)
            },
            editMessage(dto){
                this.$emit('editMessage', dto)
            },
            replyOnMessage(dto) {
                this.$emit('replyOnMessage', dto)
            },
            shareMessage(dto) {
                this.$emit('shareMessage', dto)
            },
            onFilesClicked(dto) {
                this.$emit('onFilesClicked', dto)
            },

            getOwner(owner) {
                return owner.login
            },
            getDate(item) {
                return getHumanReadableDate(item.createDateTime)
            },
            getMessageLink(item) {
                return {
                    name: this.$route.name,
                    params: {
                        id: this.chatId
                    },
                    hash: messageIdHashPrefix + item.id
                }
            },
            getEmbedLinkTo(item) {
                let chatId;
                let name;
                if (item.embedMessage.embedType == embed_message_reply) {
                    chatId = this.chatId;
                    name = this.$route.name;
                } else if (item.embedMessage.embedType == embed_message_resend && item.embedMessage.isParticipant) {
                    chatId = item.embedMessage.chatId;
                    name = chat_name;
                }
                return {
                    name: name,
                    params: {
                        id: chatId
                    },
                    hash: messageIdHashPrefix + item.embedMessage.id
                }
            },
            getEmbedHead(item){
                if (item.embedMessage.embedType == embed_message_reply) {
                    return this.getOwner(item.embedMessage.owner)
                } else if (item.embedMessage.embedType == embed_message_resend) {
                    return this.getOwner(item.embedMessage.owner) + this.$vuetify.lang.t('$vuetify.in') + item.embedMessage.chatName;
                }
            },
            getEmbedHeadLite(item){
                if (item.embedMessage.embedType == embed_message_resend) {
                    return this.getOwner(item.embedMessage.owner)
                }
            },
            shouldShowMainContainer(item) {
                return item.embedMessage == null || item.embedMessage.embedType == embed_message_reply
            },
            onShowContextMenu(event, item) {
                this.$emit('contextmenu', event, item)
            },
        },
        created() {
            this.onMessageMouseMove = debounce(this.onMessageMouseMove, 1000, {leading:true, trailing:false});
        },
    }
</script>

<style lang="stylus">
  @import "common.styl"

  .embedded-message {
      background: $embedMessageColor;
      border-radius 0 10px 10px 0
      border-left: 4px solid #ccc;
      margin: 0.5em 0.5em 0.5em 0.5em;
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
      display: flex;
      flex-direction: column;
      justify-content: flex-start;
      align-items: baseline;
      width: fit-content;
      word-wrap break-word
      overflow-wrap break-word

      .after-embed {
          padding-top: 0
      }
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
