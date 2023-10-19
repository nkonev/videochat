<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480"  scrollable :persistent="hasSearchString()">
            <v-card>
                <v-card-title>
                    {{ $vuetify.lang.t('$vuetify.share_to') }}
                </v-card-title>
                <v-container class="ma-0 pa-0">
                    <v-text-field class="ml-4 mr-4 pt-0 mt-0" prepend-icon="mdi-magnify" hide-details single-line v-model="searchString" :label="$vuetify.lang.t('$vuetify.search_by_chats')" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput"></v-text-field>
                </v-container>
                <v-card-text class="ma-0 pa-0">
                    <v-list class="pb-0" v-if="!loading">
                        <template v-if="chats.length > 0">
                            <template v-for="(item, index) in chats">
                                <v-hover v-slot:default="{ hover }">
                                    <v-list-item link @click="resendMessageTo(item.id)">
                                        <v-list-item-avatar v-if="item.avatar">
                                            <img :src="item.avatar"/>
                                        </v-list-item-avatar>
                                        <v-list-item-content class="py-2">
                                            <v-list-item-title>{{ getNotificationTitle(item)}}</v-list-item-title>
                                            <v-list-item-subtitle :class="!hover ? 'white-colored' : ''">{{ hover ? $vuetify.lang.t('$vuetify.resend_to_here') : '-' }}</v-list-item-subtitle>
                                        </v-list-item-content>
                                    </v-list-item>
                                </v-hover>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.lang.t('$vuetify.no_chats') }}</v-card-text>
                        </template>
                    </v-list>
                    <v-progress-circular
                        class="ma-4 pa-4"
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                </v-card-text>
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
    OPEN_RESEND_TO_MODAL,
} from "./bus";
import {hasLength} from "./utils";
import axios from "axios";
import debounce from "lodash/debounce";

export default {
    data () {
        return {
            show: false,
            searchString: null,
            chats: [ ], // max 20 items and search
            loading: false,
            messageDto: null,
        }
    },

    methods: {
        showModal(messageDto) {
            this.show = true;
            this.messageDto = messageDto;
            this.loadData();
        },
        closeModal() {
            this.show = false;
            this.chats = [];
            this.loading = false;
            this.searchString = null;
            this.messageDto = null;
        },
        loadData() {
            if (!this.show) {
                return
            }
            this.loading = true;
            axios.get('/api/chat', {
                params: {
                    searchString: this.searchString,
                },
            }).then(({data}) => {
                this.chats = data.data;
                this.loading = false;
            })
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
        doSearch(){
            this.loadData();
        },
        resendMessageTo(chatId) {
            const messageDto = {
                text: this.messageDto.text, // yes, we copy text as is, along with embedded images and video
                embedMessage: {
                    id: this.messageDto.id,
                    chatId: parseInt(this.chatId),
                    embedType: "resend"
                }
            };
            axios.post(`/api/chat/`+chatId+'/message', messageDto).then(()=> {
                this.closeModal()
            })
        },
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
    },

    watch: {
        show(newValue) {
            if (!newValue) {
                this.closeModal();
            }
        },
        searchString (searchString) {
            this.doSearch();
        },
    },
    created() {
        this.doSearch = debounce(this.doSearch, 700);
        bus.$on(OPEN_RESEND_TO_MODAL, this.showModal);
    },
    destroyed() {
        bus.$off(OPEN_RESEND_TO_MODAL, this.showModal);
    },
}
</script>

<style lang="stylus">
.white-colored {
    color white !important
}
</style>
