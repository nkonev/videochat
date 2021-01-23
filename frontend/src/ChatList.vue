<template>
    <v-card>
        <v-list>
            <v-list-item-group v-model="group" color="primary">
            <v-list-item
                    v-for="(item, index) in items"
                    :key="item.id"
            >
                <v-list-item-content @click="openChat(item)">
                    <v-list-item-title>{{item.name}} <v-badge v-if="item.unreadMessages" :content="item.unreadMessages" offset-x="-4" /></v-list-item-title>
                    <v-list-item-subtitle v-html="printParticipants(item)"></v-list-item-subtitle>
                </v-list-item-content>
                <v-list-item-action>
                    <v-container class="mb-0 mt-0 pb-0 pt-0">
                        <v-btn v-if="item.canEdit" icon color="primary" @click="editChat(item)"><v-icon dark>mdi-lead-pencil</v-icon></v-btn>
                        <v-btn v-if="item.canEdit" icon @click="deleteChat(item)" color="error"><v-icon dark>mdi-delete</v-icon></v-btn>
                        <v-btn v-if="item.canLeave" icon @click="leaveChat(item)"><v-icon dark>mdi-exit-run</v-icon></v-btn>
                    </v-container>
                </v-list-item-action>
            </v-list-item>
            </v-list-item-group>
        </v-list>
        <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId"></infinite-loading>

    </v-card>

</template>

<script>
import bus, {
    CHAT_ADD,
    CHAT_EDITED,
    CHAT_DELETED,
    CHAT_SEARCH_CHANGED,
    LOGGED_IN,
    OPEN_CHAT_EDIT,
    CHANGE_TITLE, OPEN_SIMPLE_MODAL, UNREAD_MESSAGES_CHANGED, USER_PROFILE_CHANGED, CLOSE_SIMPLE_MODAL
} from "./bus";
    import {chat_name} from "./routes";
    import infinityListMixin, {
        findIndex,
        pageSize,
        replaceOrAppend,
        replaceInArray,
        moveToFirstPosition
    } from "./InfinityListMixin";
    import axios from "axios";
    import {mapGetters} from 'vuex'
    import {GET_SEARCH_STRING} from "./store";
    import {titleFactory} from "./changeTitle";

    export default {
        mixins: [infinityListMixin()],
        computed: {
            chatRoute() {
                return chat_name;
            },
            ...mapGetters({storedSearchString: GET_SEARCH_STRING})
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
                        searchString: this.storedSearchString
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
                axios.put(`/api/chat/${chat.id}/leave`)
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
        },
        created() {
            bus.$on(LOGGED_IN, this.reloadItems);
            bus.$on(CHAT_ADD, this.addItem);
            bus.$on(CHAT_EDITED, this.changeItem);
            bus.$on(CHAT_DELETED, this.removeItem);
            bus.$on(CHAT_SEARCH_CHANGED, this.searchStringChanged);
            bus.$on(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
            bus.$on(USER_PROFILE_CHANGED, this.onUserProfileChanged);
        },
        destroyed() {
            bus.$off(LOGGED_IN, this.reloadItems);
            bus.$off(CHAT_ADD, this.addItem);
            bus.$off(CHAT_EDITED, this.changeItem);
            bus.$off(CHAT_DELETED, this.removeItem);
            bus.$off(CHAT_SEARCH_CHANGED, this.searchStringChanged);
            bus.$off(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
            bus.$off(USER_PROFILE_CHANGED, this.onUserProfileChanged);
        },
        mounted() {
            bus.$emit(CHANGE_TITLE, titleFactory("Chats", true, false, null, null, null));
        }
    }
</script>
