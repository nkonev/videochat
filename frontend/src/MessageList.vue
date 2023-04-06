<template>
    <div id="messagesScroller" style="overflow-y: auto; height: 100%" @scroll.passive="onScroll" v-on:keyup.esc="onCloseContextMenu()">
        <template v-if="isChatDtoLoaded()">
            <div v-if="pinnedPromoted" class="pinned-promoted">
                <v-alert
                    :key="pinnedPromotedKey"
                    dense
                    color="red lighten-2"
                    dark
                    dismissible
                    prominent
                >
                    <router-link :to="getPinnedRouteObject(pinnedPromoted)" style="text-decoration: none; color: white; cursor: pointer">
                        {{ pinnedPromoted.text }}
                    </router-link>
                </v-alert>

            </div>
            <v-list>
                <template v-for="(item, index) in items">
                    <MessageItem
                        :key="item.id"
                        :item="item"
                        :chatId="chatId"
                        :my="item.owner.id === currentUser.id"
                        :highlight="item.id == highlightMessageId"
                        :canResend="chatDto.canResend"
                        @contextmenu="onShowContextMenu($event, item)"
                        @deleteMessage="deleteMessage"
                        @editMessage="editMessage"
                        @replyOnMessage="replyOnMessage"
                        @shareMessage="shareMessage"
                        @onFilesClicked="onFilesClicked"
                    ></MessageItem>
                </template>
            </v-list>
            <MessageItemContextMenu
                ref="contextMenuRef"
                :canResend="chatDto.canResend"
                @deleteMessage="this.deleteMessage"
                @editMessage="this.editMessage"
                @replyOnMessage="this.replyOnMessage"
                @shareMessage="this.shareMessage"
                @onFilesClicked="this.onFilesClicked"
                @pinMessage="pinMessage"
                @removedFromPinned="removedFromPinned"
            />
            <infinite-loading :key="infinityKey" @infinite="infiniteHandler" :identifier="infiniteId" :direction="aDirection" force-use-infinite-wrapper="#messagesScroller" :distance="aDistance">
                <template slot="no-more"><span/></template>
                <template slot="no-results"><span/></template>
            </infinite-loading>
        </template>
    </div>

</template>

