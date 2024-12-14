<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card :title="$vuetify.locale.t('$vuetify.pinned_messages_full')">
                <v-card-text class="ma-0 pa-0">
                    <v-list class="pb-0" v-if="!loading">
                        <template v-if="itemsDto.count > 0">
                            <template v-for="(item, index) in itemsDto.items">
                                <v-list-item
                                    class="list-item-prepend-spacer"
                                    @contextmenu.stop="onShowContextMenu($event, item)"
                                >
                                    <template v-slot:prepend v-if="hasLength(item.owner?.avatar) && !this.isMobile()">
                                        <v-avatar :image="item.owner?.avatar"></v-avatar>
                                    </template>

                                    <v-list-item-subtitle style="opacity: 1">
                                        <router-link class="colored-link" :to="{ name: 'profileUser', params: { id: item.owner?.id }}">{{getOwner(item.owner)}}</router-link><span class="with-space"> {{$vuetify.locale.t('$vuetify.time_at')}} </span><router-link class="gray-link" :to="getPinnedRouteObject(item)">{{getDate(item)}}</router-link>
                                    </v-list-item-subtitle>
                                    <v-list-item-title>
                                        <router-link :to="getPinnedRouteObject(item)" :class="getItemClass(item)">
                                            <div v-html="item.text" class="with-ellipsis"></div>
                                        </router-link>
                                    </v-list-item-title>

                                    <template v-slot:append v-if="canPin(item) && !this.isMobile()">
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

                    <PinnedMessagesContextMenu
                        ref="contextMenuRef"
                        @gotoPinnedMessage="this.gotoPinnedMessage"
                        @promotePinMessage="this.promotePinMessage"
                        @unpinMessage="this.unpinMessage"
                    />

                    <v-progress-circular
                        class="ma-4"
                        v-if="loading"
                        indeterminate
                        color="primary"
                    ></v-progress-circular>

                </v-card-text>

                <v-card-actions class="my-actions d-flex flex-wrap flex-row">

                    <!-- Pagination is shuddering / flickering on the second page without this wrapper -->
                    <v-row no-gutters class="ma-0 pa-0 d-flex flex-row">
                        <v-col class="ma-0 pa-0 flex-grow-1 flex-shrink-0" :class="isMobile() ? 'mb-2' : ''">
                            <v-pagination
                                variant="elevated"
                                active-color="primary"
                                density="comfortable"
                                v-if="shouldShowPagination"
                                v-model="page"
                                :length="pagesCount"
                                :total-visible="getTotalVisible()"
                            ></v-pagination>
                        </v-col>
                        <v-col class="ma-0 pa-0 d-flex flex-row flex-grow-1 flex-shrink-0 align-self-end justify-end">
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
  LOGGED_OUT,
  MESSAGES_RELOAD,
  OPEN_PINNED_MESSAGES_MODAL,
  PINNED_MESSAGE_EDITED,
  PINNED_MESSAGE_PROMOTED,
  PINNED_MESSAGE_UNPROMOTED,
  REFRESH_ON_WEBSOCKET_RESTORED,
} from "./bus/bus";
import axios from "axios";
import { hasLength } from "./utils";
import { getHumanReadableDate } from "@/date.js";
import {chat_name, messageIdHashPrefix, videochat_name} from "@/router/routes";
import pageableModalMixin, {pageSize} from "@/mixins/pageableModalMixin.js";
import debounce from "lodash/debounce.js";
import PinnedMessagesContextMenu from "@/PinnedMessagesContextMenu.vue";

export default {
    components: {
      PinnedMessagesContextMenu,
    },
    mixins: [
        pageableModalMixin()
    ],
    data () {
        return {
            chatId: null,
        }
    },
    methods: {
        hasLength,
        isCachedRelevantToArguments({chatId}) {
            return this.chatId == chatId
        },
        initializeWithArguments({chatId}) {
            this.chatId = chatId;
        },
        initiateRequest() {
            return axios.get(`/api/chat/${this.chatId}/message/pin`, {
                params: {
                    page: this.translatePage(),
                    size: pageSize,
                },
            })
        },
        extractDtoFromEventDto(dto) {
            return [dto.message]
        },
        initiateFilteredRequest(dto) {
            return Promise.resolve({
                data: [
                    {
                        id: dto.message.id
                    }
                ]
            })
        },
        initiateCountRequest(dto) {
            return Promise.resolve({
                data: {
                    count: dto.count
                }
            })
        },
        clearOnClose() {
            // empty
        },
        clearOnReset() {
            this.chatId = null;
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
        onPinnedMessageUnpromoted(dto) {
            this.onItemRemovedEvent(dto);
        },
        onPinnedMessagePromoted(dto) {
            if (this.dataLoaded) {
                // reset previously promoted
                this.itemsDto.items.forEach((item)=>{
                    item.pinnedPromoted = false;
                })
            }

            this.onItemCreatedEvent(dto);
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
        resetOnRouteIdChange() {
            return true
        },
        shouldReactOnPageChange() {
            return this.show
        },
        canPin(item) {
            return item.canPin
        },
        onMessagesReload() {
            this.reset();
            this.closeModal();
        },
        debouncedUpdate() {
          this.updateItems();
        },
        onWsRestoredRefresh() {
          if (this.dataLoaded) {
            this.debouncedUpdate();
          }
        },
        onShowContextMenu(e, menuableItem) {
          this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
        },
        gotoPinnedMessage(item) {
          const routeObj = this.getPinnedRouteObject(item);
          this.$router.push(routeObj)
        },
    },
    created() {
      this.debouncedUpdate = debounce(this.debouncedUpdate, 300, {leading:false, trailing:true})
    },
    mounted() {
        bus.on(OPEN_PINNED_MESSAGES_MODAL, this.showModal);
        bus.on(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
        bus.on(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);
        bus.on(PINNED_MESSAGE_EDITED, this.onItemUpdatedEvent);
        bus.on(LOGGED_OUT, this.onLogout);
        bus.on(MESSAGES_RELOAD, this.onMessagesReload);
        bus.on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    },
    beforeUnmount() {
        bus.off(OPEN_PINNED_MESSAGES_MODAL, this.showModal);
        bus.off(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
        bus.off(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);
        bus.off(PINNED_MESSAGE_EDITED, this.onItemUpdatedEvent);
        bus.off(LOGGED_OUT, this.onLogout);
        bus.off(MESSAGES_RELOAD, this.onMessagesReload);
        bus.off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    },
}
</script>

<style lang="stylus" scoped>
@import "pinned.styl"

.pinned-bold {
    font-weight bold
}

</style>
