<template>
    <v-container class="ma-0 pa-0" style="height: 100%" fluid>
        <v-list v-if="items.length">
            <v-list-item-group v-model="group" color="primary" id="chat-list-items">
            <v-list-item @keydown.esc="onCloseContextMenu()"
                    v-for="(item, index) in items"
                    :key="item.id"
                    @contextmenu="onShowContextMenu($event, item)"
                    @click.prevent="openChat(item)"
                    :href="getLink(item)"
            >
                <v-badge
                    v-if="item.avatar"
                    color="success accent-4"
                    dot
                    bottom
                    overlap
                    bordered
                    :value="item.online"
                >
                    <v-list-item-avatar class="ma-0 pa-0">
                        <img :src="item.avatar"/>
                    </v-list-item-avatar>
                </v-badge>
                <v-list-item-content :id="'chat-item-' + item.id" :class="item.avatar ? 'ml-4' : ''">
                    <v-list-item-title>
                        <span class="min-height">
                            {{getChatName(item)}}
                        </span>
                        <v-badge v-if="item.unreadMessages" inline :content="item.unreadMessages" class="mt-0"></v-badge>
                        <v-badge v-if="item.videoChatUsersCount" color="success" icon="mdi-phone" inline  class="mt-0"/>
                    </v-list-item-title>
                    <v-list-item-subtitle v-html="printParticipants(item)"></v-list-item-subtitle>
                </v-list-item-content>
                <v-list-item-action v-if="!isMobile()">
                    <v-container class="mb-0 mt-0 pl-0 pr-0 pb-0 pt-0">
                        <v-btn v-if="item.canEdit" icon color="primary" @click.stop="editChat(item)" :title="$vuetify.lang.t('$vuetify.edit_chat')"><v-icon dark>mdi-lead-pencil</v-icon></v-btn>
                        <v-btn v-if="item.canDelete" icon @click.stop="deleteChat(item)" :title="$vuetify.lang.t('$vuetify.delete_chat')" color="error"><v-icon dark>mdi-delete</v-icon></v-btn>
                        <v-btn v-if="item.canLeave" icon @click.stop="leaveChat(item)" :title="$vuetify.lang.t('$vuetify.leave_chat')"><v-icon dark>mdi-exit-run</v-icon></v-btn>
                    </v-container>
                </v-list-item-action>
            </v-list-item>
            </v-list-item-group>
        </v-list>
        <ChatListContextMenu ref="contextMenuRef" @editChat="this.editChat" @deleteChat="this.deleteChat" @leaveChat="this.leaveChat"/>
        <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId">
            <template slot="no-more"><span/></template>
            <template slot="no-results"><span/></template>
        </infinite-loading>

        <v-container fill-height fluid v-if="shouldShowWelcome()" :style="$vuetify.breakpoint.lgAndUp ? 'max-width: 420px' : ''">
            <v-row align="center" justify="center">
                <v-col>
                    <v-card>
                        <v-card-title class="d-flex justify-space-around">{{$vuetify.lang.t('$vuetify.welcome_participant', currentUser.login)}}</v-card-title>
                        <v-card-actions  class="d-flex justify-space-around">
                            <v-btn color="primary" @click="createChat()" text>
                                <v-icon>mdi-plus</v-icon>
                                {{ $vuetify.lang.t('$vuetify.new_chat') }}
                            </v-btn>
                            <v-btn @click="findUser()" text>
                                <v-icon>mdi-magnify</v-icon>
                                {{ $vuetify.lang.t('$vuetify.find_user') }}
                            </v-btn>
                        </v-card-actions>
                    </v-card>

                </v-col>
            </v-row>
        </v-container>
    </v-container>

</template>

