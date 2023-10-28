<template>
    <v-menu
        :class="className()"
        :model-value="chatStore.contextMenuOpened"
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
import {copyCallLink, copyChatLink, getBlogLink} from "@/utils";
import contextMenuMixin from "@/mixins/contextMenuMixin";

export default {
    mixins: [
      contextMenuMixin(),
    ],
    methods:{
        className() {
            return "chat-item-context-menu"
        },
        onShowContextMenu(e, menuableItem) {
          this.onShowContextMenuBase(e, menuableItem);
        },
        onCloseContextMenu() {
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
                if (!this.menuableItem.isResultFromSearch) {
                    if (this.menuableItem.pinned) {
                        ret.push({
                            title: this.$vuetify.locale.t('$vuetify.remove_from_pinned'),
                            icon: 'mdi-pin-off-outline',
                            action: () => this.$emit('removedFromPinned', this.menuableItem)
                        });
                    } else {
                        ret.push({
                            title: this.$vuetify.locale.t('$vuetify.pin_chat'),
                            icon: 'mdi-pin',
                            action: () => this.$emit('pinChat', this.menuableItem)
                        });
                    }
                }
                if (this.menuableItem.canEdit) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.edit'), icon: 'mdi-lead-pencil', iconColor: 'primary', action: () => this.$emit('editChat', this.menuableItem) });
                }
                if (this.menuableItem.canDelete) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.delete_btn'), icon: 'mdi-delete', iconColor: 'error', action: () => this.$emit('deleteChat', this.menuableItem) });
                }
                if (this.menuableItem.canLeave) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.leave_btn'), icon: 'mdi-exit-run', action: () => this.$emit('leaveChat', this.menuableItem) });
                }
                if (this.menuableItem.blog) {
                  ret.push({title: this.$vuetify.locale.t('$vuetify.go_to_blog_post'), icon: 'mdi-postage-stamp', action: () => this.goToBlog(this.menuableItem) });
                }
                ret.push({title: this.$vuetify.locale.t('$vuetify.copy_link_to_chat'), icon: 'mdi-link', action: () => this.copyLink(this.menuableItem) });
                ret.push({title: this.$vuetify.locale.t('$vuetify.copy_video_call_link'), icon: 'mdi-content-copy', action: () => this.copyCallLink(this.menuableItem) });
            }
            return ret;
        },
        goToBlog(item) {
          window.location.href = getBlogLink(item.id)
        },
        copyLink(item) {
            copyChatLink(item.id)
        },
        copyCallLink(item) {
            copyCallLink(item.id)
        },
    }
}
</script>
