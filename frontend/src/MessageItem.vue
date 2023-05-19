<template>
    <div class="pr-1 mr-1 pl-4 mt-4 message-item-root" :id="'message-' + item.id">
        <router-link :to="{ name: 'profileUser', params: { id: item.owner.id }}" class="user-link">
            <v-list-item-avatar v-if="item.owner && item.owner.avatar" class="pr-0 mr-3">
                <img :src="item.owner.avatar"></img>
            </v-list-item-avatar>
        </router-link>

        <div class="message-item-with-buttons-wrapper">
            <v-container class="ma-0 pa-0 d-flex list-item-head">
                <router-link :to="{ name: 'profileUser', params: { id: item.owner.id }}">{{getOwner(item.owner)}}</router-link><span class="with-space"> {{$vuetify.lang.t('$vuetify.time_at')}} </span>{{getDate(item)}}
                <template v-if="!isMobile() && !isInBlog">
                    <v-icon class="mx-1 ml-2" v-if="item.fileItemUuid" @click="onFilesClicked(item)" small :title="$vuetify.lang.t('$vuetify.attached_message_files')">mdi-file-download</v-icon>
                    <v-icon class="mx-1" v-if="item.canDelete" color="error" @click="deleteMessage(item)" dark small :title="$vuetify.lang.t('$vuetify.delete_btn')">mdi-delete</v-icon>
                    <v-icon class="mx-1" v-if="item.canEdit" color="primary" @click="editMessage(item)" dark small :title="$vuetify.lang.t('$vuetify.edit')">mdi-lead-pencil</v-icon>
                    <v-icon class="mx-1" small :title="$vuetify.lang.t('$vuetify.reply')" @click="replyOnMessage(item)">mdi-reply</v-icon>
                    <v-icon v-if="canResend" class="mx-1" small :title="$vuetify.lang.t('$vuetify.share')" @click="shareMessage(item)">mdi-share</v-icon>
                    <v-icon v-if="!item.pinned" class="mx-1" small :title="$vuetify.lang.t('$vuetify.pin_message')" @click="pinMessage(item)">mdi-pin</v-icon>
                    <v-icon v-if="item.pinned" class="mx-1" small :title="$vuetify.lang.t('$vuetify.remove_from_pinned')" @click="removedFromPinned(item)">mdi-pin-off-outline</v-icon>
                    <a v-if="item.blogPost" class="mx-1" :href="getBlogLink(item)" :title="$vuetify.lang.t('$vuetify.go_to_blog_post')"><v-icon small>mdi-postage-stamp</v-icon></a>
                    <router-link class="mx-1 hash" :to="getMessageLink(item)" :title="$vuetify.lang.t('$vuetify.link')">#</router-link>
                </template>
            </v-container>
            <div class="pa-0 ma-0 mt-1 message-item-wrapper" :class="{ my: my, highlight: highlight }" @click="onMessageClick(item)" @mousemove="onMessageMouseMove(item)" @contextmenu="onShowContextMenu($event, item)">
                <div v-if="item.embedMessage" class="embedded-message">
                    <template v-if="canRenderLinkToSource(item)">
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
    import {blog, chat_name, messageIdHashPrefix} from "./routes"

    export default {
        props: ['item', 'chatId', 'my', 'highlight', 'canResend', 'isInBlog'],
        methods: {
            onMessageClick(dto) {
                if (!this.isInBlog) {
                    axios.put(`/api/chat/${this.chatId}/message/read/${dto.id}`)
                }
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
            pinMessage(dto) {
                this.$emit('pinMessage', dto)
            },
            removedFromPinned(dto) {
                this.$emit('removedFromPinned', dto)
            },
            getBlogLink() {
                return blog + '/post/' + this.chatId;
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
            canRenderLinkToSource(item) {
                if (item.embedMessage.embedType == embed_message_reply) {
                    return true
                } else if (item.embedMessage.embedType == embed_message_resend) {
                    if (item.embedMessage.chatName) {
                        return true
                    }
                }
                return false
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


  .with-space {
      white-space: pre;
  }

  .hash {
      align-items: center;
      display: inline-flex;
      font-size: larger;
      text-decoration: none;
  }

  .highlight {
      animation: anothercolor 10s;
  }

  @keyframes anothercolor {
      0% { background: yellow }
  }

</style>
