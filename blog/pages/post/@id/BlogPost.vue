<template>
  <v-container class="ma-0 pa-0" fluid>
    <h1 v-html="blogDto.title" class="ml-3 mt-2"></h1>

    <div class="pr-1 mr-1 pl-1 mt-0 ml-3 message-item-root" >
      <div class="message-item-with-buttons-wrapper">
        <v-list-item class="pl-2 pt-0" v-if="blogDto?.owner">
          <template v-slot:prepend v-if="hasLength(blogDto.owner.avatar)">
            <div class="item-avatar pr-0 mr-3">
              <a :href="getProfileLink(blogDto.owner)" class="user-link">
                  <img :src="blogDto.owner.avatar">
              </a>
            </div>

          </template>

          <template v-slot:default>
            <div class="ma-0 pa-0 d-flex top-panel">
              <div class="author-and-date">
                <v-list-item-title><a class="colored-link" :href="getProfileLink(blogDto.owner)">{{blogDto.owner.login}}</a></v-list-item-title>
                <v-list-item-subtitle>{{getDate(blogDto.createDateTime)}}</v-list-item-subtitle>
              </div>
              <div class="ma-0 pa-0 go-to-chat">
                <v-btn variant="plain" rounded size="large" :href="getChatLink()" @click.prevent="toChat()"><v-icon size="large">mdi-forum</v-icon></v-btn>
              </div>
            </div>
          </template>
        </v-list-item>

        <div class="pa-0 ma-0 mt-1 message-item-wrapper post-content">
          <v-container v-html="blogDto.text" class="message-item-text ml-0"></v-container>
          <div class="mt-0 ml-2 mr-4 reactions" v-if="shouldShowReactions(blogDto)">
            <v-btn v-for="(reaction, i) in blogDto.reactions" variant="tonal" size="small" height="32px" rounded :class="reactionClass(i)" :title="getReactedUsers(reaction)"><span v-if="reaction.count > 1" class="text-body-2 with-space">{{ '' + reaction.count + ' ' }}</span><span class="text-h6">{{ reaction.reaction }}</span></v-btn>
          </div>
        </div>
      </div>
    </div>

    <template v-if="blogDto.messageId">
        <v-container class="ma-0 pa-0 mb-2" fluid>
          <MessageItem v-for="(item, index) in items"
            :id="getItemId(item.id)"
            :key="item.id"
            :item="item"
            :chatId="item.chatId"
            :isInBlog="true"
          ></MessageItem>
        </v-container>
    </template>
  </v-container>
</template>

<script>
import MessageItem from "./MessageItem.vue";
import {getHumanReadableDate, hasLength, isMobileBrowser} from "#root/common/utils";
import {chat, messageIdHashPrefix, messageIdPrefix, profile} from "#root/common/router/routes";
import { navigate } from 'vike/client/router';
import {usePageContext} from "#root/renderer/usePageContext.js";


export default {
  setup() {
    const pageContext = usePageContext();

    // expose to template and other options API hooks
    return {
        pageContext
    }
  },
  data() {
      return this.pageContext.data;
  },
  methods: {
    hasLength,
    isMobile() {
       return isMobileBrowser()
    },
    getProfileLink(user) {
      let url = profile + "/" + user.id;
      return url;
    },
    getChatLink() {
      return chat + '/' + this.blogDto.chatId + messageIdHashPrefix + this.blogDto.messageId;
    },
    async toChat() {
        await navigate(this.getChatLink());
    },
    getDate(date) {
      if (hasLength(date)) {
        return getHumanReadableDate(date)
      } else {
        return null
      }
    },
    getItemId(id) {
      return messageIdPrefix + id
    },
    shouldShowReactions(item) {
      return item?.reactions?.length
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
  components: {
    MessageItem,
  },
  computed: {
  },
  async mounted() {
  },
  beforeUnmount() {
  },
}
</script>

<style lang="stylus" scoped>
@import "../../../common/styles/common.styl"
@import "../../../common/styles/messageWrapper.styl"
@import "../../../common/styles/itemAvatar.styl"

.my-messages-scroller {
  height 100%
  width: 100%
  display flex
  flex-direction column
  overflow-y scroll !important
  background white
}

.top-panel {
  width 100%
}

.go-to-chat {
  align-self center
}

.post-content {
  position relative
  z-index 100
  background white
  border-color $borderColor
  border-style: dashed;
  border-width 1px
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.05),
    0px 10px 20px rgba(0, 0, 0, 0.1)
}

.author-and-date {
  flex: 0 1 auto;
}
.user-link {
    height 100%
}

</style>
