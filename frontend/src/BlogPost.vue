<template>
  <v-container class="ma-0 pa-0" :style="heightWithoutAppBar" fluid>
  <div class="my-messages-scroller" @scroll.passive="onScroll">
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
                <v-btn variant="plain" rounded size="large" :href="getChatLink()" @click.prevent="toChat()" :title="$vuetify.locale.t('$vuetify.go_to_chat')"><v-icon size="large">mdi-forum</v-icon></v-btn>
              </div>
            </div>
          </template>
        </v-list-item>

        <div class="pa-0 ma-0 mt-1 message-item-wrapper post-content">
          <v-container v-html="blogDto.text" class="message-item-text ml-0"></v-container>
          <div class="mt-0 ml-2 mr-4 reactions" v-if="shouldShowReactions(blogDto)">
            <v-btn v-for="(reaction, i) in blogDto.reactions" variant="tonal" size="small" height="32px" rounded :class="reactionClass(i)" @click="onExistingReactionClick(reaction.reaction)" :title="getReactedUsers(reaction)"><span v-if="reaction.count > 1" class="text-body-2 with-space">{{ '' + reaction.count + ' ' }}</span><span class="text-h6">{{ reaction.reaction }}</span></v-btn>
          </div>
        </div>
      </div>
    </div>

    <template v-if="blogDto.messageId">
        <div class="message-first-element" style="min-height: 1px; background: white"></div>
        <v-container class="ma-0 pa-0 mb-2" fluid>
          <MessageItem v-for="(item, index) in items"
            :id="getItemId(item.id)"
            :key="item.id"
            :item="item"
            :chatId="item.chatId"
            :isInBlog="true"
            @onreactionclick="onReactionClick"
          ></MessageItem>
        </v-container>
        <div class="message-last-element" style="min-height: 1px; background: white"></div>
    </template>
  </div>
  </v-container>
</template>

<script>
import axios from "axios";
import MessageItem from "@/MessageItem";
import {getHumanReadableDate, hasLength, replaceOrAppend, replaceOrPrepend, setTitle} from "@/utils";
import {chat, messageIdHashPrefix, messageIdPrefix, profile, profile_name} from "@/router/routes";
import {mapStores} from "pinia";
import {useBlogStore} from "@/store/blogStore";
import infiniteScrollMixin, {directionBottom, directionTop} from "@/mixins/infiniteScrollMixin";
import {removeTopMessagePosition} from "@/store/localStore";
import heightMixin from "@/mixins/heightMixin";

const PAGE_SIZE = 40;

const scrollerName = 'CommentList';

const blogDtoFactory = () => {
  return {
    chatId: 0
  }
}

export default {
  mixins: [
    heightMixin(),
    infiniteScrollMixin(scrollerName),
  ],
  data() {
    return {
      blogDto: blogDtoFactory(),

      startingFromItemIdTop: null,
      startingFromItemIdBottom: null,
      startingFromItemId: null,
    }
  },
  methods: {
    hasLength,
    getProfileLink(user) {
      let url = profile + "/" + user.id;
      return url;
    },
    getChatLink() {
      return chat + '/' + this.blogDto.chatId + messageIdHashPrefix + this.blogDto.messageId;
    },
    toChat() {
      window.location.href = this.getChatLink();
    },
    getBlog(id) {
      this.blogStore.incrementProgressCount();
      return axios.get('/api/blog/'+id).then(({data}) => {
        this.blogDto = data;
        this.startingFromItemId = data.messageId;
        setTitle(this.blogDto.title);
      }).finally(()=>{
        this.blogStore.decrementProgressCount();
      });
    },
    getDate(date) {
      if (hasLength(date)) {
        return getHumanReadableDate(date)
      } else {
        return null
      }
    },
    getMaxItemsLength() {
      return 200
    },
    getReduceToLength() {
      return 100
    },
    getMaximumItemId() {
      return this.items.length ? Math.max(...this.items.map(it => it.id)) : null
    },
    getMinimumItemId() {
      return this.items.length ? Math.min(...this.items.map(it => it.id)) : null
    },
    reduceBottom() {
      this.items = this.items.slice(0, this.getReduceToLength());
      this.startingFromItemIdBottom = this.getMaximumItemId();
    },
    reduceTop() {
      this.items = this.items.slice(-this.getReduceToLength());
      this.startingFromItemIdTop = this.getMinimumItemId();
    },
    saveScroll(top) {
      this.preservedScroll = top ? this.getMinimumItemId() : this.getMaximumItemId();
      console.log("Saved scroll", this.preservedScroll, "in ", scrollerName);
    },
    initialDirection() {
      return directionBottom
    },
    async onFirstLoad() {
      this.loadedTop = true;
    },
    async load() {
      if (!this.canDrawMessages()) {
        return Promise.resolve()
      }

      this.blogStore.incrementProgressCount();
      let startingFromItemId;
      if (this.startingFromItemId) {
        startingFromItemId = this.startingFromItemId;
        this.startingFromItemId = null;
      } else {
        startingFromItemId = this.isTopDirection() ? this.startingFromItemIdTop : this.startingFromItemIdBottom;
      }

      return axios.get(`/api/blog/${this.$route.params.id}/comment`, {
        params: {
          startingFromItemId: startingFromItemId,
          size: PAGE_SIZE,
          reverse: this.isTopDirection(),
        },
      })
        .then((res) => {
          const items = res.data;
          console.log("Get items in ", scrollerName, items, "page", this.startingFromItemIdTop, this.startingFromItemIdBottom, "chosen", startingFromItemId);

          if (this.isTopDirection()) {
            replaceOrPrepend(this.items, items);
          } else {
            replaceOrAppend(this.items, items);
          }

          if (items.length < PAGE_SIZE) {
            if (this.isTopDirection()) {
              //console.log("Setting this.loadedTop");
              this.loadedTop = true;
            } else {
              //console.log("Setting this.loadedBottom");
              this.loadedBottom = true;
            }
          }
          this.updateTopAndBottomIds();

        }).finally(()=>{
          this.blogStore.decrementProgressCount();
        })
    },
    updateTopAndBottomIds() {
      this.startingFromItemIdTop = this.getMinimumItemId();
      this.startingFromItemIdBottom = this.getMaximumItemId();
    },
    bottomElementSelector() {
      return ".message-last-element"
    },
    topElementSelector() {
      return ".message-first-element"
    },

    getItemId(id) {
      return messageIdPrefix + id
    },
    scrollerSelector() {
      return ".my-messages-scroller"
    },
    reset() {
      this.resetInfiniteScrollVars();
      this.blogStore.showScrollDown = false;

      this.startingFromItemIdTop = null;
      this.startingFromItemIdBottom = null;
    },
    canDrawMessages() {
      return true
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
    onExistingReactionClick(reaction) {
        this.onReactionClick({id: this.blogDto.messageId, reaction: reaction})
    },
    onReactionClick(obj) {
        const aRoute = chat + '/' + this.blogDto.chatId + messageIdHashPrefix + obj.id;
        window.location.href = aRoute;
    },
  },
  components: {
    MessageItem,
  },
  computed: {
    ...mapStores(useBlogStore),
  },
  async mounted() {
    return this.getBlog(this.$route.params.id).then(async ()=>{
      await this.reloadItems();
    });
  },
  beforeUnmount() {
    this.blogDto = blogDtoFactory();
    this.uninstallScroller();
  },
}
</script>

<style lang="stylus" scoped>
@import "common.styl"
@import "messageWrapper.styl"
@import "itemAvatar.styl"

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
