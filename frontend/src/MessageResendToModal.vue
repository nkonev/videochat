<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="700" scrollable :persistent="hasSearchString()">
            <v-card>
                <v-card-title class="d-flex align-center ml-2">
                    <template v-if="showSearchButton">
                        {{ $vuetify.locale.t('$vuetify.share_to') }}
                    </template>

                    <CollapsedSearch :provider="{
                      getModelValue: this.getModelValue,
                      setModelValue: this.setModelValue,
                      getShowSearchButton: this.getShowSearchButton,
                      setShowSearchButton: this.setShowSearchButton,
                      searchName: this.searchName,
                      textFieldVariant: 'outlined',
                    }"/>

                </v-card-title>

                <v-card-text class="ma-0 pa-0 resend-to-wrapper">
                    <v-list class="pb-0" v-if="!loading">
                        <template v-if="chats.length > 0">
                            <template v-for="(item, index) in chats">
                                <v-hover v-slot="{ isHovering, props }">
                                    <v-list-item @click="resendMessageTo(item.id)" v-bind="props" class="list-item-prepend-spacer-16">
                                        <template v-slot:prepend v-if="hasLength(item.avatar)">
                                            <v-avatar :image="item.avatar"></v-avatar>
                                        </template>
                                        <v-list-item-title class="chat-name">{{ getChatName(item)}}</v-list-item-title>
                                        <v-list-item-subtitle :class="!isHovering ? 'white-colored' : ''">{{ isHovering ? $vuetify.locale.t('$vuetify.resend_to_here') : '-' }}</v-list-item-subtitle>
                                    </v-list-item>

                                </v-hover>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.locale.t('$vuetify.no_chats') }}</v-card-text>
                        </template>
                    </v-list>
                    <v-progress-circular
                        class="ma-4"
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                </v-card-text>
                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-spacer></v-spacer>
                    <v-btn
                        variant="elevated"
                        color="red"
                        @click="closeModal()"
                    >
                        {{ $vuetify.locale.t('$vuetify.close') }}
                    </v-btn>
                </v-card-actions>

            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    OPEN_RESEND_TO_MODAL,
} from "./bus/bus";
import {hasLength} from "./utils";
import axios from "axios";
import debounce from "lodash/debounce";
import CollapsedSearch from "@/CollapsedSearch.vue";
import Mark from "mark.js";

export default {
    data () {
        return {
            show: false,
            searchString: null,
            chats: [ ], // max 20 items and search
            loading: false,
            messageDto: null,
            showSearchButton: true,
            markInstance: null,
        }
    },

    methods: {
        hasLength,
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
            this.showSearchButton = true;
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
                this.performMarking();
            })
        },
        getChatName(item) {
            return item.name
        },
        hasSearchString() {
            return hasLength(this.searchString)
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
        getModelValue() {
            return this.searchString
        },
        setModelValue(v) {
            this.searchString = v
        },
        getShowSearchButton() {
            return this.showSearchButton
        },
        setShowSearchButton(v) {
            this.showSearchButton = v
        },
        searchName() {
            return this.$vuetify.locale.t('$vuetify.search_by_chats')
        },
        performMarking() {
            this.$nextTick(()=>{
                if (hasLength(this.searchString)) {
                    this.markInstance.unmark();
                    this.markInstance.mark(this.searchString);
                }
            })
        },
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
    },
    components: {
        CollapsedSearch
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
    },
    mounted() {
        this.markInstance = new Mark(".resend-to-wrapper .chat-name");
        bus.on(OPEN_RESEND_TO_MODAL, this.showModal);
    },
    beforeUnmount() {
        this.markInstance.unmark();
        this.markInstance = null;
        bus.off(OPEN_RESEND_TO_MODAL, this.showModal);
    },
}
</script>

<style lang="stylus">
.white-colored {
    color white !important
}
</style>
