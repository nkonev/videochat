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
import contextMenuMixin from "@/mixins/contextMenuMixin";
import {videochat_name} from "@/router/routes";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";

export default {
    mixins: [
      contextMenuMixin(),
    ],
    data() {
      return {
        dto: null
      }
    },
    computed: {
      ...mapStores(useChatStore),
    },
    methods:{
        className() {
            return "chat-participants-context-menu"
        },
        onShowContextMenu(e, menuableItem, dto) {
          this.dto = dto;
          this.onShowContextMenuBase(e, menuableItem);
        },
        onCloseContextMenu() {
          this.onCloseContextMenuBase();
          this.dto = null;
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

                if (this.dto.canChangeChatAdmins && this.menuableItem.id != this.chatStore.currentUser.id) {
                  if (this.menuableItem.admin) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.revoke_chat_admin'), icon: 'mdi-crown', iconColor: 'disabled', action: () => this.$emit('changeChatAdmin', this.menuableItem) });
                  } else {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.grant_chat_admin'), icon: 'mdi-crown', iconColor: 'primary', action: () => this.$emit('changeChatAdmin', this.menuableItem) });
                  }
                }

                if (this.dto.canEdit && this.menuableItem.id != this.chatStore.currentUser.id) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.delete_from_chat'), icon: 'mdi-delete', iconColor: 'red', action: () => this.$emit('deleteParticipantFromChat', this.menuableItem) });
                }
                if (this.dto.canVideoKick && this.menuableItem.id != this.chatStore.currentUser.id && this.isVideo()) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.kick'), icon: 'mdi-block-helper', iconColor: 'red', action: () => this.$emit('kickParticipantFromChat', this.menuableItem) });
                }
                if (this.dto.canAudioMute && this.menuableItem.id != this.chatStore.currentUser.id && this.isVideo()) {
                    ret.push({title: this.$vuetify.locale.t('$vuetify.force_mute'), icon: 'mdi-microphone-off', action: () => this.$emit('forceMuteParticipantInChat', this.menuableItem) });
                }
            }
            return ret;
        },
        isVideo() {
          return this.$route.name == videochat_name
        },

    }
}
</script>
