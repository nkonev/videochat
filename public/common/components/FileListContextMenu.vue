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
import contextMenuMixin from "../mixins/contextMenuMixin";
import {usePageContext} from "#root/renderer/usePageContext.js";

export default {
  setup() {
    const pageContext = usePageContext();

    // expose to template and other options API hooks
    return {
      pageContext
    }
  },

  mixins: [
    contextMenuMixin(),
  ],
  methods:{
    className() {
      return "file-item-context-menu"
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
            title: 'Close',
            icon: 'mdi-close',
            action: () => {
              this.onCloseContextMenu()
            }
          });
        }

        if (this.menuableItem.canShowAsImage) {
          ret.push({
            title: 'View',
            icon: 'mdi-image',
            action: () => {
              this.$emit('showAsImage', this.menuableItem)
            }
          });
        }
        if (this.menuableItem.canPlayAsVideo) {
          ret.push({
            title: 'Play',
            icon: 'mdi-play',
            action: () => {
              this.$emit('playAsVideo', this.menuableItem)
            }
          });
        }
        if (this.menuableItem.canPlayAsAudio) {
          ret.push({
            title: 'Play',
            icon: 'mdi-play',
            action: () => {
              this.$emit('playAsAudio', this.menuableItem)
            }
          });
        }
      }
      return ret;
    },
    isMobile() {
      return this.pageContext.isMobile
    },
  }
}
</script>
