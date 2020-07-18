<template>
    <v-card
            max-width="1000"
            class="mx-auto"
    >
        <v-list>
            <v-list-item
                    v-for="(item, index) in items"
                    :key="item.id"
            >
                <v-list-item-content>
                    <router-link :to="{name: chatRoute, params: { id: item.id} }">
                        <v-list-item-title v-html="item.name"></v-list-item-title>
                    </router-link>
                    <v-list-item-subtitle v-html="printParticipants(item)"></v-list-item-subtitle>
                </v-list-item-content>
                <v-list-item-action v-if="item.canEdit">
                    <v-btn color="primary" fab dark small @click="editChat(item)"><v-icon dark>mdi-lead-pencil</v-icon></v-btn>
                </v-list-item-action>
            </v-list-item>
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
        CHANGE_TITLE
    } from "./bus";
    import {chat_name} from "./routes";
    import infinityListMixin, {
        findIndex,
        ACTION_CREATE,
        pageSize,
        replaceOrAppend,
        ACTION_EDIT, replaceInArray, ACTION_DELETE
    } from "./InfinityListMixin";
    import axios from "axios";

    export default {
        mixins: [infinityListMixin()],
        computed: {
            chatRoute() {
                return chat_name;
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
                    // TODO move to first position
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
                let idxOf = findIndex(data.items, item);
                return idxOf !== -1;
            },

            infiniteHandler($state) {
                axios.get('/api/chat', {
                    params: {
                        page: this.page,
                        size: pageSize,
                        searchString: this.searchString
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
        },
        created() {
            bus.$on(LOGGED_IN, this.reloadItems);
            bus.$on(CHAT_ADD, this.addItem);
            bus.$on(CHAT_EDITED, this.changeItem);
            bus.$on(CHAT_DELETED, this.removeItem);
            bus.$on(CHAT_SEARCH_CHANGED, this.setSearchString);
        },
        destroyed() {
            bus.$off(LOGGED_IN, this.reloadItems);
            bus.$off(CHAT_ADD, this.addItem);
            bus.$off(CHAT_EDITED, this.changeItem);
            bus.$off(CHAT_DELETED, this.removeItem);
            bus.$off(CHAT_SEARCH_CHANGED, this.setSearchString);
        },
        mounted() {
            bus.$emit(CHANGE_TITLE, "Chats");
        }
    }
</script>
