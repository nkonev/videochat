<template>
    <div class="px-1 mx-1 mt-4 message-item-root" :class="isCompact ? 'message-item-root-compact' : ''" :id="id">
        <div v-if="hasLength(item?.owner?.avatar)" class="item-avatar ml-1 mr-2 mt-1">
          <a :href="getOwnerLink(item)" class="user-link" @click.prevent.stop="onProfileClick(item)">
            <img :src="item.owner?.avatar">
          </a>
        </div>
        <div class="message-item-with-buttons-wrapper">
            <v-container class="ma-0 pa-0 d-flex align-center caption-small">
                <a :href="getOwnerLink(item)" class="nodecorated-link" @click.prevent.stop="onProfileClick(item)" :style="getLoginColoredStyle(item.owner, true)" :title="getDate(item)" v-html="getOwner(item.owner)"></a>
                <span class="with-space" v-if="!isCompact"> {{$vuetify.locale.t('$vuetify.time_at')}} </span>
                <router-link v-if="!isCompact" class="gray-link" :to="getMessageLink(item)" :title="$vuetify.locale.t('$vuetify.link')"><span class="mr-1">{{getDate(item)}}</span></router-link>
                <span v-if="!isMobile() && !isCompact" class="ma-0 pa-0 message-quick-buttons">
                    <v-icon class="mx-1" v-if="item.fileItemUuid" @click="onFilesClicked(item)" size="small" :title="$vuetify.locale.t('$vuetify.attached_message_files')">mdi-file-download</v-icon>
                    <v-icon class="mx-1" v-if="item.canDelete" color="red" @click="deleteMessage(item)" dark size="small" :title="$vuetify.locale.t('$vuetify.delete_btn')">mdi-delete</v-icon>
                    <v-icon class="mx-1" v-if="item.canEdit" color="primary" @click="editMessage(item)" dark size="small" :title="$vuetify.locale.t('$vuetify.edit')">mdi-lead-pencil</v-icon>
                    <v-icon class="mx-1" size="small" :title="$vuetify.locale.t('$vuetify.reply')" @click="replyOnMessage(item)">mdi-reply</v-icon>
                    <v-icon class="mx-1" v-if="areReactionsAllowed" size="small" :title="$vuetify.locale.t('$vuetify.add_reaction_on_message')" @click="reactionOnMessage(item)">mdi-emoticon-outline</v-icon>
                    <a v-if="item.blogPost" class="mx-1 colored-link" :href="getBlogLink(item)" :title="$vuetify.locale.t('$vuetify.go_to_blog_post')"><v-icon size="small">mdi-postage-stamp</v-icon></a>
                </span>
            </v-container>
            <div class="pa-0 ma-0 mt-1 message-item-wrapper" :class="{ my: my, highlight: highlight }" @click="onMessageClick($event, item)" @mousemove="onMessageMouseMove(item)" @contextmenu="onShowContextMenu($event, item)">
                <div v-if="item.embedMessage" class="embedded-message">
                    <template v-if="canRenderLinkToSource(item)">
                        <router-link class="caption-small" :to="getEmbedLinkTo(item)">{{getEmbedHead(item)}}</router-link>
                    </template>
                    <template v-else>
                        <div class="caption-small">
                            {{getEmbedHeadLite(item)}}
                        </div>
                    </template>
                    <div :class="embedClass()">
                      <span v-html="buildEmbedHtml(item)"></span>
                      <template v-if="item.embedMessage.initiallyCollapsed">
                        <span class="caption-small" v-if="item.embedMessage.collapsed" @click="item.embedMessage.collapsed = false" style="cursor: pointer"> ...{{$vuetify.locale.t('$vuetify.expand')}}</span>
                        <span class="caption-small" v-if="!item.embedMessage.collapsed" @click="item.embedMessage.collapsed = true" style="cursor: pointer"> ...{{$vuetify.locale.t('$vuetify.collapse')}}</span>
                      </template>
                    </div>
                </div>
                <v-container v-if="shouldShowMainContainer(item)" v-html="item.text" :class="messageClass(item)"></v-container>
                <div class="mt-0 ml-2 mr-4 reactions" v-if="shouldShowReactions(item)">
                  <v-btn v-for="(reaction, i) in item.reactions" variant="flat" size="small" height="32px" rounded :class="reactionClass(i)" @click="onExistingReactionClick(reaction.reaction)" :title="getReactedUsers(reaction)"><span v-if="reaction.count > 1" class="text-body-1 with-space">{{ '' + reaction.count + ' ' }}</span><span class="text-h6">{{ reaction.reaction }}</span></v-btn>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
    import axios from "axios";
    import debounce from "lodash/debounce";
    import {
      embed_message_reply,
      embed_message_resend, getBlogLink,
      getLoginColoredStyle, getMessageLinkRouteObject, hasLength, isStrippedUserLogin,
    } from "@/utils";
    import { getHumanReadableDate } from "@/date.js";
    import "./messageBody.styl";
    import "./messageWrapper.styl";
    import "./itemAvatar.styl";

    import {chat_name, messageIdHashPrefix, profile, profile_name} from "@/router/routes"
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore.js";

    export default {
        props: ['id', 'item', 'chatId', 'my', 'highlight', 'isCompact'],
        computed: {
          ...mapStores(useChatStore),
          areReactionsAllowed() {
            return this.chatStore.chatDto.canReact
          },
        },
        methods: {
            getLoginColoredStyle,
            hasLength,
            getOwnerRoute(item) {
                return { name: profile_name, params: { id: item.owner?.id }}
            },
            getOwnerLink(item) {
                return profile + "/" + item.owner?.id;
            },
            onProfileClick(item) {
                const route = this.getOwnerRoute(item);
                this.$router.push(route);
            },
            onMessageClick(event, dto) {
                if (this.isMobile()) {
                  this.emitContextmenuEvent(event, dto);
                }
                this.sendRead(dto);
            },
            onMessageMouseMove(item) {
                this.sendRead(item);
            },
            // we cannot move it from front to back because there are messages-over-websocket
            // and itsn't clear did user read them or not
            sendRead(dto) {
              axios.put(`/api/chat/${this.chatId}/message/read/${dto.id}`)
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
            reactionOnMessage(dto) {
                this.$emit('addReaction', dto)
            },
            onFilesClicked(dto) {
                this.$emit('onFilesClicked', dto)
            },
            getBlogLink() {
                return getBlogLink(this.chatId)
            },

            getOwner(owner) {
                let bldr = owner?.login;
                if (bldr) {
                  if (isStrippedUserLogin(owner)) {
                    bldr = "<s>" + bldr + "</s>"
                  }
                }
                return bldr
            },
            getDate(item) {
                return getHumanReadableDate(item.createDateTime)
            },
            getMessageLink(item) {
                const routeObject = getMessageLinkRouteObject(this.chatId, item.id);
                routeObject.name = this.$route.name;
                routeObject.query = this.$route.query;
                return routeObject
            },
            getEmbedLinkTo(item) {
                let chatId;
                let name;
                let query = undefined;
                if (item.embedMessage.embedType == embed_message_reply) {
                    chatId = this.chatId;
                    name = this.$route.name;
                    query = this.$route.query;
                } else if (item.embedMessage.embedType == embed_message_resend && item.embedMessage.isParticipant) {
                    chatId = item.embedMessage.chatId;
                    name = chat_name;
                }
                return {
                    name: name,
                    params: {
                        id: chatId
                    },
                    query: query,
                    hash: messageIdHashPrefix + item.embedMessage.id,
                }
            },
            canRenderLinkToSource(item) {
                if (item.embedMessage.embedType == embed_message_reply) {
                    return true
                } else if (item.embedMessage.embedType == embed_message_resend && item.embedMessage.isParticipant) {
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
                    return this.getOwner(item.embedMessage.owner) + this.$vuetify.locale.t('$vuetify.in') + item.embedMessage.chatName;
                }
            },
            getEmbedHeadLite(item){
                if (item.embedMessage.embedType == embed_message_resend) {
                    return this.getOwner(item.embedMessage.owner)
                }
            },
            shouldShowReactions(item) {
              return item?.reactions?.length
            },
            shouldShowMainContainer(item) {
                return item.embedMessage == null || item.embedMessage.embedType == embed_message_reply
            },
            onShowContextMenu(event, item) {
                if (!this.isMobile()) {
                    this.emitContextmenuEvent(event, item)
                }
            },
            emitContextmenuEvent(event, item) {
                this.$emit('customcontextmenu', event, item)
            },
            embedClass() {
                return this.isMobile() ? ['message-item-text', 'message-item-text-mobile'] : ['message-item-text']
            },
            messageClass(item) {
              let classes = ['message-item-text', 'ml-0'];
              if (this.isMobile()) {
                classes.push('message-item-text-mobile');
              }
              if (item.embedMessage) {
                classes.push('after-embed');
              }
              if (this.shouldShowReactions(item)) {
                classes.push('pb-2');
              }
              return classes
            },
            onExistingReactionClick(reaction) {
              if (this.areReactionsAllowed) {
                this.$emit('onreactionclick', {id: this.item.id, reaction: reaction})
              }
            },
            reactionClass(i) {
              const classes = [];
              classes.push("mb-2")
              if (i > 0) {
                classes.push("ml-2")
              }
              return classes
            },
            getReactedUsers(reactionObj) {
                return reactionObj.users?.map(u => u.login).join(", ")
            },
            buildEmbedHtml(item) {
              if (item.embedMessage.collapsed) {
                return item.embedMessage.collapsedText
              } else {
                return item.embedMessage.text
              }
            },
        },
        created() {
            this.onMessageMouseMove = debounce(this.onMessageMouseMove, 1000, {leading:true, trailing:false});
        },
    }
</script>

<style lang="stylus" scoped>
  @import "constants.styl"

  .caption-small {
    text-decoration none
    a {
      text-decoration none
    }
  }

  .embedded-message {
      background: $embedMessageColor;
      border-radius 0 10px 10px 0
      border-left: 4px solid #ccc;
      margin: 0.5em 0.5em 0.5em 0.5em;
      padding: 0.3em 0.5em 0.5em 0.5em;
      quotes: "\201C""\201D""\2018""\2019";

      .message-item-text {
        padding: revert
      }

  }

  .user-link {
    height 100%
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

  .message-quick-buttons {
      opacity: 0.35;
  }

  .message-quick-buttons:hover {
      opacity: 1;
  }

</style>
