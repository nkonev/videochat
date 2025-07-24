<template>
    <v-menu
        :class="className()"
        :model-value="showContextMenu"
        :transition="false"
        :open-on-click="false"
        :open-on-focus="false"
        :open-on-hover="false"
        :open-delay="0"
        :close-delay="0"
        :close-on-back="false"
        @update:modelValue="onUpdate"
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

import {checkUpByTreeObj, getMessageLink, getPublicMessageLink, hasLength} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {SEARCH_MODE_MESSAGES, searchString} from "@/mixins/searchString";
import contextMenuMixin from "@/mixins/contextMenuMixin";

export default {
    mixins: [
      searchString(SEARCH_MODE_MESSAGES),
      contextMenuMixin(),
    ],
    props: ['canResend', 'isBlog'],
    data(){
      return {
        selection: null,
      }
    },
    methods:{
        className() {
          return "message-item-context-menu"
        },
        onShowContextMenu(e, menuableItem) {
          this.selection = this.getSelection();
          this.onShowContextMenuBase(e, menuableItem);
        },
        onCloseContextMenu() {
          this.selection = null;
          this.onCloseContextMenuBase();
        },
        getContextMenuItems() {
            const ret = [];
            if (this.menuableItem) {
                if (this.isMobile()) {
                    ret.push({
                        title: this.$vuetify.locale.t('$vuetify.close'),
                        icon: 'mdi-close',
                        action: () => {
                            this.onCloseContextMenu()
                        }
                    });
                }
                if (hasLength(this.selection)) {
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
                if (hasLength(this.selection)) {
                  ret.push({
                    title: this.$vuetify.locale.t('$vuetify.copy_selected'),
                    icon: 'mdi-content-copy',
                    action: this.copySelected
                  });
                }
                if (!this.isMobile()) {
                  ret.push({
                    title: this.$vuetify.locale.t('$vuetify.copy'),
                    icon: 'mdi-content-copy',
                    action: this.copy
                  });
                }
                ret.push({
                  title: this.$vuetify.locale.t('$vuetify.copy_text'),
                  icon: 'mdi-content-copy',
                  action: this.copyText
                });
                ret.push({title: this.$vuetify.locale.t('$vuetify.users_read'), icon: 'mdi-account-supervisor', action: () => this.$emit('showReadUsers', this.menuableItem) });
                if (this.menuableItem.canPin) {
                    if (this.menuableItem.pinned) {
                        ret.push({
                            title: this.$vuetify.locale.t('$vuetify.remove_from_pinned'),
                            icon: 'mdi-pin-off-outline',
                            action: () => this.$emit('removedFromPinned', this.menuableItem)
                        });
                    } else {
                        ret.push({
                            title: this.$vuetify.locale.t('$vuetify.pin_message'),
                            icon: 'mdi-pin',
                            action: () => this.$emit('pinMessage', this.menuableItem)
                        });
                    }
                }
                ret.push({title: this.$vuetify.locale.t('$vuetify.reply'), icon: 'mdi-reply', action: () => this.$emit('replyOnMessage', this.menuableItem) });
                if (this.canResend) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.resend'), icon: 'mdi-share', action: () => this.$emit('shareMessage', this.menuableItem) });
                }
                ret.push({title: this.$vuetify.locale.t('$vuetify.copy_link_to_message'), icon: 'mdi-link', action: () => this.copyLink(this.menuableItem) });
                if (!this.menuableItem.blogPost && this.isBlog && this.menuableItem.owner?.id == this.chatStore.currentUser.id) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.make_blog_post'), icon: 'mdi-postage-stamp', action: () => this.$emit('makeBlogPost', this.menuableItem)});
                }
                if (this.isBlog) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.go_to_blog_post'), icon: 'mdi-postage-stamp', action: () => this.$emit('goToBlog', this.menuableItem)});
                }
                if (this.areReactionsAllowed) {
                  ret.push({
                    title: this.$vuetify.locale.t('$vuetify.add_reaction_on_message'),
                    icon: 'mdi-emoticon-outline',
                    action: () => this.$emit('addReaction', this.menuableItem)
                  });
                }
                if (this.menuableItem.canPublish) {
                    if (this.menuableItem.published) {
                        ret.push({
                            title: this.$vuetify.locale.t('$vuetify.copy_public_link_to_message'),
                            icon: 'mdi-link',
                            action: () => this.copyPublicLink(this.menuableItem)
                        });
                        ret.push({
                            title: this.$vuetify.locale.t('$vuetify.unpublish_message'),
                            icon: 'mdi-lock',
                            action: () => this.$emit('removePublic', this.menuableItem)
                        });
                    } else {
                        ret.push({
                            title: this.$vuetify.locale.t('$vuetify.publish_message'),
                            icon: 'mdi-export',
                            action: () => this.$emit('publishMessage', this.menuableItem)
                        });
                    }
                }
            }
            return ret;
        },
        copyLink(item) {
            const link = getMessageLink(this.chatId, item.id)
            navigator.clipboard.writeText(link);
            this.setTempNotification(this.$vuetify.locale.t('$vuetify.message_link_copied'));
        },
        copyPublicLink(item) {
            const link = getPublicMessageLink(this.chatId, item.id)
            navigator.clipboard.writeText(link);
            this.setTempNotification(this.$vuetify.locale.t('$vuetify.published_message_link_copied'));
        },
        getSelection() {
            return window.getSelection().toString();
        },
        async copy() {
          try {
            if (this.targetEl) {
              const {
                found,
                el
              } = checkUpByTreeObj(this.targetEl, 10, (el) => el?.classList?.contains("message-item-wrapper"));
              if (found) {
                const type = "text/html";
                const blob = new Blob([el.innerHTML], {type: type});
                const clipboardItem = new ClipboardItem({
                  [blob.type]: blob,
                });
                await navigator.clipboard.write([clipboardItem]);

                this.setTempNotification(this.$vuetify.locale.t('$vuetify.message_copied'));
              } else {
                this.setWarning("element is not found")
              }
            }
          } catch (e) {
            this.setError(e, "unable to copy")
          }
        },
        copyText() {
          try {
            if (this.targetEl) {
              const {
                found,
                el
              } = checkUpByTreeObj(this.targetEl, 10, (el) => el?.classList?.contains("message-item-wrapper"));
              if (found) {
                navigator.clipboard.writeText(el.textContent);
                this.setTempNotification(this.$vuetify.locale.t('$vuetify.message_copied'));
              } else {
                this.setWarning("element is not found")
              }
            }
          } catch (e) {
            this.setError(e, "unable to copy")
          }
        },
        copySelected() {
          try {
            const selectedText = this.selection;
            navigator.clipboard.writeText(selectedText);
            this.setTempNotification(this.$vuetify.locale.t('$vuetify.message_copied'));
          } catch (e) {
            this.setError(e, "unable to copy")
          }
        },
        searchBySelected() {
            const selectedText = this.selection;
            this.searchString = selectedText;
            this.chatStore.searchType = SEARCH_MODE_MESSAGES;
        },
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
        ...mapStores(useChatStore),
        areReactionsAllowed() {
          return this.chatStore.chatDto.canReact
        },
    },
}
</script>
