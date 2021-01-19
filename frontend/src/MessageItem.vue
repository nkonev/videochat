<template>
    <v-list-item
        dense
        class="pr-1 mr-1 pl-4"
    >
    <v-list-item-avatar v-if="item.owner && item.owner.avatar">
        <v-img :src="item.owner.avatar"></v-img>
    </v-list-item-avatar>
    <v-list-item-content @click="onMessageClick(item)" @mousemove="onMessageMouseMove(item)">
        <v-container class="ma-0 pa-0 d-flex list-item-head">
            {{getSubtitle(item)}}
            <v-icon class="mx-1 ml-2" v-if="item.canEdit" color="error" @click="deleteMessage(item)" dark small>mdi-delete</v-icon>
            <v-icon class="mx-1" v-if="item.canEdit" color="primary" @click="editMessage(item)" dark small>mdi-lead-pencil</v-icon>
        </v-container>
        <v-list-item-content class="pre-formatted pa-0 ma-0 mt-1" v-html="item.text"></v-list-item-content>
    </v-list-item-content>

    </v-list-item>
</template>

<script>
    import axios from "axios";
    import bus, {SET_EDIT_MESSAGE} from "./bus";
    import debounce from "lodash/debounce";
    import { format, parseISO, differenceInDays } from 'date-fns'

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
                const parsedDate = parseISO(item.createDateTime);
                let formatString = 'HH:mm:ss';
                if (differenceInDays(new Date(), parsedDate) >= 1) {
                    formatString = 'd MMM yyyy, ' + formatString;
                }
                return `${item.owner.login} at ${format(parsedDate, formatString)}`
            },
        },
        created() {
            this.onMessageMouseMove = debounce(this.onMessageMouseMove, 1000, {leading:true, trailing:false});
        },
    }
</script>

<style lang="stylus">
  .list-item-head {
    color:rgba(0, 0, 0, .6);
    font-size: .8125rem;
    font-weight: 500;
    line-height: 1rem;
  }
</style>