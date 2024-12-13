<template>
    <div class="px-1 mx-1 mt-4 message-item-root" :id="id">
        <div v-if="hasLength(item?.owner?.avatar)" class="item-avatar ml-1 mr-2 mt-1">
          <a :href="getOwnerLink(item)" class="user-link" >
            <img :src="item.owner?.avatar">
          </a>
        </div>
        <div class="message-item-with-buttons-wrapper">
            <v-container class="ma-0 pa-0 d-flex caption-small">
                <a :href="getOwnerLink(item)" class="nodecorated-link" :style="getLoginColoredStyle(item.owner, true)">{{getOwner(item.owner)}}</a>
                <span class="with-space"> at </span>
                <span class="mr-1">{{getDate(item)}}</span>
            </v-container>
            <div class="pa-0 ma-0 mt-1 message-item-wrapper" :class="{ my: my, highlight: highlight }">
                <div v-if="item.embedMessage" class="embedded-message">
                    <template v-if="canRenderLinkToSource(item)">
                        <a class="caption-small">{{getEmbedHead(item)}}</a>
                    </template>
                    <template v-else>
                        <div class="caption-small">
                            {{getEmbedHeadLite(item)}}
                        </div>
                    </template>
                    <div :class="embedClass()" v-html="item.embedMessage.text"></div>
                </div>
                <!-- We use div instead of v-container because last is not working with SSR -->
                <div v-if="shouldShowMainContainer(item)" v-html="item.text" :class="messageClass(item)"></div>
                <div class="mt-0 ml-2 mr-4 reactions" v-if="shouldShowReactions(item)">
                  <v-btn v-for="(reaction, i) in item.reactions" variant="flat" size="small" height="32px" rounded :class="reactionClass(i)" :title="getReactedUsers(reaction)"><span v-if="reaction.count > 1" class="text-body-1 with-space">{{ '' + reaction.count + ' ' }}</span><span class="text-h6">{{ reaction.reaction }}</span></v-btn>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
    import {usePageContext} from "#root/renderer/usePageContext.js";
    import {
        embed_message_reply,
        embed_message_resend,
        getHumanReadableDate, getLoginColoredStyle, hasLength,
    } from "#root/common/utils";
    import "#root/common/styles/messageBody.styl";
    import "#root/common/styles/messageWrapper.styl";
    import "#root/common/styles/itemAvatar.styl";

    import {profile} from "#root/common/router/routes"

    export default {
        setup() {
            const pageContext = usePageContext();

            // expose to template and other options API hooks
            return {
                pageContext
            }
        },
        props: ['id', 'item', 'chatId', 'my', 'highlight', 'isInBlog'],
        methods: {
            getLoginColoredStyle,
            hasLength,
            isMobile() {
                return this.pageContext.isMobile
            },
            getOwnerLink(item) {
                return profile + "/" + item.owner?.id;
            },

            getOwner(owner) {
                return owner?.login
            },
            getDate(item) {
                return getHumanReadableDate(item.createDateTime)
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
                    return this.getOwner(item.embedMessage.owner) + ' in ' + item.embedMessage.chatName;
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
            embedClass() {
                return this.isMobile() ? ['message-item-text', 'message-item-text-mobile'] : ['message-item-text']
            },
            messageClass(item) {
              let classes = ['message-item-text', 'ml-0', 'v-container'];
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

        },
        created() {
        },
    }
</script>

<style lang="stylus" scoped>
  @import "../styles/constants.styl"

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

</style>
