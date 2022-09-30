<template>
    <v-menu
        v-model="showContextMenu"
        :position-x="contextMenuX"
        :position-y="contextMenuY"
        absolute
        offset-y
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
export default {
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
                if (this.menuableItem.canEdit) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.edit'), icon: 'mdi-lead-pencil', iconColor: 'primary', action: () => this.$emit('editChat', this.menuableItem) });
                }
                if (this.menuableItem.canDelete) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.delete_btn'), icon: 'mdi-delete', iconColor: 'error', action: () => this.$emit('deleteChat', this.menuableItem) });
                }
                if (this.menuableItem.canLeave) {
                    ret.push({title: this.$vuetify.lang.t('$vuetify.leave_btn'), icon: 'mdi-exit-run', action: () => this.$emit('leaveChat', this.menuableItem) });
                }
            }
            return ret;
        }
    }
}
</script>