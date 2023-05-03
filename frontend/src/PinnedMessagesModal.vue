<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="800" scrollable>
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.pinned_messages') }}</v-card-title>

                <v-card-text class="ma-0 pa-0">
                    <v-list v-if="!loading">
                        <template v-if="dto.totalCount > 0">
                            <template v-for="(item, index) in dto.data">
                                <v-list-item>
                                    <v-list-item-avatar>
                                        <img :src="item.owner.avatar"/>
                                    </v-list-item-avatar>
                                    <v-list-item-content class="py-2">
                                        <v-list-item-title>
                                            <router-link :to="{ name: 'profileUser', params: { id: item.owner.id }}">{{getOwner(item.owner)}}</router-link><span class="with-space"> {{$vuetify.lang.t('$vuetify.time_at')}} </span>{{getDate(item)}}
                                        </v-list-item-title>
                                        <v-list-item-subtitle>
                                            <router-link :to="getPinnedRouteObject(item)" style="text-decoration: none; cursor: pointer" class="text--primary">
                                                {{ item.text }}
                                            </router-link>
                                        </v-list-item-subtitle>
                                    </v-list-item-content>

                                    <v-btn icon @click="promotePinMessage(item)">
                                        <v-icon color="primary" dark :title="$vuetify.lang.t('$vuetify.pin_message')">mdi-pin</v-icon>
                                    </v-btn>
                                    <v-btn icon @click="unpinMessage(item)">
                                        <v-icon color="error" dark :title="$vuetify.lang.t('$vuetify.remove_from_pinned')">mdi-delete</v-icon>
                                    </v-btn>
                                </v-list-item>
                                <v-divider></v-divider>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.lang.t('$vuetify.no_pin_messages') }}</v-card-text>
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
                    <v-pagination
                        v-if="shouldShowPagination"
                        v-model="page"
                        :length="pagesCount"
                    ></v-pagination>
                    <v-spacer></v-spacer>
                    <v-btn color="error" class="my-1" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    OPEN_PINNED_MESSAGES_MODAL, PINNED_MESSAGE_PROMOTED, PINNED_MESSAGE_UNPROMOTED,
} from "./bus";
import {mapGetters} from "vuex";
import {GET_USER} from "./store";
import axios from "axios";
import {getHumanReadableDate, formatSize, findIndex, replaceOrAppend} from "./utils";
import {chat_name, messageIdHashPrefix, videochat_name} from "@/routes";

const firstPage = 1;
const pageSize = 20;

const dtoFactory = () => {return {data: []} };

export default {
    data () {
        return {
            show: false,
            dto: dtoFactory(),
            loading: false,
            page: firstPage,
        }
    },
    computed: {
        ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        pagesCount() {
            const count = Math.ceil(this.dto.totalCount / pageSize);
            console.debug("Calc pages count", count);
            return count;
        },
        shouldShowPagination() {
            return this.dto != null && this.dto.data && this.dto.totalCount > pageSize
        },
        chatId() {
            return this.$route.params.id
        },
    },

    methods: {
        showModal() {
            this.show = true;
            this.getPinnedMessages();
        },
        translatePage() {
            return this.page - 1;
        },
        getPinnedMessages() {
            if (!this.show) {
                return
            }
            this.loading = true;
            axios.get(`/api/chat/${this.chatId}/message/pin`, {
                params: {
                    page: this.translatePage(),
                    size: pageSize,
                },
            })
                .then(({data}) => {
                    this.dto = data;
                })
                .finally(() => {
                    this.loading = false;
                })
        },
        closeModal() {
            this.show = false;
            this.page = firstPage;
            this.dto = dtoFactory();
        },
        unpinMessage(dto) {
            axios.put(`/api/chat/${this.chatId}/message/${dto.id}/pin`, null, {
                params: {
                    pin: false
                },
            });

        },
        promotePinMessage(dto) {
            axios.put(`/api/chat/${this.chatId}/message/${dto.id}/pin`, null, {
                params: {
                    pin: true
                },
            });
        },
        getDate(item) {
            return getHumanReadableDate(item.createDateTime)
        },
        getOwner(owner) {
            return owner.login
        },
        removeItem(dto) {
            console.log("Removing item", dto);
            const idxToRemove = findIndex(this.dto.data, dto);
            this.dto.data.splice(idxToRemove, 1);
            this.$forceUpdate();
        },
        replaceItem(dto) {
            console.log("Replacing item", dto);

            replaceOrAppend(this.dto.data, [dto]);

            this.$forceUpdate();
        },
        onPinnedMessageUnpromoted(dto) {
            if (this.show) {
                if (dto.message.chatId == this.chatId) {
                    this.removeItem(dto.message);
                    this.dto.totalCount = dto.totalCount;
                } else {
                    console.log("Skipping", dto)
                }
            }
        },
        onPinnedMessagePromoted(dto) {
            if (this.show) {
                if (dto.message.chatId == this.chatId) {
                    this.replaceItem(dto.message);
                    this.dto.totalCount = dto.totalCount;
                } else {
                    console.log("Skipping", dto)
                }
            }
        },
        isVideoRoute() {
            return this.$route.name == videochat_name
        },
        getPinnedRouteObject(item) {
            const routeName = this.isVideoRoute() ? videochat_name : chat_name;
            return {name: routeName, params: {id: item.chatId}, hash: messageIdHashPrefix + item.id};
        },
    },
    filters: {
        formatSizeFilter(size) {
            return formatSize((size))
        },
    },
    watch: {
        page(newValue) {
            if (this.show) {
                console.debug("SettingNewPage", newValue);
                this.dto = dtoFactory();
                this.getPinnedMessages();
            }
        },
        show(newValue) {
            if (!newValue) {
                this.closeModal();
            }
        }
    },
    created() {
        bus.$on(OPEN_PINNED_MESSAGES_MODAL, this.showModal);
        bus.$on(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
        bus.$on(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);
    },
    destroyed() {
        bus.$off(OPEN_PINNED_MESSAGES_MODAL, this.showModal);
        bus.$off(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
        bus.$off(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);
    },
}
</script>
