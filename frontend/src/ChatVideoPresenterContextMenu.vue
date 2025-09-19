<template>
    <v-menu
        attach="#video-splitpanes"
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
                :disabled="!item.enabled"
            >
              <template v-slot:prepend>
                <v-icon v-if="item.icon" :color="item.iconColor">
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
import bus, {PIN_VIDEO, UN_PIN_VIDEO} from "@/bus/bus.js";

export default {
    mixins: [
      contextMenuMixin(),
    ],
    computed: {
        ...mapStores(useChatStore),
    },
    props: [
        'userName'
    ],
    methods:{
        className() {
            return "presenter-context-menu"
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

                ret.push({
                    title: this.userName,
                    icon: 'mdi-close',
                    action: () => {
                        this.onCloseContextMenu()
                    },
                    enabled: true,
                });

                if (!this.chatStore.pinnedTrackSid) {
                  ret.push({
                    title: this.$vuetify.locale.t('$vuetify.pin_video'),
                    icon: 'mdi-pin',
                    action: () => {
                      bus.emit(PIN_VIDEO, this.menuableItem.getPresenterVideoStreamId())
                    },
                    enabled: true,
                  });
                } else {
                  ret.push({
                    title: this.$vuetify.locale.t('$vuetify.unpin_video'),
                    icon: 'mdi-pin-off-outline',
                    action: () => {
                      bus.emit(UN_PIN_VIDEO)
                    },
                    enabled: true,
                  });
                }

            }
            return ret;
        },
    }
}
</script>
