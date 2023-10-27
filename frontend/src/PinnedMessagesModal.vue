<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card :title="$vuetify.locale.t('$vuetify.pinned_messages_full')">
                <v-card-text class="ma-0 pa-0">
                    <v-list class="pb-0" v-if="!loading">
                        <template v-if="dto.totalCount > 0">
                            <template v-for="(item, index) in dto.data">
                                <v-list-item class="list-item-prepend-spacer-16">
                                    <template v-slot:prepend v-if="hasLength(item.owner.avatar)">
                                        <v-avatar :image="item.owner.avatar"></v-avatar>
                                    </template>

                                    <v-list-item-subtitle style="opacity: 1">
                                        <router-link class="colored-link" :to="{ name: 'profileUser', params: { id: item.owner.id }}">{{getOwner(item.owner)}}</router-link><span class="with-space"> {{$vuetify.locale.t('$vuetify.time_at')}} </span>{{getDate(item)}}
                                    </v-list-item-subtitle>
                                    <v-list-item-title>
                                        <router-link :to="getPinnedRouteObject(item)" :class="getItemClass(item)">
                                            {{ item.text }}
                                        </router-link>
                                    </v-list-item-title>

                                    <template v-slot:append>
                                        <v-btn variant="flat" icon @click="promotePinMessage(item)">
                                            <v-icon color="primary" dark :title="$vuetify.locale.t('$vuetify.pin_message')">mdi-pin</v-icon>
                                        </v-btn>
                                        <v-btn variant="flat" icon @click="unpinMessage(item)">
                                            <v-icon color="red" dark :title="$vuetify.locale.t('$vuetify.remove_from_pinned')">mdi-delete</v-icon>
                                        </v-btn>
                                    </template>
                                </v-list-item>
                                <v-divider></v-divider>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.locale.t('$vuetify.no_pin_messages') }}</v-card-text>
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

                    <!-- Pagination is shuddering / flickering on the second page without this wrapper -->
                    <v-row no-gutters class="ma-0 pa-0 d-flex flex-row">
                        <v-col class="ma-0 pa-0 flex-grow-1 flex-shrink-0">
                            <v-pagination
                                variant="elevated"
                                active-color="primary"
                                density="comfortable"
                                v-if="shouldShowPagination"
                                v-model="page"
                                :length="pagesCount"
                                :total-visible="isMobile() ? 3 : 7"
                            ></v-pagination>
                        </v-col>
                        <v-col class="ma-0 pa-0 d-flex flex-row flex-grow-0 flex-shrink-0 align-self-end">
                            <v-btn
                                variant="elevated"
                                color="red"
                                @click="closeModal()"
                            >
                                {{ $vuetify.locale.t('$vuetify.close') }}
                            </v-btn>
                        </v-col>
                    </v-row>
                </v-card-actions>

            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    OPEN_PINNED_MESSAGES_MODAL, PINNED_MESSAGE_PROMOTED, PINNED_MESSAGE_UNPROMOTED,
} from "./bus/bus";
import axios from "axios";
import {getHumanReadableDate, formatSize, findIndex, replaceOrAppend, hasLength, deepCopy} from "./utils";
import {chat_name, messageIdHashPrefix, videochat_name} from "@/router/routes";

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
        pagesCount() {
            const count = Math.ceil(this.dto.totalCount / pageSize);
            // console.debug("Calc pages count", count);
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
        hasLength,
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
        },
        replaceItem(dto) {
            //console.log("Replacing item", dto);
            replaceOrAppend(this.dto.data, [dto]);
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
                    //reset previous promoted
                    this.dto.data.forEach((item)=>{
                        item.pinnedPromoted = false;
                    })
                    const copied = deepCopy(dto.message);
                    this.replaceItem(copied);
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
        getItemClass(item) {
            return {
                "text-primary": true,
                "pinned-text": true,
                'pinned-bold': !!item.pinnedPromoted,
            }
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
    mounted() {
        bus.on(OPEN_PINNED_MESSAGES_MODAL, this.showModal);
        bus.on(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
        bus.on(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);
    },
    beforeUnmount() {
        bus.off(OPEN_PINNED_MESSAGES_MODAL, this.showModal);
        bus.off(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
        bus.off(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);
    },
}
</script>

<style lang="stylus" scoped>
@import "pinned.styl"

.pinned-bold {
    font-weight bold
}

</style>
