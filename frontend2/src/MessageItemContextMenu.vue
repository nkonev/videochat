<template>
    <v-menu
        v-model="showContextMenu"
        :transition="false"
        :activator="el"
        :open-on-click="false"
        :open-on-focus="false"
        :open-on-hover="false"
        :open-delay="0"
    >
        <v-list>
            <v-list-item
                v-for="(item, index) in getContextMenuItems()"
                :key="index"
                @click="item.action"
            >
              <template v-slot:prepend>
                <v-icon :color="item.iconColor">
                  {{item.icon}}
                </v-icon>
              </template>
              <template v-slot:title>{{ item.title }}</template>
            </v-list-item>
        </v-list>
    </v-menu>
</template>

<script>

import {chat, messageIdHashPrefix} from "./router/routes"
import {deepCopy, getUrlPrefix, hasLength} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {SEARCH_MODE_MESSAGES, searchString} from "@/mixins/searchString";

export default {
    mixins: [
      searchString(SEARCH_MODE_MESSAGES)
    ],
    props: ['canResend', 'isBlog'],
    data(){
        return {
            showContextMenu: false,
            menuableItem: null,
            el: null,
        }
    },
    methods:{
        onShowContextMenu(e, menuableItem, el) {
            e.preventDefault();
            this.showContextMenu = false;
            this.el = el;
            this.menuableItem = menuableItem;
            this.$nextTick(() => {
                this.showContextMenu = true;
            })
        },
        onCloseContextMenu(){
            this.showContextMenu = false;
            this.menuableItem = null;
            this.el = null;
        },
        getContextMenuItems() {
            const ret = [];
            if (this.menuableItem) {
                if (hasLength(this.getSelection())) {
                    ret.push({
                        title: this.$vuetify.locale.t('$vuetify.copy_selected'),
                        icon: 'mdi-content-copy',
                        action: () => this.copySelected()
                    });
                    ret.push({
                        title: this.$vuetify.locale.t('$vuetify.search_by_selected'),
                        icon: 'mdi-clipboard-search-outline',
                        action: () => this.searchBySelected()
                    });
                }
                if (this.menuableItem.fileItemUuid) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.attached_message_files'), icon: 'mdi-file-download', action: () => this.$emit('onFilesClicked', this.menuableItem) });
                }
                if (this.menuableItem.canDelete) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.delete_btn'), icon: 'mdi-delete', iconColor: 'error', action: () => this.$emit('deleteMessage', this.menuableItem) });
                }
                if (this.menuableItem.canEdit) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.edit'), icon: 'mdi-lead-pencil', iconColor: 'primary', action: () => this.$emit('editMessage', this.menuableItem) });
                }
                ret.push({title: this.$vuetify.locale.t('$vuetify.users_read'), icon: 'mdi-account-supervisor', action: () => this.$emit('showReadUsers', this.menuableItem) });
                if (this.menuableItem.pinned) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.remove_from_pinned'), icon: 'mdi-pin-off-outline', action: () => this.$emit('removedFromPinned', this.menuableItem)});
                } else {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.pin_message'), icon: 'mdi-pin', action: () => this.$emit('pinMessage', this.menuableItem)});
                }
                ret.push({title: this.$vuetify.locale.t('$vuetify.reply'), icon: 'mdi-reply', action: () => this.$emit('replyOnMessage', this.menuableItem) });
                if (this.canResend) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.share'), icon: 'mdi-share', action: () => this.$emit('shareMessage', this.menuableItem) });
                }
                ret.push({title: this.$vuetify.locale.t('$vuetify.copy_link_to_message'), icon: 'mdi-link', action: () => this.copyLink(this.menuableItem) });
                if (!this.menuableItem.blogPost && this.isBlog && this.menuableItem.owner.id == this.chatStore.currentUser.id) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.make_blog_post'), icon: 'mdi-postage-stamp', action: () => this.$emit('makeBlogPost', this.menuableItem)});
                }
                if (this.menuableItem.blogPost) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.go_to_blog_post'), icon: 'mdi-postage-stamp', action: () => this.$emit('goToBlog', this.menuableItem)});
                }
            }
            return ret;
        },
        copyLink(item) {
            const link = getUrlPrefix() + chat + '/' + this.chatId + messageIdHashPrefix + item.id;
            navigator.clipboard.writeText(link);
        },
        getSelection() {
            return window.getSelection().toString();
        },
        copySelected() {
            const selectedText = this.getSelection();
            navigator.clipboard.writeText(selectedText);
        },
        searchBySelected() {
            const selectedText = this.getSelection();
            this.searchString = selectedText;
        },
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
        ...mapStores(useChatStore),
    },
}
</script>
