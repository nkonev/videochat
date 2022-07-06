<template>
    <div class="pr-1 mr-1 pl-4 mt-4 message-item-root" :id="'message-item-' + item.id">
        <router-link :to="{ name: 'profileUser', params: { id: item.owner.id }}">
            <v-list-item-avatar v-if="item.owner && item.owner.avatar">
                <v-img :src="item.owner.avatar"></v-img>
            </v-list-item-avatar>
        </router-link>

        <div @click="onMessageClick(item)" class="message-item-with-buttons-wrapper" @mousemove="onMessageMouseMove(item)">
            <v-container class="ma-0 pa-0 d-flex list-item-head">
                <router-link :to="{ name: 'profileUser', params: { id: item.owner.id }}">{{getOwner(item)}}</router-link><span class="with-space"> {{$vuetify.lang.t('$vuetify.time_at')}} </span>{{getDate(item)}}
                <v-icon class="mx-1 ml-2" v-if="item.fileItemUuid" @click="onFilesClicked(item.fileItemUuid)" small>mdi-file-download</v-icon>
                <v-icon class="mx-1" v-if="item.canEdit" color="error" @click="deleteMessage(item)" dark small>mdi-delete</v-icon>
                <v-icon class="mx-1" v-if="item.canEdit" color="primary" @click="editMessage(item)" dark small>mdi-lead-pencil</v-icon>
            </v-container>
            <div class="pa-0 ma-0 mt-1 message-item-wrapper" :class="{ highlight: highlight }" >
                <v-container v-html="item.text" class="ma-0 pre-formatted message-item-text"></v-container>
            </div>
        </div>
    </div>
</template>

<script>
    import axios from "axios";
    import bus, {
        CLOSE_SIMPLE_MODAL,
        OPEN_SIMPLE_MODAL,
        OPEN_VIEW_FILES_DIALOG,
        OPEN_EDIT_MESSAGE
    } from "./bus";
    import debounce from "lodash/debounce";
    import { format, parseISO, differenceInDays } from 'date-fns';
    import {getData} from "@/centrifugeConnection";
    import {setIcon} from "@/utils";
    import "./messageImage.styl";

    const TYPE_MESSAGE_READ = "message_read";

    export default {
        props: ['item', 'chatId', 'highlight'],
        methods: {
            onMessageClick(dto) {
                this.centrifuge.namedRPC(TYPE_MESSAGE_READ, { chatId: parseInt(this.chatId), messageId: dto.id}).then(value => {
                    if (getData(value)) {
                        const currentNewMessages = getData(value).allUnreadMessages > 0;
                        setIcon(currentNewMessages)
                    }
                })
            },
            onMessageMouseMove(item) {
                this.onMessageClick(item);
            },
            deleteMessage(dto){
                bus.$emit(OPEN_SIMPLE_MODAL, {
                    buttonName: this.$vuetify.lang.t('$vuetify.delete_btn'),
                    title: this.$vuetify.lang.t('$vuetify.delete_message_title', dto.id),
                    text:  this.$vuetify.lang.t('$vuetify.delete_message_text'),
                    actionFunction: ()=> {
                        axios.delete(`/api/chat/${this.chatId}/message/${dto.id}`)
                            .then(() => {
                                bus.$emit(CLOSE_SIMPLE_MODAL);
                            })
                    }
                });
            },
            editMessage(dto){
                const editMessageDto = {id: dto.id, text: dto.text, fileItemUuid: dto.fileItemUuid};
                bus.$emit(OPEN_EDIT_MESSAGE, editMessageDto);
            },
            getOwner(item) {
                return item.owner.login
            },
            getDate(item) {
                const parsedDate = parseISO(item.createDateTime);
                let formatString = 'HH:mm:ss';
                if (differenceInDays(new Date(), parsedDate) >= 1) {
                    formatString = formatString + ', d MMM yyyy';
                }
                return `${format(parsedDate, formatString)}`
            },
            onFilesClicked(itemId) {
                bus.$emit(OPEN_VIEW_FILES_DIALOG, {chatId: this.chatId, fileItemUuid :itemId});
            }
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
  .message-item-root {
      align-items: center;
      display: flex;
      flex: 1 1 100%;
      letter-spacing: normal;
      min-height: 48px;
      outline: none;
      padding: 0 16px;
      padding-right: 16px;
      padding-left: 16px;
      position: relative;
      text-decoration: none;
  }
  .message-item-with-buttons-wrapper {
      flex 1 1
  }
  .message-item-wrapper {
      border-radius 10px
      background #efefef
      display inline-block
      word-wrap break-word
      overflow-wrap break-word
  }
  .message-item-text {
      line-height: 1.1;
      -ms-word-break: break-all;
      /* This is the dangerous one in WebKit, as it breaks things wherever */
      word-break: break-all;
      /* Instead use this non-standard one: */
      word-break: break-word;

      /* Adds a hyphen where the word breaks, if supported (No Blink) */
      -ms-hyphens: auto;
      -moz-hyphens: auto;
      -webkit-hyphens: auto;
      hyphens: auto;
      p {
          margin-bottom unset
      }
      p:empty:after {
          content: '\200b';
      }

      img {
          max-width 100%
      }
  }
  .with-space {
      white-space: pre;
  }
  .highlight {
      background #e4efff
  }

</style>