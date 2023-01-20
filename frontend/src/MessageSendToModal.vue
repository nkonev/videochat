<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480"  scrollable :persistent="hasSearchString()">
            <v-card>
                <v-card-title>
                    {{ $vuetify.lang.t('$vuetify.share_to') }}
                </v-card-title>
                <v-container class="ma-0 pa-0">
                    <v-text-field class="ml-4 mr-4 pt-0 mt-0" prepend-icon="mdi-magnify" hide-details single-line v-model="searchString" :label="$vuetify.lang.t('$vuetify.search_by_chats')" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput" autofocus></v-text-field>
                </v-container>
                <v-card-text class="ma-0 pa-0">
                    <v-list class="pb-0">
                        <template v-if="chats.length > 0">
                            <template v-for="(item, index) in chats">
                                <v-list-item link>
                                    <v-list-item-avatar v-if="item.avatar">
                                        <img :src="item.avatar"/>
                                    </v-list-item-avatar>
                                    <v-list-item-content class="py-2">
                                        <v-list-item-title>{{ getNotificationTitle(item)}}</v-list-item-title>
                                    </v-list-item-content>
                                </v-list-item>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.lang.t('$vuetify.no_chats') }}</v-card-text>
                        </template>

                    </v-list>
                </v-card-text>
                <v-divider/>
                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-spacer></v-spacer>
                    <v-btn
                        color="error"
                        class="my-1"
                        @click="closeModal()"
                    >
                        {{ $vuetify.lang.t('$vuetify.close') }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    OPEN_NOTIFICATIONS_DIALOG, OPEN_SEND_TO_MODAL,
} from "./bus";
import {mapGetters} from 'vuex'
import {GET_NOTIFICATIONS, GET_NOTIFICATIONS_SETTINGS, SET_NOTIFICATIONS_SETTINGS} from "@/store";
import {getHumanReadableDate, hasLength} from "./utils";
import axios from "axios";
import {chat, chat_name} from "@/routes";

export default {
    data () {
        return {
            show: false,
            searchString: null,
            chats: [ // max 20 items and search
                {
                    name: "Chat 1",
                    id: 1,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 2",
                    id: 2,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 3",
                    id: 3,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 4",
                    id: 4,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 5",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 6",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 7",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 8",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 9",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 10",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 11",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 12",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 13",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 14",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 15",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 16",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 17",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 18",
                    id: 5,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 19",
                    id: 19,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },
                {
                    name: "Chat 20",
                    id: 20,
                    avatar:"/api/storage/public/user/avatar/1078_AVATAR_200x200.jpg?time=1674160525"
                },

            ]
        }
    },

    methods: {
        showModal() {
            this.show = true;
        },
        closeModal() {
            this.show = false;
        },
        getNotificationTitle(item) {
            return item.name
        },
        hasSearchString() {
            return hasLength(this.searchString)
        },
        resetInput() {
            this.searchString = null;
        },
    },

    watch: {
        show(newValue) {
            if (!newValue) {
                this.closeModal();
            }
        }
    },
    created() {
        bus.$on(OPEN_SEND_TO_MODAL, this.showModal);
    },
    destroyed() {
        bus.$off(OPEN_SEND_TO_MODAL, this.showModal);
    },
}
</script>
