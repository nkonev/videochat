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
import {GET_SEARCH_STRING, GET_USER, SET_SEARCH_STRING} from "@/store";
import {mapGetters} from "vuex";

export default {
    props: ['canResend', 'isBlog'],
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
                ret.push({title: this.$vuetify.lang.t('$vuetify.copy_selected'), icon: 'mdi-content-copy', action: () => this.copySelected() });
                ret.push({title: this.$vuetify.lang.t('$vuetify.search_by_selected'), icon: 'mdi-clipboard-search-outline', action: () => this.searchBySelected() });
                if (this.menuableItem.fileItemUuid) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.attached_message_files'), icon: 'mdi-file-download', action: () => this.$emit('onFilesClicked', this.menuableItem) });
                }
                if (this.menuableItem.canDelete) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.delete_btn'), icon: 'mdi-delete', iconColor: 'error', action: () => this.$emit('deleteMessage', this.menuableItem) });
                }
                if (this.menuableItem.canEdit) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.edit'), icon: 'mdi-lead-pencil', iconColor: 'primary', action: () => this.$emit('editMessage', this.menuableItem) });
                }
                ret.push({title: this.$vuetify.lang.t('$vuetify.users_read'), icon: 'mdi-account-supervisor', action: () => this.$emit('showReadUsers', this.menuableItem) });
                if (this.menuableItem.pinned) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.remove_from_pinned'), icon: 'mdi-pin-off-outline', action: () => this.$emit('removedFromPinned', this.menuableItem)});
                } else {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.pin_message'), icon: 'mdi-pin', action: () => this.$emit('pinMessage', this.menuableItem)});
                }
                ret.push({title: this.$vuetify.lang.t('$vuetify.reply'), icon: 'mdi-reply', action: () => this.$emit('replyOnMessage', this.menuableItem) });
                if (this.canResend) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.share'), icon: 'mdi-share', action: () => this.$emit('shareMessage', this.menuableItem) });
                }
                ret.push({title: this.$vuetify.lang.t('$vuetify.copy_link_to_message'), icon: 'mdi-link', action: () => this.copyLink(this.menuableItem) });
                if (!this.menuableItem.blogPost && this.isBlog && this.menuableItem.owner.id == this.currentUser.id) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.make_blog_post'), icon: 'mdi-postage-stamp', action: () => this.$emit('makeBlogPost', this.menuableItem)});
                }
                if (this.menuableItem.blogPost) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.go_to_blog_post'), icon: 'mdi-postage-stamp', action: () => this.$emit('goToBlog', this.menuableItem)});
                }
            }
            return ret;
        },
        copyLink(item) {
            const link = getUrlPrefix() + chat + '/' + this.chatId + messageIdHashPrefix + item.id;
            navigator.clipboard.writeText(link);
        },
        copySelected() {
            const selectedText = window.getSelection().toString();
            navigator.clipboard.writeText(selectedText);
        },
        searchBySelected() {
            const selectedText = window.getSelection().toString();
            this.searchString = selectedText;
        },
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
        ...mapGetters({
            currentUser: GET_USER,
        }),
        searchString: {
            get(){
                return this.$store.getters[GET_SEARCH_STRING];
            },
            set(newVal){
                this.$store.commit(SET_SEARCH_STRING, newVal);
                return newVal;
            }
        }
    },
}
</script>