<script>
    import bus, {
        CHAT_ADD,
        CHAT_EDITED,
        CHAT_DELETED,
        OPEN_CHAT_EDIT,
        OPEN_SIMPLE_MODAL,
        UNREAD_MESSAGES_CHANGED,
        USER_PROFILE_CHANGED,
        CLOSE_SIMPLE_MODAL,
        REFRESH_ON_WEBSOCKET_RESTORED,
        VIDEO_CALL_USER_COUNT_CHANGED, LOGGED_OUT, PROFILE_SET, OPEN_FIND_USER
    } from "./bus";
    import {chat, chat_name} from "./routes";
    import InfiniteLoading from 'vue-infinite-loading';
    import {findIndex, replaceOrAppend, replaceInArray, moveToFirstPosition, hasLength} from "./utils";
    import axios from "axios";
    import debounce from "lodash/debounce";
    import queryMixin from "@/queryMixin";

    import {
        GET_USER,
        SET_CHAT_ID,
        SET_CHAT_USERS_COUNT, SET_SEARCH_NAME,
        SET_SHOW_CHAT_EDIT_BUTTON,
        SET_SHOW_SEARCH,
        SET_TITLE
    } from "./store";

    import {mapGetters} from "vuex";


    import ChatListContextMenu from "@/ChatListContextMenu";
    import graphqlSubscriptionMixin from "@/graphqlSubscriptionMixin";

    const pageSize = 40;

    export default {
        mixins: [queryMixin(), graphqlSubscriptionMixin('userOnlineTetATetInChatList')],

        data() {
            return {
                page: 0,
                items: [],
                infiniteId: +new Date(),
                group: -1,
                itemsLoaded: false,
            }
        },
        components:{
            InfiniteLoading,
            ChatListContextMenu,
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}),
            userIsSet() {
                return !!this.currentUser
            }
        },
        methods:{
            // not working until you will change this.items list
            reloadItems() {
                this.infiniteId += 1;
                console.log("Resetting infinite loader", this.infiniteId);
            },
            searchStringChanged(searchString) {
                this.resetVariables();
                this.reloadItems();
            },
            onLoggedIn() {
                if (this.items.length === 0) {
                    this.reloadItems();
                }
            },
            onLoggedOut() {
                this.resetVariables();
            },
            resetVariables() {
                this.items = [];
                this.page = 0;
                this.itemsLoaded = false;
            },
            addItem(dto) {
                console.log("Adding item", dto);
                this.transformItem(dto);
                this.items.unshift(dto);
                this.$forceUpdate();
            },
            changeItem(dto) {
                console.log("Replacing item", dto);
                this.transformItem(dto);
                if (this.hasItem(dto)) {
                    replaceInArray(this.items, dto);
                    moveToFirstPosition(this.items, dto)
                } else {
                    this.items.unshift(dto);
                }
                this.$forceUpdate();
            },
            removeItem(dto) {
                if (this.hasItem(dto)) {
                    console.log("Removing item", dto);
                    const idxToRemove = findIndex(this.items, dto);
                    this.items.splice(idxToRemove, 1);
                    this.$forceUpdate();
                } else {
                    console.log("Item was not be removed", dto);
                }
            },
            // does should change items list (new item added to visible part or not for example)
            hasItem(item) {
                let idxOf = findIndex(this.items, item);
                return idxOf !== -1;
            },

            openChat(item){
                this.$router.push(({ name: chat_name, params: { id: item.id}}));
            },

            infiniteHandler($state) {
                if (!this.userIsSet) {
                    $state.complete();
                    return
                }

                console.debug("infiniteHandler", '"' + this.searchString + '"');
                axios.get('/api/chat', {
                    params: {
                        page: this.page,
                        size: pageSize,
                        searchString: this.searchString,
                    },
                }).then(({ data }) => {
                    const list = data.data;
                    if (list.length) {
                        this.page += 1;
                        list.forEach((item) => {
                            this.transformItem(item);
                        });
                        //this.items = [...this.items, ...list];
                        replaceOrAppend(this.items, list);
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                    this.itemsLoaded = true;
                });
            },
            editChat(chat) {
                const chatId = chat.id;
                console.log("Will add participants to chat", chatId);
                bus.$emit(OPEN_CHAT_EDIT, chatId);
            },
            printParticipants(chat) {
                if (chat.tetATet) {
                    return this.$vuetify.lang.t('$vuetify.tet_a_tet');
                } else {
                    const logins = chat.participants.map(p => p.login);
                    return logins.join(", ")
                }
            },
            deleteChat(chat) {
                bus.$emit(OPEN_SIMPLE_MODAL, {
                    buttonName: this.$vuetify.lang.t('$vuetify.delete_btn'),
                    title: this.$vuetify.lang.t('$vuetify.delete_chat_title', chat.id),
                    text: this.$vuetify.lang.t('$vuetify.delete_chat_text', chat.name),
                    actionFunction: ()=> {
                        axios.delete(`/api/chat/${chat.id}`)
                            .then(() => {
                                bus.$emit(CLOSE_SIMPLE_MODAL);
                            })
                    }
                });
            },
            leaveChat(chat) {
                bus.$emit(OPEN_SIMPLE_MODAL, {
                    buttonName: this.$vuetify.lang.t('$vuetify.leave_btn'),
                    title: this.$vuetify.lang.t('$vuetify.leave_chat_title', chat.id),
                    text: this.$vuetify.lang.t('$vuetify.leave_chat_text', chat.name),
                    actionFunction: ()=> {
                        axios.put(`/api/chat/${chat.id}/leave`)
                            .then(() => {
                                bus.$emit(CLOSE_SIMPLE_MODAL);
                            })
                    }
                });
            },
            onChangeUnreadMessages(dto) {
                const chatId = dto.chatId;
                let idxOf = findIndex(this.items, {id: chatId});
                if (idxOf != -1) {
                    this.items[idxOf].unreadMessages = dto.unreadMessages;
                    this.$forceUpdate();
                } else {
                    console.log("Not found to update unread messages", dto)
                }
            },
            onUserProfileChanged(user) {
                this.items.forEach(item => {
                    replaceInArray(item.participants, user);
                });
                this.$forceUpdate();
            },
            onWsRestoredRefresh() {
                this.searchStringChanged(null);
            },
            onVideoCallChanged(dto) {
                let matched = false;
                this.items.forEach(item => {
                    if (item.id == dto.chatId) {
                        item.videoChatUsersCount = dto.usersCount;
                        matched = true;
                    }
                });
                if (matched) {
                    this.$forceUpdate();
                }
            },
            onShowContextMenu(e, menuableItem){
                this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
            },
            onCloseContextMenu(){
                this.$refs.contextMenuRef.onCloseContextMenu()
            },
            shouldShowWelcome(){
                return this.userIsSet && !this.items.length && this.itemsLoaded && !hasLength(this.searchString)
            },
            createChat() {
                bus.$emit(OPEN_CHAT_EDIT, null);
            },
            findUser() {
                bus.$emit(OPEN_FIND_USER)
            },

            transformItem(item) {
                item.online = false;
            },
            getTetATetParticipantIds(items) {
                return items.filter((item) => item.tetATet).map((item) => item.participants.filter((p) => p.id != this.currentUser?.id)[0].id);
            },
            onUserOnlineChanged(rawData) {
                const dtos = rawData?.data?.userOnlineEvents;
                if (dtos) {
                    this.items.forEach(item => {
                        dtos.forEach(dtoItem => {
                            if (item.tetATet && item.participants.filter((p)=> p.id == dtoItem.id).length) {
                                item.online = dtoItem.online;
                            }
                        })
                    })
                    this.$forceUpdate();
                }
            },

            getGraphQlSubscriptionQuery() {
                return `
                subscription {
                    userOnlineEvents(userIds:[${this.getTetATetParticipantIds(this.items)}]) {
                        id
                        online
                    }
                }`
            },
            onNextSubscriptionElement(items) {
                this.onUserOnlineChanged(items);
            },

            getLink(item) {
                return chat + "/" + item.id
            },
            getChatName(item) {
                let bldr = item.name;
                if (!item.avatar && item.online) {
                    bldr += (" (" + this.$vuetify.lang.t('$vuetify.user_online') + ")");
                }
                return bldr;
            },
        },
        created() {
            this.searchStringChanged = debounce(this.searchStringChanged, 700, {leading:false, trailing:true});

            bus.$on(PROFILE_SET, this.onLoggedIn);
            bus.$on(LOGGED_OUT, this.onLoggedOut);
            bus.$on(CHAT_ADD, this.addItem);
            bus.$on(CHAT_EDITED, this.changeItem);
            bus.$on(CHAT_DELETED, this.removeItem);
            bus.$on(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
            bus.$on(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
            bus.$on(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);

            this.initQueryAndWatcher();
        },
        beforeDestroy() {
            this.graphQlUnsubscribe();
            this.closeQueryWatcher();
        },
        destroyed() {
            bus.$off(PROFILE_SET, this.onLoggedIn);
            bus.$off(LOGGED_OUT, this.onLoggedOut);
            bus.$off(CHAT_ADD, this.addItem);
            bus.$off(CHAT_EDITED, this.changeItem);
            bus.$off(CHAT_DELETED, this.removeItem);
            bus.$off(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
            bus.$off(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
            bus.$off(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
        },
        mounted() {
            this.$store.commit(SET_TITLE, this.$vuetify.lang.t('$vuetify.chats'));
            this.$store.commit(SET_CHAT_USERS_COUNT, 0);
            this.$store.commit(SET_SHOW_SEARCH, true);
            this.$store.commit(SET_SEARCH_NAME, this.$vuetify.lang.t('$vuetify.search_in_chats'));
            this.$store.commit(SET_CHAT_ID, null);
            this.$store.commit(SET_SHOW_CHAT_EDIT_BUTTON, false);
        },
        watch: {
          '$vuetify.lang.current': {
            handler: function (newValue, oldValue) {
              this.$store.commit(SET_TITLE, this.$vuetify.lang.t('$vuetify.chats'));
                this.$store.commit(SET_SEARCH_NAME, this.$vuetify.lang.t('$vuetify.search_in_chats'));
            },
          },
          items(newValue, oldValue) {
              const newParticipants = this.getTetATetParticipantIds(newValue);
              if (newParticipants.length == 0) {
                  this.graphQlUnsubscribe();
              } else {
                  this.graphQlSubscribe();
              }
          },
        },
    }
</script>

<style lang="stylus" scoped>
    .min-height {
        display inline-block
        min-height 22px
    }
</style>
