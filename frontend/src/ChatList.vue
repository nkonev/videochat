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
    import bus, {CHAT_ADD, CHAT_EDITED, CHAT_DELETED, CHAT_SEARCH_CHANGED, LOGGED_IN, OPEN_CHAT_EDIT} from "./bus";
    import {chat_name} from "./routes";
    import infinityListMixin, {findIndex, ACTION_CREATE} from "./InfinityListMixin";

    export default {
        mixins: [infinityListMixin(()=>'/api/chat', (data, item, action, isLastPage)=>{
            console.log("isLastPage", isLastPage, "action", action);
            if (isLastPage && action === ACTION_CREATE) {
                return true;
            }
            let idxOf = findIndex(data.items, item);
            return idxOf !== -1;
        }, (infinityThis, data, $state)=>{
            const list = data.data;
            infinityThis.itemsTotal = data.totalCount;
            if (list.length) {
                infinityThis.page += 1;
                infinityThis.items = [...infinityThis.items, ...list];
                //replaceOrAppend(this.items, list);
                $state.loaded();
            } else {
                $state.complete();
            }

        })],
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
    }
</script>
