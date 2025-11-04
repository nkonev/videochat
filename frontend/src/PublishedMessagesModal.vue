<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable :fullscreen="isMobile()">
            <v-card :title="$vuetify.locale.t('$vuetify.published_messages_full')">
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
                                        <router-link class="colored-link" :to="{ name: 'profileUser', params: { id: item.owner?.id }}">{{getOwner(item.owner)}}</router-link><span class="with-space"> {{$vuetify.locale.t('$vuetify.time_at')}} </span><a class="gray-link nodecorated-link" @click.prevent="gotoPublishedMessage(item)" :href="getPublishedHref(item)">{{getDate(item)}}</a>
                                    </v-list-item-subtitle>
                                    <v-list-item-title>
                                        <a @click.prevent="gotoPublishedMessage(item)" :class="getItemClass(item)" :href="getPublishedHref(item)">
                                            <div v-html="item.text" class="with-ellipsis"></div>
                                        </a>
                                    </v-list-item-title>

                                    <template v-slot:append v-if="!this.isMobile()">
                                        <v-btn variant="flat" icon @click="openPublishedMessage(item)">
                                            <v-icon :title="$vuetify.locale.t('$vuetify.open_published_message')">mdi-eye</v-icon>
                                        </v-btn>
                                        <v-btn variant="flat" icon @click="copyLinkToPublishedMessage(item)">
                                            <v-icon color="primary" :title="$vuetify.locale.t('$vuetify.copy_public_link_to_message')">mdi-content-copy</v-icon>
                                        </v-btn>
                                        <v-btn variant="flat" icon @click="unpublishMessage(item)" v-if="canUnpublish(item)">
                                            <v-icon color="red" :title="$vuetify.locale.t('$vuetify.unpublish_message')">mdi-delete</v-icon>
                                        </v-btn>
                                    </template>
                                </v-list-item>
                                <v-divider></v-divider>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.locale.t('$vuetify.no_published_messages') }}</v-card-text>
                        </template>
                    </v-list>

                    <PublishedMessagesContextMenu
                        ref="contextMenuRef"
                        @openPublishedMessage="this.openPublishedMessage"
                        @copyLinkToPublishedMessage="this.copyLinkToPublishedMessage"
                        @unpublishMessage="this.unpublishMessage"
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
                                :density="isMobile() ? 'compact' : 'comfortable'"
                                v-if="shouldShowPagination"
                                v-model="page"
                                :length="pagesCount"
                                :total-visible="getTotalVisible()"
                            ></v-pagination>
                            <v-divider v-if="shouldShowPagination && isMobile()" class="mt-2"/>
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
  LOGGED_OUT, MESSAGES_RELOAD,
  OPEN_PUBLISHED_MESSAGES_MODAL,
  PUBLISHED_MESSAGE_ADD, PUBLISHED_MESSAGE_EDITED, PUBLISHED_MESSAGE_REMOVE, REFRESH_ON_WEBSOCKET_RESTORED,
} from "./bus/bus";
import axios from "axios";
import {getPublicMessageLink, hasLength} from "./utils";
import { getHumanReadableDate } from "@/date.js";
import {chat, chat_name, messageIdHashPrefix, video_suffix, videochat_name} from "@/router/routes";
import pageableModalMixin, {pageSize} from "@/mixins/pageableModalMixin.js";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore.js";
import debounce from "lodash/debounce.js";
import PublishedMessagesContextMenu from "@/PublishedMessagesContextMenu.vue";

export default {
    components: {
      PublishedMessagesContextMenu,
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
        isCachedRelevantToEvent(event) {
          return true
        },
        initializeWithArguments({chatId}) {
            this.chatId = chatId;
        },
        initiateRequest() {
            return axios.get(`/api/chat/${this.chatId}/message/publish`, {
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

        unpublishMessage(dto) {
            axios.put(`/api/chat/${this.chatId}/message/${dto.id}/publish`, null, {
                params: {
                    publish: false
                },
            });
        },
        copyLinkToPublishedMessage(dto) {
            const link = getPublicMessageLink(this.chatId, dto.id)
            navigator.clipboard.writeText(link);
            this.setTempNotification(this.$vuetify.locale.t('$vuetify.published_message_link_copied'));
        },
        getDate(item) {
            return getHumanReadableDate(item.createDateTime)
        },
        getOwner(owner) {
            return owner.login
        },
        isVideoRoute() {
            return this.$route.name == videochat_name
        },
        getItemClass(item) {
            return {
                "text-primary": true,
                "pinned-text": true,
            }
        },
        canUpdateItems() {
          return !!this.chatId
        },
        resetOnRouteIdChange() {
            return true
        },
        shouldReactOnPageChange() {
            return this.show
        },
        canUnpublish(item) {
            return item.canPublish
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
        openPublishedMessage(dto) {
          const link = getPublicMessageLink(this.chatId, dto.id);
          window.open(link, '_blank').focus();
        },
        getPublishedRouteObject(item) {
          const routeName = this.isVideoRoute() ? videochat_name : chat_name;
          return {name: routeName, params: {id: item.chatId}, hash: messageIdHashPrefix + item.id};
        },
        gotoPublishedMessage(item) {
          const routeObj = this.getPublishedRouteObject(item);
          this.$router.push(routeObj).then(()=>{
            if (this.isMobile()) {
              this.closeModal()
            }
          })
        },
        getPublishedHref(item) {
          let bldr = "";
          bldr += chat;
          bldr += "/";
          bldr += item.chatId;
          if (this.isVideoRoute()) {
            bldr += video_suffix;
          }
          bldr += messageIdHashPrefix + item.id;
          return bldr;
        },
    },
    computed: {
        ...mapStores(useChatStore),
    },
    created() {
      this.debouncedUpdate = debounce(this.debouncedUpdate, 300, {leading:false, trailing:true})
    },
    mounted() {
        bus.on(OPEN_PUBLISHED_MESSAGES_MODAL, this.showModal);
        bus.on(PUBLISHED_MESSAGE_ADD, this.onItemCreatedEvent);
        bus.on(PUBLISHED_MESSAGE_REMOVE, this.onItemRemovedEvent);
        bus.on(PUBLISHED_MESSAGE_EDITED, this.onItemUpdatedEvent);
        bus.on(LOGGED_OUT, this.onLogout);
        bus.on(MESSAGES_RELOAD, this.onMessagesReload);
        bus.on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    },
    beforeUnmount() {
        bus.off(OPEN_PUBLISHED_MESSAGES_MODAL, this.showModal);
        bus.off(PUBLISHED_MESSAGE_ADD, this.onItemCreatedEvent);
        bus.off(PUBLISHED_MESSAGE_REMOVE, this.onItemRemovedEvent);
        bus.off(PUBLISHED_MESSAGE_EDITED, this.onItemUpdatedEvent);
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
