<template>
    <v-card>
        <v-list>
            <v-list-item-group v-model="group" color="primary">
            <v-list-item
                    v-for="(item, index) in items"
                    :key="item.id"
            >
                <v-list-item-content @click="openChat(item)">
                    <v-list-item-title>
                        <span class="min-height">
                            {{item.name}}
                        </span>
                        <v-badge v-if="item.unreadMessages" inline :content="item.unreadMessages" class="mt-0"></v-badge>
                        <v-badge v-if="item.videoChatUsersCount" color="success" icon="mdi-phone" inline  class="mt-0"/>
                    </v-list-item-title>
                    <v-list-item-subtitle v-html="printParticipants(item)"></v-list-item-subtitle>
                </v-list-item-content>
                <v-list-item-action>
                    <v-container class="mb-0 mt-0 pl-0 pr-0 pb-0 pt-0">
                        <v-btn v-if="item.canEdit" icon color="primary" @click="editChat(item)"><v-icon dark>mdi-lead-pencil</v-icon></v-btn>
                        <v-btn v-if="item.canDelete" icon @click="deleteChat(item)" color="error"><v-icon dark>mdi-delete</v-icon></v-btn>
                        <v-btn v-if="item.canLeave" icon @click="leaveChat(item)"><v-icon dark>mdi-exit-run</v-icon></v-btn>
                    </v-container>
                </v-list-item-action>
            </v-list-item>
            </v-list-item-group>
        </v-list>
        <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId">
            <template slot="no-more"><span/></template>
            <template slot="no-results"><span/></template>
        </infinite-loading>

    </v-card>

</template>

<script>
    import bus, {
        CHAT_ADD,
        CHAT_EDITED,
        CHAT_DELETED,
        LOGGED_IN,
        OPEN_CHAT_EDIT,
        OPEN_SIMPLE_MODAL,
        UNREAD_MESSAGES_CHANGED,
        USER_PROFILE_CHANGED,
        CLOSE_SIMPLE_MODAL,
        REFRESH_ON_WEBSOCKET_RESTORED,
        VIDEO_CALL_CHANGED
    } from "./bus";
    import {chat_name} from "./routes";
    import infinityListMixin, { pageSize } from "./InfinityListMixin";
    import { findIndex, replaceOrAppend, replaceInArray, moveToFirstPosition } from "./utils";
    import axios from "axios";
    import {
        SET_CHAT_ID,
        SET_CHAT_USERS_COUNT,
        SET_SHOW_CHAT_EDIT_BUTTON,
        SET_SHOW_SEARCH,
        SET_TITLE
    } from "./store";

    export default {
        mixins: [infinityListMixin()],
        computed: {
            chatRoute() {
                return chat_name;
            },
        },
        data() {
            return {
                group: -1
            }
        },
        methods:{
            addItem(dto) {
                console.log("Adding item", dto);
                this.items.unshift(dto);
                this.$forceUpdate();
            },
            changeItem(dto) {
                console.log("Replacing item", dto);
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
                axios.get('/api/chat', {
                    params: {
                        page: this.page,
                        size: pageSize,
                    },
                }).then(({ data }) => {
                    const list = data.data;
                    this.itemsTotal = data.totalCount;
                    if (list.length) {
                        this.page += 1;
                        //this.items = [...this.items, ...list];
                        replaceOrAppend(this.items, list);
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                });
            },
            editChat(chat) {
                const chatId = chat.id;
                console.log("Will add participants to chat", chatId);
                bus.$emit(OPEN_CHAT_EDIT, chatId);
            },
            printParticipants(chat) {
                const logins = chat.participants.map(p => p.login);
                return logins.join(", ")
            },
            deleteChat(chat) {
                bus.$emit(OPEN_SIMPLE_MODAL, {
                    buttonName: 'Delete',
                    title: `Delete chat #${chat.id}`,
                    text: `Are you sure to delete chat '${chat.name}' ?`,
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
                    buttonName: 'Leave',
                    title: `Leave chat #${chat.id}`,
                    text: `Are you sure to leave from chat '${chat.name}' ?`,
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
                this.searchStringChanged();
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
        },
        created() {
            bus.$on(LOGGED_IN, this.reloadItems);
            bus.$on(CHAT_ADD, this.addItem);
            bus.$on(CHAT_EDITED, this.changeItem);
            bus.$on(CHAT_DELETED, this.removeItem);
            bus.$on(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
            bus.$on(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
            bus.$on(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
        },
        destroyed() {
            bus.$off(LOGGED_IN, this.reloadItems);
            bus.$off(CHAT_ADD, this.addItem);
            bus.$off(CHAT_EDITED, this.changeItem);
            bus.$off(CHAT_DELETED, this.removeItem);
            bus.$off(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
            bus.$off(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
            bus.$off(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
        },
        mounted() {
            this.$store.commit(SET_TITLE, "Chats");
            this.$store.commit(SET_CHAT_USERS_COUNT, 0);
            this.$store.commit(SET_SHOW_SEARCH, true);
            this.$store.commit(SET_CHAT_ID, null);
            this.$store.commit(SET_SHOW_CHAT_EDIT_BUTTON, false);
        }
    }
</script>

<style lang="stylus" scoped>
    .min-height {
        display inline-block
        min-height 22px
    }
</style>