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
import {copyCallLink, copyChatLink, getBlogLink, getChatLink} from "@/utils";
import contextMenuMixin from "@/mixins/contextMenuMixin";

export default {
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
            title: this.$vuetify.locale.t('$vuetify.close'),
            icon: 'mdi-close',
            action: () => {
              this.onCloseContextMenu()
            }
          });
        }
        if (!this.menuableItem.hasNoMessage) {
          ret.push({
            title: this.$vuetify.locale.t('$vuetify.search_related_message'),
            icon: 'mdi-note-search-outline',
            action: () => {
              this.$emit('searchRelatedMessage', this.menuableItem)
            }
          });
        }
        if (this.menuableItem.canShowAsImage) {
          ret.push({
            title: this.$vuetify.locale.t('$vuetify.view'),
            icon: 'mdi-image',
            action: () => {
              this.$emit('showAsImage', this.menuableItem)
            }
          });
        }
        if (this.menuableItem.canPlayAsVideo) {
          ret.push({
            title: this.$vuetify.locale.t('$vuetify.play'),
            icon: 'mdi-play',
            action: () => {
              this.$emit('playAsVideo', this.menuableItem)
            }
          });
        }
        if (this.menuableItem.canPlayAsAudio) {
          ret.push({
            title: this.$vuetify.locale.t('$vuetify.play'),
            icon: 'mdi-play',
            action: () => {
              this.$emit('playAsAudio', this.menuableItem)
            }
          });
        }
        if (this.menuableItem.canEdit) {
          ret.push({
            title: this.$vuetify.locale.t('$vuetify.edit'),
            icon: 'mdi-pencil',
            action: () => {
              this.$emit('edit', this.menuableItem)
            }
          });
        }

        if (this.menuableItem.canShare) {
          if (!this.menuableItem.publicUrl) {
            ret.push({
              title: this.$vuetify.locale.t('$vuetify.share_file'),
              icon: 'mdi-export',
              action: () => {
                this.$emit('share', this.menuableItem)
              }
            });
          } else {
            ret.push({
              title: this.$vuetify.locale.t('$vuetify.unshare_file'),
              icon: 'mdi-lock',
              action: () => {
                this.$emit('unshare', this.menuableItem)
              }
            });
          }
        }
        if (this.menuableItem.canDelete) {
          ret.push({
            title: this.$vuetify.locale.t('$vuetify.delete_btn'),
            icon: 'mdi-delete',
            iconColor: 'red',
            action: () => {
              this.$emit('delete', this.menuableItem)
            }
          });
        }
      }
      return ret;
    },
  }
}
</script>
