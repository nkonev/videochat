<template>
    <v-list-item
        dense
        class="pr-0 pl-4"
    >
    <v-list-item-avatar v-if="item.owner && item.owner.avatar">
        <v-img :src="item.owner.avatar"></v-img>
    </v-list-item-avatar>
    <v-list-item-content @click="onMessageClick(item)" @mousemove="onMessageMouseMove(item)">
        <v-list-item-subtitle>{{getSubtitle(item)}}</v-list-item-subtitle>
        <v-list-item-content class="pre-formatted pa-0" v-html="item.text"></v-list-item-content>
    </v-list-item-content>
    <v-list-item-action>
        <v-container class="mb-0 mt-0 pb-0 pt-0 mx-2 px-1">
            <v-icon class="mr-2" v-if="item.canEdit" color="error" @click="deleteMessage(item)" dark small>mdi-delete</v-icon>
            <v-icon v-if="item.canEdit" color="primary" @click="editMessage(item)" dark small>mdi-lead-pencil</v-icon>
        </v-container>
    </v-list-item-action>
    </v-list-item>
</template>

<script>
    import axios from "axios";
    import bus, {SET_EDIT_MESSAGE} from "./bus";
    import debounce from "lodash/debounce";

    export default {
        props: ['item', 'chatId'],
        methods: {
            onMessageClick(dto) {
                axios.put(`/api/chat/${this.chatId}/message/read/${dto.id}`);
            },
            onMessageMouseMove(item) {
                this.onMessageClick(item);
            },
            deleteMessage(dto){
                axios.delete(`/api/chat/${this.chatId}/message/${dto.id}`)
            },
            editMessage(dto){
                const editMessageDto = {id: dto.id, text: dto.text};
                bus.$emit(SET_EDIT_MESSAGE, editMessageDto);
            },
            getSubtitle(item) {
                return `${item.owner.login} at ${item.createDateTime}`
            },
        },
        created() {
            this.onMessageMouseMove = debounce(this.onMessageMouseMove, 1000, {leading:true, trailing:false});
        },
    }
</script>