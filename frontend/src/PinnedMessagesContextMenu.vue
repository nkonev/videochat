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
      return "pinned-messages-context-menu"
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

        ret.push({title: this.$vuetify.locale.t('$vuetify.go_to_the_message'), icon: 'mdi-eye', action: () => this.$emit('gotoPinnedMessage', this.menuableItem) });
        if (this.canPin(this.menuableItem)) {
          ret.push({title: this.$vuetify.locale.t('$vuetify.pin_message'), icon: 'mdi-pin', iconColor: 'primary', action: () => this.$emit('promotePinMessage', this.menuableItem) });
          ret.push({title: this.$vuetify.locale.t('$vuetify.remove_from_pinned'), icon: 'mdi-delete', iconColor: 'red', action: () => this.$emit('unpinMessage', this.menuableItem) });
        }
      }
      return ret;
    },
    canPin(item) {
      return item.canPin
    },

  }
}
</script>
