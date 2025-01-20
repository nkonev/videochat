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
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";

export default {
  mixins: [
    contextMenuMixin(),
  ],
  computed: {
    ...mapStores(useChatStore),
  },
  methods:{
    className() {
      return "published-messages-context-menu"
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

        ret.push({title: this.$vuetify.locale.t('$vuetify.open_published_message'), icon: 'mdi-eye', action: () => this.$emit('openPublishedMessage', this.menuableItem) });
        ret.push({title: this.$vuetify.locale.t('$vuetify.copy_public_link_to_message'), icon: 'mdi-content-copy', iconColor: 'primary', action: () => this.$emit('copyLinkToPublishedMessage', this.menuableItem) });
        if (this.canUnpublish(this.menuableItem)) {
          ret.push({title: this.$vuetify.locale.t('$vuetify.unpublish_message'), icon: 'mdi-delete', iconColor: 'red', action: () => this.$emit('unpublishMessage', this.menuableItem) });
        }
      }
      return ret;
    },
    canUnpublish(item) {
      return item.canPublish
    },

  }
}
</script>
