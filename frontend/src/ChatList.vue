<template>
    <v-card
            max-width="1000"
            class="mx-auto"
    >
        <v-list>
            <v-list-item
                    v-for="(item, index) in chats"
                    :key="item.id"
            >
                <v-list-item-content>
                    <router-link :to="{name: chatRoute, params: { id: item.id} }">
                        <v-list-item-title v-html="item.name"></v-list-item-title>
                    </router-link>
                    <v-list-item-subtitle v-html="printParticipants(item)"></v-list-item-subtitle>
                </v-list-item-content>
                <v-list-item-action>
                    <v-btn color="primary" fab dark small @click="editChat(item)"><v-icon dark>mdi-plus</v-icon></v-btn>
                </v-list-item-action>
            </v-list-item>
        </v-list>
        <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId"></infinite-loading>

    </v-card>

</template>

<script>
    import axios from 'axios';
    import InfiniteLoading from 'vue-infinite-loading';
    import bus, {CHAT_SAVED, CHAT_SEARCH_CHANGED, LOGGED_IN, OPEN_CHAT_EDIT} from "./bus";
    import {chat_name} from "./routes";

    const replaceInArray = (array, element) => {
        const foundIndex = array.findIndex(value => value.id === element.id);
        if (foundIndex === -1) {
            return false;
        } else {
            array[foundIndex] = element;
            return true;
        }
    };

    const replaceOrAppend = (array, newArray) => {
        newArray.forEach((element, index) => {
            const replaced = replaceInArray(array, element);
            if (!replaced) {
                array.push(element);
            }
        });
    };

    const pageSize = 20;

    export default {
        data () {
            return {
                page: 0,
                lastPageActualSize: 0,
                chats: [],
                openEditModal: false,
                editChatId: null,
                infiniteId: new Date(),
                searchString: ""
            }
        },
        components:{
            InfiniteLoading,
        },
        computed: {
            chatRoute() {
                return chat_name;
            }
        },
        methods:{
            editChat(chat) {
                const chatId = chat.id;
                console.log("Will add participants to chat", chatId);
                bus.$emit(OPEN_CHAT_EDIT, chatId);
            },
            infiniteHandler($state) {
                axios.get(`/api/chat`, {
                    params: {
                        page: this.page,
                        size: pageSize,
                        searchString: this.searchString
                    },
                }).then(({ data }) => {
                    if (data.length) {
                        this.page += 1;
                        //this.chats.push(...data);
                        replaceOrAppend(this.chats, data);
                        this.lastPageActualSize = data.length;
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                });
            },
            reloadChats() {
                this.infiniteId += 1;
                console.log("Resetting infinite loader", this.infiniteId);
            },
            rerenderChat(dto) {
                console.log("Rerendering chat", dto);
                const replaced = replaceInArray(this.chats, dto);
                console.debug("Replaced:", replaced);
                if (!replaced) {
                    this.reloadLastPage();
                }
                this.$forceUpdate();
            },
            reloadLastPage() {
                console.log("this.lastPageActualSize", this.lastPageActualSize);
                if (this.lastPageActualSize > 0) {
                    this.page--;
                    // remove lastPageActualSize
                    this.chats.splice(-1, this.lastPageActualSize);
                    console.log("removing last", this.lastPageActualSize);
                } else {
                    this.page--;
                    // remove 20
                    this.chats.splice(-1, pageSize);
                    console.log("removing last", pageSize);
                }
                this.reloadChats();
            },
            printParticipants(chat) {
                const logins = chat.participants.map(p => p.login);
                return logins.join(", ")
            },
            setSearchString(searchString) {
                this.searchString = searchString;
                this.chats = [];
                this.page = 0;
                this.reloadChats();
            },
        },
        created() {
            bus.$on(LOGGED_IN, this.reloadChats);
            bus.$on(CHAT_SAVED, this.rerenderChat);
            bus.$on(CHAT_SEARCH_CHANGED, this.setSearchString);
        },
        destroyed() {
            bus.$off(LOGGED_IN, this.reloadChats);
            bus.$off(CHAT_SAVED, this.rerenderChat);
            bus.$off(CHAT_SEARCH_CHANGED, this.setSearchString);
        },
    }
</script>
