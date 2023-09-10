<template>
  <v-container fill-height fluid :style="heightWithoutAppBar">
    <v-row align="center" justify="center" style="height: 100%">
      <v-card>
        <v-card-title class="d-flex justify-space-around">{{$vuetify.locale.t('$vuetify.welcome_participant', chatStore.currentUser?.login)}}</v-card-title>
        <v-card-actions class="d-flex justify-space-around flex-wrap flex-row pb-0">
          <v-btn color="primary" @click="createChat()" text>
            <v-icon>mdi-plus</v-icon>
            {{ $vuetify.locale.t('$vuetify.new_chat') }}
          </v-btn>
          <v-btn @click="findUser()" text>
            <v-icon>mdi-magnify</v-icon>
            {{ $vuetify.locale.t('$vuetify.find_user') }}
          </v-btn>
          <v-btn @click="availableChats()" text>
            <v-icon>mdi-forum</v-icon>
            {{ $vuetify.locale.t('$vuetify.public_chats') }}
          </v-btn>
          <v-btn @click="goBlog()" text>
            <v-icon>mdi-postage-stamp</v-icon>
            {{ $vuetify.locale.t('$vuetify.blogs') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-row>
  </v-container>
</template>

<script>
import {publicallyAvailableForSearchChatsQuery, setTitle} from "@/utils";
  import {mapStores} from "pinia";
  import {useChatStore} from "@/store/chatStore";
  import heightMixin from "@/mixins/heightMixin";
import {blog, chat_list_name} from "@/router/routes";
  import bus, {OPEN_CHAT_EDIT, OPEN_FIND_USER} from "@/bus/bus";
import {SEARCH_MODE_CHATS} from "@/mixins/searchString";

  export default {
    mixins: [
      heightMixin()
    ],
    computed: {
      ...mapStores(useChatStore),
    },
    methods: {
      createChat() {
        bus.emit(OPEN_CHAT_EDIT, null);
      },
      findUser() {
        bus.emit(OPEN_FIND_USER)
      },
      availableChats() {
        this.$router.push({ name: chat_list_name, hash: null, query: {[SEARCH_MODE_CHATS] : publicallyAvailableForSearchChatsQuery} })
      },
      goBlog() {
        window.location.href = blog
      },
    },
    mounted() {
      this.chatStore.title = this.$vuetify.locale.t('$vuetify.welcome');
      setTitle(this.$vuetify.locale.t('$vuetify.welcome'));
    },
    beforeUnmount() {
      setTitle(null);
      this.chatStore.title = null;
    }
  }
</script>
