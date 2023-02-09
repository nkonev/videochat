<template>
    <v-menu
        v-model="showContextMenu"
        :position-x="contextMenuX"
        :position-y="contextMenuY"
        absolute
        offset-y
        :transition="false"
    >
        <v-list>
            <v-list-item
                v-for="(item, index) in getContextMenuItems()"
                :key="index"
                link
                @click="item.action"
            >
                <v-list-item-avatar><v-icon :color="item.iconColor">{{item.icon}}</v-icon></v-list-item-avatar>
                <v-list-item-title>{{ item.title }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>

<script>

import {chat, messageIdHashPrefix} from "./routes"
import {getUrlPrefix} from "@/utils";

export default {
    props: ['canResend'],
    data(){
        return {
            showContextMenu: false,
            menuableItem: null,
            contextMenuX: 0,
            contextMenuY: 0,
        }
    },
    methods:{
        onShowContextMenu(e, menuableItem) {
            e.preventDefault();
            this.showContextMenu = false;
            this.contextMenuX = e.clientX;
            this.contextMenuY = e.clientY;
            this.menuableItem = menuableItem;
            this.$nextTick(() => {
                this.showContextMenu = true;
            })
        },
        onCloseContextMenu(){
            this.showContextMenu = false
        },
        getContextMenuItems() {
            const ret = [];
            if (this.menuableItem) {
                if (this.menuableItem.fileItemUuid) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.attached_message_files'), icon: 'mdi-file-download', action: () => this.$emit('onFilesClicked', this.menuableItem) });
                }
                if (this.menuableItem.canDelete) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.delete_btn'), icon: 'mdi-delete', iconColor: 'error', action: () => this.$emit('deleteMessage', this.menuableItem) });
                }
                if (this.menuableItem.canEdit) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.edit'), icon: 'mdi-lead-pencil', iconColor: 'primary', action: () => this.$emit('editMessage', this.menuableItem) });
                }
                ret.push({title: this.$vuetify.lang.t('$vuetify.reply'), icon: 'mdi-reply', action: () => this.$emit('replyOnMessage', this.menuableItem) });
                if (this.canResend) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.share'), icon: 'mdi-share', action: () => this.$emit('shareMessage', this.menuableItem) });
                }
                ret.push({title: this.$vuetify.lang.t('$vuetify.copy_link_to_message'), icon: 'mdi-link', action: () => this.copyLink(this.menuableItem) });
            }
            return ret;
        },
        copyLink(item) {
            const link = getUrlPrefix() + chat + '/' + this.chatId + messageIdHashPrefix + item.id;
            navigator.clipboard.writeText(link);
        },
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
    }
}
</script>