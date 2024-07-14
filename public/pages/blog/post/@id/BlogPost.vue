<template>
  <v-container class="ma-0 pa-0 my-list-container" fluid>
  <h1 v-if="blogDto.is404">404 Not found</h1>
  <div v-else class="my-messages-scroller">
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
              <div class="author-and-date" v-if="blogDto.owner">
                <v-list-item-title><a class="nodecorated-link" :style="getLoginColoredStyle(blogDto.owner, true)" :href="getProfileLink(blogDto.owner)">{{blogDto.owner.login}}</a></v-list-item-title>
                <v-list-item-subtitle>{{getDate(blogDto.createDateTime)}}</v-list-item-subtitle>
              </div>
              <div class="ma-0 pa-0 go-to-chat">
                <v-btn variant="plain" rounded size="large" :href="getChatMessageLink()" title="Go to the message in chat"><v-icon size="large">mdi-forum</v-icon></v-btn>
              </div>
            </div>
          </template>
        </v-list-item>

        <div class="pa-0 ma-0 mt-1 message-item-wrapper post-content" @click="onClickTrap">
          <v-container v-html="blogDto.text" class="message-item-text ml-0"></v-container>
          <div class="mt-0 ml-2 mr-4 reactions" v-if="shouldShowReactions(blogDto)">
            <v-btn v-for="(reaction, i) in blogDto.reactions" variant="tonal" size="small" height="32px" rounded :class="reactionClass(i)" :title="getReactedUsers(reaction)"><span v-if="reaction.count > 1" class="text-body-2 with-space">{{ '' + reaction.count + ' ' }}</span><span class="text-h6">{{ reaction.reaction }}</span></v-btn>
          </div>
        </div>
      </div>
    </div>

    <template v-if="blogDto.messageId">
        <v-container class="ma-0 pa-0 mb-2" fluid>
          <MessageItem v-for="(item, index) in items" v-if="!commentsLoading"
            :id="getItemId(item.id)"
            :key="item.id"
            :item="item"
            :chatId="item.chatId"
            :isInBlog="true"
            @click="onClickTrap"
          ></MessageItem>

          <v-progress-linear
            class="my-2"
            v-else
            color="primary"
            indeterminate
          ></v-progress-linear>

          <v-btn class="mt-2 mx-2" variant="flat" color="primary" :href="getChatLink()">Write a comment</v-btn>

          <v-pagination v-model="page" @update:modelValue="onClickPage" :length="pagesCount" v-if="shouldShowPagination()"/>
        </v-container>
    </template>
  </div>
  </v-container>
</template>

<script>
import axios from "axios";
import MessageItem from "#root/common/components/MessageItem.vue";
import {getHumanReadableDate, hasLength, getLoginColoredStyle, PAGE_SIZE, PAGE_PARAM, checkUpByTreeObj} from "#root/common/utils";
import {chat, messageIdHashPrefix, messageIdPrefix, profile} from "#root/common/router/routes";
import {usePageContext} from "#root/renderer/usePageContext.js";
import { navigate } from 'vike/client/router';
import bus, {
    PLAYER_MODAL,
} from "#root/common/bus";

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
      return chat + '/' + this.blogDto.chatId + messageIdHashPrefix + this.blogDto.messageId;
    },
    getChatLink() {
      return chat + '/' + this.blogDto.chatId;
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
      this.page = e;

      let actualPage = e - 1;

      const url = new URL(window.location.href);
      url.searchParams.set(PAGE_PARAM, e);
      navigate(url.pathname + url.search);
      this.loadComments(actualPage)
    },
    shouldShowPagination() {
      return this.count > PAGE_SIZE
    },

    loadComments(page) {
        this.commentsLoading = true;
        axios.get(`/api/blog/${this.blogDto.chatId}/comment`, {
            params: {
                page: page,
                size: PAGE_SIZE,
                reverse: false,
            },
        }).then((res) => {
            this.items = res.data.items;
            // this.page = res.data.page;
            this.pagesCount = res.data.pagesCount;
            this.count = res.data.count;
        }).finally(()=>{
            this.commentsLoading = false;
        })
    },

    onClickTrap(e) {
        const foundElements = [
            checkUpByTreeObj(e?.target, 1, (el) => {
                return el?.tagName?.toLowerCase() == "img" ||
                    Array.from(el?.children).find(ch => ch?.classList?.contains("video-in-message-button"))
            })
        ].filter(r => r.found);
        if (foundElements.length) {
            const found = foundElements[foundElements.length - 1].el;
            switch (found?.tagName?.toLowerCase()) {
                case "img": {
                    bus.emit(PLAYER_MODAL, {canShowAsImage: true, url: found.src})
                    break;
                }
                case "div": { // contains video
                    const video = Array.from(found?.children).find(ch => ch?.tagName?.toLowerCase() == "video");
                    bus.emit(PLAYER_MODAL, {canPlayAsVideo: true, url: video.src, previewUrl: video.poster})
                    break;
                }
            }
        }
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
@import "../../../../common/styles/common.styl"
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
