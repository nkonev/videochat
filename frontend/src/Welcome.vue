<template>
  <v-container fill-height fluid :style="heightWithoutAppBar" v-if="chatStore.currentUser">
    <v-row align="center" justify="center" style="height: 100%">
      <v-card>
        <v-card-title class="d-flex justify-space-around">{{$vuetify.locale.t('$vuetify.welcome_participant', chatStore.currentUser?.login)}}</v-card-title>
        <v-card-actions class="d-flex justify-space-around flex-wrap flex-row pb-0">
          <v-btn :size="getBtnSize()" @click="findUser()" text :class="isMobile() ? 'my-2' : ''">
            <v-icon :size="getIconSize()">mdi-account-group</v-icon>
            {{ $vuetify.locale.t('$vuetify.users') }}
          </v-btn>
          <v-btn :size="getBtnSize()" color="primary" @click="createChat()" text :class="isMobile() ? 'my-2' : ''">
            <v-icon :size="getIconSize()">mdi-plus</v-icon>
            {{ $vuetify.locale.t('$vuetify.new_chat') }}
          </v-btn>
          <v-btn :size="getBtnSize()" @click="chats()" text :class="isMobile() ? 'my-2' : ''">
            <v-icon :size="getIconSize()">mdi-forum</v-icon>
            {{ $vuetify.locale.t('$vuetify.chats') }}
          </v-btn>
          <v-btn :size="getBtnSize()" @click="availableForSearchChats()" text :class="isMobile() ? 'my-2' : ''">
            <v-icon :size="getIconSize()">mdi-forum</v-icon>
            {{ $vuetify.locale.t('$vuetify.public_chats') }}
          </v-btn>
          <v-btn :size="getBtnSize()" @click="goBlog()" text :class="isMobile() ? 'my-2' : ''">
            <v-icon :size="getIconSize()">mdi-postage-stamp</v-icon>
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
  import {blog, chat_list_name, profile_list_name} from "@/router/routes";
  import bus, {OPEN_CHAT_EDIT} from "@/bus/bus";
  import {goToPreserving, SEARCH_MODE_CHATS} from "@/mixins/searchString";

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
        goToPreserving(this.$route, this.$router, { name: profile_list_name});
      },
      availableForSearchChats() {
        this.$router.push({ name: chat_list_name, hash: null, query: {[SEARCH_MODE_CHATS] : publicallyAvailableForSearchChatsQuery} })
      },
      chats() {
        this.$router.push({ name: chat_list_name })
      },
      goBlog() {
        window.location.href = blog
      },

      getBtnSize() {
        if (this.isMobile()) {
            return 'large'
        } else {
            return undefined
        }
      },
      getIconSize() {
        if (this.isMobile()) {
            return 'large'
        } else {
            return undefined
        }
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
