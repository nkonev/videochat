<template>
        <div class="ma-0 px-0 pt-0 pb-2 my-messages-scroller" @scroll.passive="onScroll">
          <div class="message-first-element" style="min-height: 1px; background: white"></div>
          <MessageItem v-for="item in items"
            :id="getItemId(item.id)"
            :key="item.id"
            :item="item"
            :chatId="chatId"
            :my="meIsOwnerOfMessage(item)"
            :highlight="item.id == highlightItemId"
            @customcontextmenu.stop="onShowContextMenu($event, item)"
            @deleteMessage="deleteMessage"
            @editMessage="editMessage"
            @replyOnMessage="replyOnMessage"
            @onFilesClicked="onFilesClicked"
            @onreactionclick="onExistingReactionClick"
          ></MessageItem>
          <template v-if="items.length == 0 && !showProgress">
            <v-sheet class="mx-2">{{$vuetify.locale.t('$vuetify.messages_not_found')}}</v-sheet>
          </template>

          <div class="message-last-element" style="min-height: 1px; background: white"></div>
          <MessageItemContextMenu
            ref="contextMenuRef"
            :canResend="canResend"
            :isBlog="blog"
            @deleteMessage="this.deleteMessage"
            @editMessage="this.editMessage"
            @replyOnMessage="this.replyOnMessage"
            @onFilesClicked="onFilesClicked"
            @showReadUsers="this.showReadUsers"
            @pinMessage="this.pinMessage"
            @removedFromPinned="this.removedFromPinned"
            @shareMessage="this.shareMessage"
            @makeBlogPost="makeBlogPost"
            @goToBlog="goToBlog"
            @addReaction="addReaction"
          />
        </div>

</template>

