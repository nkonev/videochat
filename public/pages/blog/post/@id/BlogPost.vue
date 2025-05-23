<template>
  <v-container class="ma-0 pa-0 my-list-container" fluid>
  <h1 v-if="pageContext.data.blogDto.is404">404 Not found</h1>
  <div v-else class="my-messages-scroller">
    <h1 v-html="pageContext.data.blogDto.title" class="ml-3 mt-2"></h1>

    <div class="pr-1 mr-1 pl-1 mt-0 ml-3 message-item-root" >
      <div class="message-item-with-buttons-wrapper">
        <v-list-item class="pl-2 pt-0" v-if="pageContext.data.blogDto?.owner">
          <template v-slot:prepend v-if="hasLength(pageContext.data.blogDto.owner.avatar)">
            <div class="item-avatar pr-0 mr-3">
              <a :href="getProfileLink(pageContext.data.blogDto.owner)" class="user-link">
                  <img :src="pageContext.data.blogDto.owner.avatar">
              </a>
            </div>

          </template>

          <template v-slot:default>
            <div class="ma-0 pa-0 d-flex top-panel">
              <div class="author-and-date" v-if="pageContext.data.blogDto.owner">
                <v-list-item-title><a class="nodecorated-link" :style="getLoginColoredStyle(pageContext.data.blogDto.owner, true)" :href="getProfileLink(pageContext.data.blogDto.owner)" v-html="getOwner(pageContext.data.blogDto.owner)"></a></v-list-item-title>
                <v-list-item-subtitle>{{getDate(pageContext.data.blogDto.createDateTime)}}</v-list-item-subtitle>
              </div>
              <div class="ma-0 pa-0 go-to-chat">
                <v-btn variant="plain" tile size="large" :href="getChatMessageLink()" title="Go to the message in chat" min-width="32px" class="px-3"><v-icon size="large">mdi-forum</v-icon></v-btn>
                <v-btn v-if="pageContext.data.blogDto.fileItemUuid" variant="plain" tile size="large" @click="onFilesClickedPostHeader()" title="Attached message files" min-width="32px" class="px-3"><v-icon size="large">mdi-file-download</v-icon> </v-btn>
              </div>
            </div>
          </template>
        </v-list-item>

        <div class="pa-0 ma-0 mt-1 message-item-wrapper post-content" @click="onClickTrap">
          <v-container v-html="pageContext.data.blogDto.text" class="message-item-text ml-0"></v-container>
          <div class="mt-0 ml-2 mr-4 reactions" v-if="shouldShowReactions(pageContext.data.blogDto)">
            <v-btn v-for="(reaction, i) in pageContext.data.blogDto.reactions" variant="tonal" size="small" height="32px" rounded :class="reactionClass(i)" :title="getReactedUsers(reaction)"><span v-if="reaction.count > 1" class="text-body-2 with-space">{{ '' + reaction.count + ' ' }}</span><span class="text-h6">{{ reaction.reaction }}</span></v-btn>
          </div>
        </div>
      </div>
    </div>

    <template v-if="pageContext.data.blogDto.messageId">
        <v-container class="ma-0 pa-0" fluid>
          <MessageItem v-for="(item, index) in pageContext.data.items"
            :id="getItemId(item.id)"
            :key="item.id"
            :item="item"
            :chatId="item.chatId"
            :isInBlog="true"
            @click="onClickTrap"
            @onFilesClicked="onFilesClicked"
          ></MessageItem>

          <v-btn class="my-2 mx-2" variant="flat" color="primary" :href="getChatLink()"> {{ pageContext.data.canWriteMessage ? 'Write a comment' : 'Join' }}</v-btn>

          <v-pagination
              v-model="pageContext.data.page"
              @update:modelValue="onClickPage"
              :length="pageContext.data.pagesCount"
              v-if="shouldShowPagination()"
              :total-visible="pageContext.data.pagesCount < 10 && !isMobile() ? 10 : undefined"
              variant="elevated"
              active-color="primary"
              density="comfortable"
          />
        </v-container>
    </template>
  </div>
  </v-container>
</template>

<script>
import MessageItem from "#root/common/components/MessageItem.vue";
import {hasLength, getLoginColoredStyle, PAGE_SIZE, PAGE_PARAM, onClickTrap, isStrippedUserLogin} from "#root/common/utils";
import {
  getHumanReadableDate,
} from "#root/common/date";
import {chat, messageIdHashPrefix, messageIdPrefix, profile} from "#root/common/router/routes";
import {usePageContext} from "#root/renderer/usePageContext.js";
import { navigate } from 'vike/client/router';
import bus, {OPEN_VIEW_FILES_DIALOG} from "#root/common/bus.js";

export default {
  setup() {
    const pageContext = usePageContext();

    // expose to template and other options API hooks
    return {
        pageContext
    }
  },
  methods: {
    getLoginColoredStyle,
    hasLength,
    isMobile() {
      return this.pageContext.isMobile
    },
    getProfileLink(user) {
      let url = profile + "/" + user.id;
      return url;
    },
    getChatMessageLink() {
      return chat + '/' + this.pageContext.data.blogDto.chatId + messageIdHashPrefix + this.pageContext.data.blogDto.messageId;
    },
    getChatLink() {
      return chat + '/' + this.pageContext.data.blogDto.chatId;
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

    onClickPage(e) {
      const url = new URL(window.location.href);
      url.searchParams.set(PAGE_PARAM, e);
      navigate(url.pathname + url.search);
    },
    shouldShowPagination() {
      return this.pageContext.data.count > PAGE_SIZE
    },

    onClickTrap(e) {
        onClickTrap(e)
    },
    onFilesClicked(item) {
      const obj = {
        chatId: this.pageContext.data.blogDto.chatId,
        messageId: item.id,
        fileItemUuid : item.fileItemUuid
      };
      bus.emit(OPEN_VIEW_FILES_DIALOG, obj);
    },
    onFilesClickedPostHeader() {
      const obj = {
        chatId: this.pageContext.data.blogDto.chatId,
        messageId: this.pageContext.data.blogDto.messageId,
        fileItemUuid : this.pageContext.data.blogDto.fileItemUuid
      };
      bus.emit(OPEN_VIEW_FILES_DIALOG, obj);
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
  },
  components: {
    MessageItem,
  },
  mounted() {
  },
  beforeUnmount() {
  },
}
</script>

<style lang="stylus" scoped>
@import "../../../../common/styles/constants.styl"
@import "../../../../common/styles/messageWrapper.styl"
@import "../../../../common/styles/itemAvatar.styl"

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
  border-style: solid;
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
