<template>
    <v-menu
        class="message-item-context-menu"
        :model-value="showContextMenu"
        :transition="false"
        :open-on-click="false"
        :open-on-focus="false"
        :open-on-hover="false"
        :open-delay="0"
        :close-delay="0"
        :close-on-back="false"
    >
        <v-list class="my-m-list">
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
import {getUrlPrefix, hasLength} from "@/utils";
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
            selection: null,
            contextMenuX: 0,
            contextMenuY: 0,
        }
    },

    methods:{
        setPosition() {
          const element = document.querySelector(".message-item-context-menu .v-overlay__content");
          if (element) {
            element.style.position = "absolute";
            element.style.top = this.contextMenuY + "px";
            element.style.left = this.contextMenuX + "px";

            const bottom = Number(getComputedStyle(element).bottom.replace("px", ''));
            if (bottom < 0) {
              const newTop = this.contextMenuY + bottom - 8;
              element.style.top = newTop + "px";
            }
          }
        },
        onShowContextMenu(e, menuableItem) {
            this.showContextMenu = false;
            e.preventDefault();
            this.contextMenuX = e.clientX;
            this.contextMenuY = e.clientY;

            this.menuableItem = menuableItem;
            this.selection = this.getSelection();

            this.$nextTick(()=>{
              this.showContextMenu = true;
            }).then(()=>{
              this.setPosition()
            })
        },
        onCloseContextMenu(){
            this.showContextMenu = false;
            this.menuableItem = null;
            this.selection = null;
        },
        getContextMenuItems() {
            const ret = [];
            if (this.menuableItem) {
                if (hasLength(this.selection)) {
                    ret.push({
                        title: this.$vuetify.locale.t('$vuetify.copy_selected'),
                        icon: 'mdi-content-copy',
                        action: this.copySelected
                    });
                    ret.push({
                        title: this.$vuetify.locale.t('$vuetify.search_by_selected'),
                        icon: 'mdi-clipboard-search-outline',
                        action: this.searchBySelected
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
            const selectedText = this.selection;
            navigator.clipboard.writeText(selectedText);
        },
        searchBySelected() {
            const selectedText = this.selection;
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