<script>
    import axios from "axios";
    import infiniteScrollMixin, {directionTop} from "@/mixins/infiniteScrollMixin";
    import {searchString, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";
    import bus, {
      CLOSE_SIMPLE_MODAL,
      LOGGED_OUT,
      MESSAGE_ADD,
      MESSAGE_DELETED,
      MESSAGE_EDITED,
      OPEN_EDIT_MESSAGE,
      OPEN_MESSAGE_READ_USERS_DIALOG,
      OPEN_RESEND_TO_MODAL,
      OPEN_SIMPLE_MODAL,
      OPEN_VIEW_FILES_DIALOG,
      PROFILE_SET, REFRESH_ON_WEBSOCKET_RESTORED,
      SCROLL_DOWN,
      SET_EDIT_MESSAGE, PARTICIPANT_CHANGED, OPEN_MESSAGE_EDIT_SMILEY, REACTION_CHANGED, REACTION_REMOVED
    } from "@/bus/bus";
    import {
      deepCopy, edit_message, embed_message_reply,
      findIndex, getBlogLink,
      hasLength, haveEmbed, isChatRoute,
      replaceInArray,
      replaceOrAppend,
      replaceOrPrepend, reply_message,
      setAnswerPreviewFields
    } from "@/utils";
    import debounce from "lodash/debounce";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import MessageItem from "@/MessageItem.vue";
    import MessageItemContextMenu from "@/MessageItemContextMenu.vue";
    import {messageIdHashPrefix, messageIdPrefix} from "@/router/routes";
    import {getTopMessagePosition, removeTopMessagePosition, setTopMessagePosition} from "@/store/localStore";
    import Mark from "mark.js";
    import hashMixin from "@/mixins/hashMixin";

    const PAGE_SIZE = 40;
    const SCROLLING_THRESHHOLD = 200; // px

    const scrollerName = 'MessageList';

    export default {
      mixins: [
        infiniteScrollMixin(scrollerName),
        hashMixin(),
        searchString(SEARCH_MODE_MESSAGES),
      ],
      props: ['canResend', 'blog'],
      data() {
        return {
          markInstance: null,
        }
      },

      computed: {
        ...mapStores(useChatStore),
        chatId() {
          return this.$route.params.id
        },
        showProgress() {
          return this.chatStore.progressCount > 0
        },
      },

      components: {
          MessageItemContextMenu,
          MessageItem
      },

      methods: {
        addItem(dto) {
          console.log("Adding item", dto);
          this.items.unshift(dto);
          this.reduceListIfNeed();
          this.updateTopAndBottomIds();
        },
        changeItem(dto) {
          console.log("Replacing item", dto);
          replaceInArray(this.items, dto);
          this.updateTopAndBottomIds();
        },
        removeItem(dto) {
          console.log("Removing item", dto);
          const idxToRemove = findIndex(this.items, dto);
          this.items.splice(idxToRemove, 1);
          this.updateTopAndBottomIds();
        },

        onNewMessage(dto) {
          const chatIdsAreEqual = dto.chatId == this.chatId;
          const isScrolledToBottom = this.isScrolledToBottom();
          const emptySearchString = !hasLength(this.searchString);
          if (chatIdsAreEqual && isScrolledToBottom && emptySearchString) {
            this.addItem(dto);
            this.performMarking();
          } else if (chatIdsAreEqual && isScrolledToBottom) { // not empty searchString
              axios.put(`/api/chat/${this.chatId}/message/filter`, {
                  searchString: this.searchString,
                  messageId: dto.id
              }).then(({data}) => {
                  if (data.found) {
                      this.addItem(dto);
                      this.performMarking();
                  }
              })
          } else {
            console.log("Skipping", dto, chatIdsAreEqual, isScrolledToBottom, emptySearchString)
          }
        },
        onDeleteMessage(dto) {
          if (dto.chatId == this.chatId) {
            this.removeItem(dto);
          } else {
            console.log("Skipping", dto)
          }
        },
        onEditMessage(dto) {
          if (dto.chatId == this.chatId) {
            this.changeItem(dto);
            this.performMarking();
          } else {
            console.log("Skipping", dto)
          }
        },

        getMaxItemsLength() {
            return 200
        },
        getReduceToLength() {
            return 100
        },
        reduceBottom() {
          this.items = this.items.slice(-this.getReduceToLength());
          this.startingFromItemIdBottom = this.getMaximumItemId();
        },
        reduceTop() {
          this.items = this.items.slice(0, this.getReduceToLength());
          this.startingFromItemIdTop = this.getMinimumItemId();
        },
        saveScroll(top) {
          this.preservedScroll = top ? this.getMinimumItemId() : this.getMaximumItemId();
          console.log("Saved scroll", this.preservedScroll, "in ", scrollerName);
        },
        initialDirection() {
          return directionTop
        },
        async onFirstLoad(loadedResult) {
            await this.doScrollOnFirstLoad(messageIdHashPrefix);
            if (loadedResult === true) {
                removeTopMessagePosition(this.chatId);
            }
        },
        getPositionFromStore() {
              return getTopMessagePosition(this.chatId)
        },
        async doDefaultScroll() {
              await this.scrollDown(); // we need it to prevent browser's scrolling
              this.loadedBottom = true;
        },

        async load() {
          if (!this.canDrawMessages()) {
              return Promise.resolve()
          }

          this.chatStore.incrementProgressCount();

          const { startingFromItemId, hasHash } = this.prepareHashesForLoad();

          return axios.get(`/api/chat/${this.chatId}/message`, {
            params: {
              startingFromItemId: startingFromItemId,
              size: PAGE_SIZE,
              reverse: this.isTopDirection(),
              searchString: this.searchString,
              hasHash: hasHash
            },
          })
          .then((response) => {
            if (response.status == 204) {
              // do nothing because we 're going to exit from ChatView.MessageList to ChatList inside ChatView itself
              return Promise.resolve()
            }

            const items = response.data;
            console.log("Get items in ", scrollerName, items, "page", this.startingFromItemIdTop, this.startingFromItemIdBottom, "chosen", startingFromItemId);

            if (this.isTopDirection()) {
              replaceOrAppend(this.items, items);
            } else {
              replaceOrPrepend(this.items, items);
            }

            if (items.length < PAGE_SIZE) {
              if (this.isTopDirection()) {
                this.loadedTop = true;
              } else {
                this.loadedBottom = true;
              }
            }
            this.updateTopAndBottomIds();

            if (!this.isFirstLoad) {
              this.clearRouteHash()
            }
            this.performMarking();
            return Promise.resolve(true)
          }).finally(()=>{
              this.chatStore.decrementProgressCount();
          })
        },
        afterScrollRestored(el) {
            el?.parentElement?.scrollBy({
              top: !this.isTopDirection() ? 14 : -20,
              behavior: "instant",
            });
        },
        bottomElementSelector() {
          return ".message-first-element"
        },
        topElementSelector() {
          return ".message-last-element"
        },

        getItemId(id) {
          return messageIdPrefix + id
        },

        async scrollDown() {
          return await this.$nextTick(() => {
            this.scrollerDiv.scrollTop = 0;
          });
        },
        scrollerSelector() {
          return ".my-messages-scroller"
        },

        reset() {
          this.resetInfiniteScrollVars();
          this.chatStore.showScrollDown = false;

          this.startingFromItemIdTop = null;
          this.startingFromItemIdBottom = null;
        },
        async onSearchStringChangedDebounced() {
          await this.onSearchStringChanged()
        },
        async onSearchStringChanged() {
          if (!hasLength(this.highlightItemId)) { // if is required for case
            // user searched for some text ("telegram", or "www")
            // then in one of found messages user clicks on the original of the answered (which jumps to th original)
            // without this fix because of two events (a. search string changed (see in the search mixin), b. route changed (see here in watch))
            // the message list is loaded 2 times and as a result the second load resets both scrolling and highlighting)
            await this.reloadItems();
          }
        },
        async onProfileSet() {
          await this.setHashAndReloadItems();
        },
        onLoggedOut() {
          this.reset();
          this.beforeUnload();
        },
        canDrawMessages() {
          return !!this.chatStore.currentUser
        },

        async onScrollDownButton() {
          this.clearRouteHash();
          await this.reloadItems();
        },

        onScrollCallback() {
          this.chatStore.showScrollDown = !this.isScrolledToBottom();
          if (this.chatStore.showScrollDown) {
            // during scrolling we disable adding new elements, so some messages can appear on server, so
            // we set loadedBottom to false in order to force infiniteScrollMixin to fetch new messages during scrollBottom()
            this.loadedBottom = false;
            this.loadedTop = false;
          }
        },
        isScrolledToBottom() {
          if (this.scrollerDiv) {
            return Math.abs(this.scrollerDiv.scrollTop) < SCROLLING_THRESHHOLD
          } else {
            return false
          }
        },
        updateTopAndBottomIds() {
          this.startingFromItemIdTop = this.getMinimumItemId();
          this.startingFromItemIdBottom = this.getMaximumItemId();
        },
        saveLastVisibleElement(chatId) {
          if (!this.isScrolledToBottom()) {
            const elems = [...document.querySelectorAll(this.scrollerSelector() + " .message-item-root")].map((item) => {
              const visible = item.getBoundingClientRect().top > 0
              return {item, visible}
            });

            const visible = elems.filter((el) => el.visible);
            // console.log("visible", visible, "elems", elems);
            if (visible.length == 0) {
              console.warn("Unable to get top visible")
              return
            }
            const topVisible = visible[visible.length - 1].item

            const mid = this.getIdFromRouteHash(topVisible.id);
            console.log("Found topVisible", topVisible, "in chat", chatId, "messageId", mid);

            setTopMessagePosition(chatId, mid)
          } else {
            console.log("Skipped saved topVisible because we are already scrolled to the bottom ")
          }
        },
        beforeUnload() {
          this.saveLastVisibleElement(this.chatId);
        },
        performMarking() {
          this.$nextTick(() => {
            if (hasLength(this.searchString)) {
              this.markInstance.unmark();
              this.markInstance.mark(this.searchString);
            }
          })
        },

        deleteMessage(dto){
          bus.emit(OPEN_SIMPLE_MODAL, {
            buttonName: this.$vuetify.locale.t('$vuetify.delete_btn'),
            title: this.$vuetify.locale.t('$vuetify.delete_message_title', dto.id),
            text:  this.$vuetify.locale.t('$vuetify.delete_message_text'),
            actionFunction: (that)=> {
              that.loading = true;
              axios.delete(`/api/chat/${this.chatId}/message/${dto.id}`)
                .then(() => {
                  bus.emit(CLOSE_SIMPLE_MODAL);
                })
                .finally(()=>{
                  that.loading = false;
                })
            }
          });
        },
        editMessage(dto){
          const editMessageDto = deepCopy(dto);
          if (haveEmbed(dto)) {
            setAnswerPreviewFields(editMessageDto, dto.embedMessage.text, dto.embedMessage.owner.login);
          }
          if (!this.isMobile()) {
            bus.emit(SET_EDIT_MESSAGE, {dto: editMessageDto, actionType: edit_message});
          } else {
            bus.emit(OPEN_EDIT_MESSAGE, {dto: editMessageDto, actionType: edit_message});
          }
        },
        replyOnMessage(dto) {
          const replyMessage = {
              embedMessage: {
                  id: dto.id,
                  embedType: embed_message_reply
              },
          };
          setAnswerPreviewFields(replyMessage, dto.text, dto.owner.login);
          if (!this.isMobile()) {
              bus.emit(SET_EDIT_MESSAGE, {dto: replyMessage, actionType: reply_message});
          } else {
              bus.emit(OPEN_EDIT_MESSAGE, {dto: replyMessage, actionType: reply_message});
          }
        },
        onFilesClicked(item) {
          const obj = {chatId: this.chatId, fileItemUuid : item.fileItemUuid};
          if (this.meIsOwnerOfMessage(item)) {
            obj.messageIdToDetachFiles = item.id;
          }
          bus.emit(OPEN_VIEW_FILES_DIALOG, obj);
        },
        meIsOwnerOfMessage(item) {
          return item.owner?.id === this.chatStore.currentUser?.id;
        },
        showReadUsers(dto) {
          bus.emit(OPEN_MESSAGE_READ_USERS_DIALOG, {chatId: dto.chatId, messageId: dto.id})
        },
        pinMessage(dto) {
          axios.put(`/api/chat/${this.chatId}/message/${dto.id}/pin`, null, {
            params: {
              pin: true
            },
          });
        },
        removedFromPinned(dto) {
          axios.put(`/api/chat/${this.chatId}/message/${dto.id}/pin`, null, {
            params: {
              pin: false
            },
          });
        },
        shareMessage(dto) {
          bus.emit(OPEN_RESEND_TO_MODAL, dto)
        },
        onExistingReactionClick(dto) {
          axios.put(`/api/chat/${this.chatId}/message/${dto.id}/reaction`, {
            reaction: dto.reaction,
          })
        },
        addReaction(dto) {
          bus.emit(OPEN_MESSAGE_EDIT_SMILEY,
            {
              addSmileyCallback: (smiley) => {
                axios.put(`/api/chat/${this.chatId}/message/${dto.id}/reaction`, {
                  reaction: smiley,
                })
              },
              title: this.$vuetify.locale.t('$vuetify.add_reaction_on_message')
            }
          );
        },
        onShowContextMenu(e, menuableItem){
          // console.log("onShowContextMenu", e, tag, tagParent);
          if (
            !this.checkUpByTree(e?.target, 1, (el) => el?.tagName?.toLowerCase() == "img") &&
            !this.checkUpByTree(e?.target, 1, (el) => el?.tagName?.toLowerCase() == "video") &&
            !this.checkUpByTree(e?.target, 1, (el) => el?.tagName?.toLowerCase() == "a") &&
            !this.checkUpByTree(e?.target, 3, (el) => el?.classList?.contains("reactions"))
          ) {
            this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
          }
        },
        checkUpByTree(el, maxLevels, condition) {
          let level = 0;
          let underCheck = el;
          do {
            if (condition(underCheck)) {
              return true
            }
            underCheck = underCheck.parentElement;
            level++;
          } while (level <= maxLevels);
          return false;
        },
        onUserProfileChanged(user) {
          this.items.forEach(item => {
            if (item.owner.id == user.id) {
              item.owner = user;
            }
          });
        },
        getBlogLink() {
          return getBlogLink(this.chatId);
        },
        makeBlogPost(dto) {
          axios.put(`/api/chat/${this.chatId}/message/${dto.id}/blog-post`);
        },
        goToBlog() {
          window.location.href = this.getBlogLink();
        },
        onWsRestoredRefresh() {
          this.onSearchStringChanged();
        },
        onReactionChanged(dto) {
          const foundMessage = this.items.find(item => item.id == dto.messageId);
          if (foundMessage) {
            const foundReaction = foundMessage.reactions.find(reaction => reaction.reaction == dto.reaction.reaction);
            if (foundReaction) {
              foundReaction.count = dto.reaction.count;
              foundReaction.users = dto.reaction.users;
            } else {
              foundMessage.reactions.push(dto.reaction)
            }
          }
        },
        onReactionRemoved(dto) {
          const foundMessage = this.items.find(item => item.id == dto.messageId);
          if (foundMessage) {
            foundMessage.reactions = foundMessage.reactions.filter(reaction => reaction.reaction != dto.reaction.reaction);
          }
        },
      },
      created() {
        this.onSearchStringChangedDebounced = debounce(this.onSearchStringChangedDebounced, 700, {leading:false, trailing:true})
      },

      watch: {
          // We use the same handler in order to fix resetting of message highlight when we click on resent message
          // Reproduction:
          // 1. Open chat A
          // 2. Resend message to chat B
          // 3. Open chat B
          // 4. Click on user login of the last resent message
          // 5. It should move you to chat A
          // 6. Without this single handler, both handlers would invoke what leads us to resetting yellow highlight
          '$route': {
            handler: async function (newValue, oldValue) {
              if (newValue.params.id != oldValue.params.id) {
                // save the top message id always, including exiting case, e.g. to the Welcome page
                console.debug("Chat id has been changed", oldValue.params.id, "->", newValue.params.id);
                this.saveLastVisibleElement(oldValue.params.id);

                // reaction on switching chat at left
                if (isChatRoute(newValue) && hasLength(newValue.params.id)) { // filtering out the case when we go to profile - it also has route id
                  await this.onProfileSet();
                  return
                }
              }

              // reaction on setting hash
              if (isChatRoute(newValue)) {
                // hash
                if (hasLength(newValue.hash)) {
                  console.log("Changed route hash, going to scroll", newValue.hash)
                  await this.scrollToOrLoad(newValue.hash);
                  return
                }
              }

              // reaction on changing query
              const newQuery = newValue.query[SEARCH_MODE_MESSAGES];
              const oldQuery = oldValue.query[SEARCH_MODE_MESSAGES];
              if (newQuery != oldQuery) {
                this.onSearchStringChangedDebounced();
                return
              }
            }
          }
      },

      async mounted() {
        this.markInstance = new Mark(this.scrollerSelector() + " .message-item-text");

        // we trigger actions on load if profile was set
        // else we rely on PROFILE_SET
        // should be before bus.on(PROFILE_SET, this.onProfileSet);
        if (this.canDrawMessages()) {
          await this.onProfileSet();
        }

        addEventListener("beforeunload", this.beforeUnload);

        bus.on(PROFILE_SET, this.onProfileSet);
        bus.on(LOGGED_OUT, this.onLoggedOut);
        bus.on(SCROLL_DOWN, this.onScrollDownButton);
        bus.on(MESSAGE_ADD, this.onNewMessage);
        bus.on(MESSAGE_DELETED, this.onDeleteMessage);
        bus.on(MESSAGE_EDITED, this.onEditMessage);
        bus.on(REACTION_CHANGED, this.onReactionChanged);
        bus.on(REACTION_REMOVED, this.onReactionRemoved);
        bus.on(PARTICIPANT_CHANGED, this.onUserProfileChanged);
        bus.on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);

        this.chatStore.searchType = SEARCH_MODE_MESSAGES;
      },

      beforeUnmount() {
        this.markInstance.unmark();
        this.markInstance = null;
        removeEventListener("beforeunload", this.beforeUnload);

        this.uninstallScroller();
        bus.off(MESSAGE_ADD, this.onNewMessage);
        bus.off(MESSAGE_DELETED, this.onDeleteMessage);
        bus.off(MESSAGE_EDITED, this.onEditMessage);
        bus.off(REACTION_CHANGED, this.onReactionChanged);
        bus.off(REACTION_REMOVED, this.onReactionRemoved);
        bus.off(PROFILE_SET, this.onProfileSet);
        bus.off(LOGGED_OUT, this.onLoggedOut);
        bus.off(SCROLL_DOWN, this.onScrollDownButton);
        bus.off(PARTICIPANT_CHANGED, this.onUserProfileChanged);
        bus.off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
      }
    }
</script>

<style lang="stylus">
    .my-messages-scroller {
      height 100%
      width: 100%
      overflow-y scroll !important
      display flex
      flex-direction column-reverse
      background white
    }

</style>