<script>
    import axios from "axios";
    import Vue from 'vue';
    import InfiniteLoading from 'vue-infinite-loading';
    import throttle from "lodash/throttle";
    import {mapGetters} from "vuex";
    import {GET_USER, UNSET_SEARCH_STRING} from "@/store";
    import bus, {
        CLOSE_SIMPLE_MODAL,
        MESSAGE_ADD,
        MESSAGE_DELETED, MESSAGE_EDITED, OPEN_EDIT_MESSAGE, OPEN_RESEND_TO_MODAL,
        OPEN_SIMPLE_MODAL, OPEN_VIEW_FILES_DIALOG,
        PINNED_MESSAGE_PROMOTED,
        PINNED_MESSAGE_UNPROMOTED, SET_EDIT_MESSAGE
    } from "@/bus";
    import queryMixin from "@/queryMixin";
    import {chat_name, messageIdHashPrefix, videochat_name} from "@/routes";
    import {
        embed_message_reply,
        findIndex,
        findIndexNonStrictly,
        hasLength,
        replaceInArray,
        setAnswerPreviewFields
    } from "@/utils";
    import MessageItem from "@/MessageItem";
    import MessageItemContextMenu from "@/MessageItemContextMenu";
    import debounce from "lodash/debounce";
    import cloneDeep from "lodash/cloneDeep";
    import Mark from "mark.js";

    const directionTop = 'top';
    const directionBottom = 'bottom';

    const maxItemsLength = 200;
    const reduceToLength = 100;

    const pageSize = 40;

    const scrollingThreshold = 200; // px


    export default {
        mixins: [
            queryMixin()
        ],
        props: ['chatDto'],
        data() {
            return {
                startingFromItemId: null,
                items: [],
                infiniteId: +new Date(),
                highlightMessageId: null,
                aDirection: directionTop,
                infinityKey: 1,
                scrollerDiv: null,
                markInstance: null,
                initialHash: null,

                scrollerProbeCurrent: 0,
                scrollerProbePrevious: 0,
                scrollerProbePreviousPrevious: 0,
                pinnedPromoted: null,
                pinnedPromotedKey: +new Date()
            }
        },

        methods: {
            addItem(dto) {
                console.log("Adding item", dto);
                this.items.push(dto);
                this.reduceListIfNeed();
                this.$forceUpdate();
            },
            changeItem(dto) {
                console.log("Replacing item", dto);
                replaceInArray(this.items, dto);
                this.$forceUpdate();
            },
            removeItem(dto) {
                console.log("Removing item", dto);
                const idxToRemove = findIndex(this.items, dto);
                this.items.splice(idxToRemove, 1);
                this.$forceUpdate();
            },

            onNewMessage(dto) {
                if (dto.chatId == this.chatId) {
                    const wasScrolled = this.isScrolledToBottom();
                    this.addItem(dto);
                    if (this.currentUser.id == dto.ownerId || wasScrolled) {
                        this.scrollDown();
                    }
                    this.performMarking();
                } else {
                    console.log("Skipping", dto)
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
                    const isScrolled = this.isScrolledToBottom();
                    this.changeItem(dto);
                    if (isScrolled) {
                        this.scrollDown();
                    }
                    this.performMarking();
                } else {
                    console.log("Skipping", dto)
                }
            },
            onClickScrollDown() {
                // condition is a dummy heuristic (because right now doe to outdated vue-infinite-loading we cannot scroll down several times. nevertheless I think it's a pretty good heuristic so I think it worth to remain it here after updating to vue 3 and another modern infinity scroller)
                if (this.items.length <= pageSize * 2 && !this.getRouteHash()) {
                    this.scrollDown();
                } else {
                    this.resetVariables();
                    this.reloadItems();
                }
                this.clearRouteHash();
                this.initialHash = null;
            },
            scrollDown() {
                Vue.nextTick(() => {
                    console.log("Scrolling down myDiv.scrollTop", this.scrollerDiv.scrollTop, "myDiv.scrollHeight", this.scrollerDiv.scrollHeight);
                    this.scrollerDiv.scrollTop = this.scrollerDiv.scrollHeight;
                });
            },
            isScrolledToBottom() {
                if (this.scrollerDiv) {
                    return this.scrollerDiv.scrollHeight - this.scrollerDiv.scrollTop - this.scrollerDiv.clientHeight < scrollingThreshold
                } else {
                    return false
                }
            },

            onScroll(e) {
                this.scrollerProbePreviousPrevious = this.scrollerProbePrevious;
                this.scrollerProbePrevious = this.scrollerProbeCurrent;
                this.scrollerProbeCurrent = this.scrollerDiv.scrollTop;
                console.debug("onScroll prevPrev=", this.scrollerProbePreviousPrevious , " prev=", this.scrollerProbePrevious, "cur=", this.scrollerProbeCurrent);

                this.trySwitchDirection();
            },
            trySwitchDirection() {
                if (this.scrollerProbeCurrent > this.scrollerProbePrevious && this.scrollerProbePrevious > this.scrollerProbePreviousPrevious && this.isTopDirection()) {
                    this.aDirection = directionBottom;
                    this.infinityKey++;
                    console.log("Infinity scrolling direction has been changed to bottom");
                } else if (this.scrollerProbePreviousPrevious > this.scrollerProbePrevious && this.scrollerProbePrevious > this.scrollerProbeCurrent && !this.isTopDirection()) {
                    this.aDirection = directionTop;
                    this.infinityKey++;
                    console.log("Infinity scrolling direction has been changed to top");
                } else {
                    console.log("Infinity scrolling direction has been remained untouched");
                }
            },
            isChatDtoLoaded() {
                return this.currentUser && this.chatDto.id
            },
            onPinnedMessagePromoted(item) {
                this.pinnedPromoted = item;
                this.pinnedPromotedKey++;
            },
            onPinnedMessageUnpromoted(item) {
                if (this.pinnedPromoted && this.pinnedPromoted.id == item.id) {
                    this.pinnedPromoted = null;
                }
            },
            keydownListener(e) {
                if (e.key === 'Escape') {
                    this.onCloseContextMenu()
                }
            },
            onCloseContextMenu(){
                if (this.$refs.contextMenuRef) {
                    this.$refs.contextMenuRef.onCloseContextMenu()
                }
            },
            infiniteHandler($state) {
                if (this.items.length) {
                    if (this.isTopDirection()) {
                        this.startingFromItemId = Math.min(...this.items.map(it => it.id));
                    } else {
                        this.startingFromItemId = Math.max(...this.items.map(it => it.id));
                    }
                    console.log("this.startingFromItemId set to", this.startingFromItemId);
                }

                if (!this.userIsSet) {
                    $state.complete();
                    return
                }

                axios.get(`/api/chat/${this.chatId}/message`, {
                    params: {
                        startingFromItemId: this.hasInitialHash ? this.highlightMessageId : this.startingFromItemId,
                        size: pageSize,
                        reverse: this.isTopDirection(),
                        searchString: this.searchString,
                        hasHash: this.hasInitialHash
                    },
                }).then(({data}) => {
                    const list = data;
                    if (list.length) {
                        if (this.isTopDirection()) {
                            this.items = list.reverse().concat(this.items);
                        } else {
                            this.items = this.items.concat(list);
                        }
                        if (this.items.length > pageSize) {
                            this.clearRouteHash();
                        }
                        this.reduceListIfNeed();
                        return true;
                    } else {
                        return false
                    }
                }).then(value => {
                    if (value) {
                        $state?.loaded();
                    } else {
                        $state?.complete();
                    }
                    if (this.hasInitialHash) {
                        try {
                            this.$vuetify.goTo('#' + this.initialHash, {container: this.scrollerDiv, duration: 0});
                        } catch (err) {
                            console.debug("Didn't scrolled", err)
                        }
                    }
                    if (hasLength(this.searchString)) {
                        this.markInstance.mark(this.searchString);
                    }
                    this.performMarking();
                    this.initialHash = null;
                })
            },
            performMarking() {
                Vue.nextTick(() => {
                    if (hasLength(this.searchString)) {
                        this.markInstance.unmark();
                        this.markInstance.mark(this.searchString);
                    }
                })
            },
            reduceListIfNeed() {
                if (this.items.length > maxItemsLength) {
                    setTimeout(() => {
                        console.log("Reducing to", maxItemsLength);
                        if (this.isTopDirection()) {
                            this.items = this.items.slice(0, reduceToLength);
                        } else {
                            this.items = this.items.slice(-reduceToLength);
                        }
                    }, 1);
                }
            },
            // not working until you will change this.items list
            reloadItems() {
                this.infiniteId += 1;
                console.log("Resetting infinite loader", this.infiniteId);
            },
            searchStringChanged(searchString) {
                this.resetVariables();
                this.reloadItems();
            },

            isTopDirection() {
                return this.aDirection === directionTop
            },
            onResizedListener() {
                const isScrolled = this.isScrolledToBottom();
                if (isScrolled) {
                    this.scrollDown();
                }
            },
            setHashVariables() {
                this.initialHash = this.getRouteHash();
                this.highlightMessageId = this.getMessageId(this.initialHash);
            },
            onShowContextMenu(e, menuableItem){
                this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
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
                const editMessageDto = cloneDeep(dto);
                if (dto.embedMessage?.id) {
                    setAnswerPreviewFields(editMessageDto, dto.embedMessage.text, dto.embedMessage.owner.login);
                }
                if (!this.isMobile()) {
                    bus.$emit(SET_EDIT_MESSAGE, editMessageDto);
                } else {
                    bus.$emit(OPEN_EDIT_MESSAGE, editMessageDto);
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
                    bus.$emit(SET_EDIT_MESSAGE, replyMessage);
                } else {
                    bus.$emit(OPEN_EDIT_MESSAGE, replyMessage);
                }
            },
            shareMessage(dto) {
                bus.$emit(OPEN_RESEND_TO_MODAL, dto)
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
            onFilesClicked(item) {
                bus.$emit(OPEN_VIEW_FILES_DIALOG, {chatId: this.chatId, fileItemUuid : item.fileItemUuid});
            },
            isVideoRoute() {
                return this.$route.name == videochat_name
            },
            getPinnedRouteObject(item) {
                const routeName = this.isVideoRoute() ? videochat_name : chat_name;
                return {name: routeName, params: {id: item.chatId}, hash: messageIdHashPrefix + item.id};
            },
            resetVariables() {
                this.aDirection = directionTop;
                this.items = [];
                this.startingFromItemId = null;
            },
            fetchPromotedMessage() {
                axios.get(`/api/chat/${this.chatId}/message/pin/promoted`).then((response) => {
                    if (response.status != 204) {
                        this.pinnedPromoted = response.data;
                    }
                });
            },
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}),
            chatId() {
                return this.$route.params.id
            },
            aDistance() {
                return this.isTopDirection() ? 0 : 100
            },
            userIsSet() {
                return !!this.currentUser
            },
            hasInitialHash() {
                return hasLength(this.initialHash)
            },
        },
        components: {
            InfiniteLoading,
            MessageItem,
            MessageItemContextMenu
        },
        watch: {
            '$route': {
                // reacts on user manually typing hash - in this case we may trigger reload if we don't have the necessary message
                handler: function(newRoute, oldRoute) {
                    console.debug("Watched on newRoute in MessageList", newRoute, " oldRoute", oldRoute);
                    if (newRoute.name === chat_name || newRoute.name === videochat_name) {
                        this.setHashVariables();
                        if (this.hasInitialHash) {
                            // resets variables about searching
                            this.$store.commit(UNSET_SEARCH_STRING); // UNSET_SEARCH_STRING - silently (w/o triggering subscr in queryMixing) - to prevent one extra loading if this has is aready in scope

                            if (findIndexNonStrictly(this.items, {id: this.highlightMessageId}) === -1) {
                                this.resetVariables();
                                this.reloadItems(); // resets hash in infiniteHandler
                            } else {
                                this.initialHash = null; // reset cached hash explicitly in order not to subsequently use it in case when we have hash in scrolled to bottom
                            }
                        }
                    }
                },
                immediate: true,
                deep: true
            },
        },
        created() {
            this.searchStringChanged = debounce(this.searchStringChanged, 700, {leading:false, trailing:true});
            this.onResizedListener = debounce(this.onResizedListener, 100, {leading:true, trailing:true});
            this.onScroll = throttle(this.onScroll, 400, {leading:true, trailing:true});

            this.initQueryAndWatcher();
            this.setHashVariables();
        },
        mounted() {
            this.scrollerDiv = document.getElementById("messagesScroller");
            this.markInstance = new Mark("div#messagesScroller .message-item-text");

            bus.$on(MESSAGE_ADD, this.onNewMessage);
            bus.$on(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$on(MESSAGE_EDITED, this.onEditMessage);
            bus.$on(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
            bus.$on(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);

            document.addEventListener("keydown", this.keydownListener);
            window.addEventListener('resize', this.onResizedListener);
        },
        beforeDestroy() {
            this.closeQueryWatcher();

            window.removeEventListener('resize', this.onResizedListener);
            document.removeEventListener("keydown", this.keydownListener);

            bus.$off(MESSAGE_ADD, this.onNewMessage);
            bus.$off(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$off(MESSAGE_EDITED, this.onEditMessage);
            bus.$off(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
            bus.$off(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);

            this.pinnedPromoted = null;
            this.pinnedPromotedKey = null;
        },

    }
</script>

<style scoped lang="stylus">
    #messagesScroller {
        overflow-y: scroll !important
        background  white
    }

    .pinned-promoted {
        position: absolute;
        z-index: 4;
        width: 100%
    }

</style>
