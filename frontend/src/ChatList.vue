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
                <v-list-item-action>
                    <v-btn color="primary" fab dark small @click="editChat(item)"><v-icon dark>mdi-lead-pencil</v-icon></v-btn>
                </v-list-item-action>
            </v-list-item>
        </v-list>
        <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId"></infinite-loading>

    </v-card>

</template>

<script>
    import bus, {CHAT_SAVED, CHAT_SEARCH_CHANGED, LOGGED_IN, OPEN_CHAT_EDIT} from "./bus";
    import {chat_name} from "./routes";
    import infinityListMixin from "./InfinityListMixin";

    export default {
        mixins: [infinityListMixin(()=>'/api/chat')],
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
            printParticipants(chat) {
                const logins = chat.participants.map(p => p.login);
                return logins.join(", ")
            },
        },
        created() {
            bus.$on(LOGGED_IN, this.reloadItems);
            bus.$on(CHAT_SAVED, this.rerenderItem);
            bus.$on(CHAT_SEARCH_CHANGED, this.setSearchString);
        },
        destroyed() {
            bus.$off(LOGGED_IN, this.reloadItems);
            bus.$off(CHAT_SAVED, this.rerenderItem);
            bus.$off(CHAT_SEARCH_CHANGED, this.setSearchString);
        },
    }
</script>
