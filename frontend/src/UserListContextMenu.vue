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
import {copyCallLink, copyChatLink, getBlogLink} from "@/utils";
import contextMenuMixin from "@/mixins/contextMenuMixin";

export default {
    mixins: [
      contextMenuMixin(),
    ],
    methods:{
        className() {
            return "user-list-item-context-menu"
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
                ret.push({title: this.$vuetify.locale.t('$vuetify.user_open_chat'), icon: 'mdi-message-text-outline', action: () => this.$emit('tetATet', this.menuableItem) });
                if (this.menuableItem.canLock){
                    if (this.menuableItem?.additionalData.locked) {
                        ret.push({title: this.$vuetify.locale.t('$vuetify.unlock_user'), icon: 'mdi-lock-open-outline', action: () => this.$emit('unlockUser', this.menuableItem) });
                    } else {
                        ret.push({title: this.$vuetify.locale.t('$vuetify.lock_user'), icon: 'mdi-lock', action: () => this.$emit('lockUser', this.menuableItem) });
                    }
                }
                if (this.menuableItem.canConfirm){
                    if (this.menuableItem?.additionalData.confirmed) {
                        ret.push({title: this.$vuetify.locale.t('$vuetify.unconfirm_user'), icon: 'mdi-close-thick', action: () => this.$emit('unconfirmUser', this.menuableItem) });
                    } else {
                        ret.push({title: this.$vuetify.locale.t('$vuetify.confirm_user'), icon: 'mdi-check-bold', action: () => this.$emit('confirmUser', this.menuableItem) });
                    }
                }
                if (this.menuableItem.canDelete){
                    ret.push({title: this.$vuetify.locale.t('$vuetify.delete_user'), icon: 'mdi-delete', iconColor: 'error', action: () => this.$emit('deleteUser', this.menuableItem) });
                }
            }
            return ret;
        },
    }
}
</script>
