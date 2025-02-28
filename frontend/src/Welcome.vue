<template>
  <v-container fill-height fluid :style="heightWithoutAppBar" v-if="chatStore.currentUser">
    <v-row align="center" justify="center" style="height: 100%">
      <v-card>
        <v-card-title class="d-flex justify-center with-space">{{$vuetify.locale.t('$vuetify.welcome_participant')}}<span :style="getLoginColoredStyle(chatStore.currentUser)">{{chatStore.currentUser?.login}}</span>!</v-card-title>
        <v-card-actions class="d-flex justify-space-around flex-wrap flex-row">
          <v-btn :size="getBtnSize()" @click.prevent="findUser()" text variant="outlined" :href="getUser()">
            <template v-slot:prepend>
              <v-icon :size="getIconSize()">mdi-account-group</v-icon>
            </template>
            <template v-slot:default>
              {{ $vuetify.locale.t('$vuetify.users') }}
            </template>
          </v-btn>
          <v-btn :size="getBtnSize()" color="primary" @click.prevent="createChat()" text variant="outlined">
            <template v-slot:prepend>
              <v-icon :size="getIconSize()">mdi-plus</v-icon>
            </template>
            <template v-slot:default>
              {{ $vuetify.locale.t('$vuetify.new_chat') }}
            </template>
          </v-btn>
          <v-btn :size="getBtnSize()" @click.prevent="chats()" text variant="outlined" :href="getChats()">
            <template v-slot:prepend>
              <v-icon :size="getIconSize()">mdi-forum</v-icon>
            </template>
            <template v-slot:default>
              {{ $vuetify.locale.t('$vuetify.chats') }}
            </template>
          </v-btn>
          <v-btn :size="getBtnSize()" @click.prevent="availableForSearchChats()" text variant="outlined" :href="getAvailableForSearchChats()">
            <template v-slot:prepend>
              <v-icon :size="getIconSize()">mdi-forum</v-icon>
            </template>
            <template v-slot:default>
              {{ $vuetify.locale.t('$vuetify.public_chats') }}
            </template>
          </v-btn>
          <v-btn :size="getBtnSize()" @click.prevent="goBlog()" text variant="outlined" :href="getBlog()">
            <template v-slot:prepend>
              <v-icon :size="getIconSize()">mdi-postage-stamp</v-icon>
            </template>
            <template v-slot:default>
              {{ $vuetify.locale.t('$vuetify.blogs') }}
            </template>
          </v-btn>

          <v-btn :size="getBtnSize()" @click.prevent="enableNotifications()" text variant="outlined">
            <template v-slot:default>
              Enable notifications
            </template>
          </v-btn>
          <v-btn :size="getBtnSize()" @click.prevent="enableServiceWorker()" text variant="outlined">
            <template v-slot:default>
              Enable service worker
            </template>
          </v-btn>
          <v-btn :size="getBtnSize()" @click.prevent="disableServiceWorker()" text variant="outlined">
            <template v-slot:default>
              Disable service worker
            </template>
          </v-btn>
          <v-btn :size="getBtnSize()" @click.prevent="enablePeriodicMessage()" text variant="outlined">
            <template v-slot:default>
              Enable periodic message
            </template>
          </v-btn>
          <v-btn :size="getBtnSize()" @click.prevent="disablePeriodicMessage()" text variant="outlined">
            <template v-slot:default>
              Disable periodic message
            </template>
          </v-btn>

        </v-card-actions>
      </v-card>
    </v-row>
  </v-container>
</template>

<script>
import {getLoginColoredStyle, publicallyAvailableForSearchChatsQuery, setTitle} from "@/utils";
  import {mapStores} from "pinia";
  import {useChatStore} from "@/store/chatStore";
  import heightMixin from "@/mixins/heightMixin";
  import {blog, chat_list_name, chats, profile_list_name, profiles} from "@/router/routes";
  import bus, {OPEN_CHAT_EDIT} from "@/bus/bus";
  import {SEARCH_MODE_CHATS} from "@/mixins/searchString";

  export default {
    data() {
      return {
        interval: null,
      }
    },
    mixins: [
      heightMixin()
    ],
    computed: {
      ...mapStores(useChatStore),
    },
    methods: {
      getLoginColoredStyle,
      createChat() {
        bus.emit(OPEN_CHAT_EDIT, null);
      },
      findUser() {
        this.$router.push({name: profile_list_name});
      },
      getUser() {
        return profiles
      },
      availableForSearchChats() {
        this.$router.push({ name: chat_list_name, hash: null, query: {[SEARCH_MODE_CHATS] : publicallyAvailableForSearchChatsQuery} })
      },
      getAvailableForSearchChats() {
        return chats + "?" + SEARCH_MODE_CHATS + "=" + publicallyAvailableForSearchChatsQuery
      },
      chats() {
        this.$router.push({ name: chat_list_name })
      },
      getChats() {
        return chats
      },
      goBlog() {
        window.location.href = blog
      },
      getBlog() {
        return blog
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
      setTopTitle() {
          this.chatStore.title = this.$vuetify.locale.t('$vuetify.welcome');
          setTitle(this.$vuetify.locale.t('$vuetify.welcome'));
      },

      enableNotifications() {
        Notification.requestPermission()
      },
      async enableServiceWorker() {
        this.registerServiceWorker();
      },
      async disableServiceWorker() {
        this.unRegisterServiceWorker();
      },
      enablePeriodicMessage() {
        this.interval = setInterval(()=>{
          console.warn("Sending a notification");
          this.sendNotification();
        }, 5000)
      },
      disablePeriodicMessage() {
        clearInterval(this.interval)
      },

      async registerServiceWorker() {
        await navigator.serviceWorker.register('/service-worker.js')
        // updateUI();
      },

      getRegistration() {
        return navigator.serviceWorker.getRegistration();
      },

      // Unregister a service worker, then update the UI.
      async unRegisterServiceWorker() {
        // Get a reference to the service worker registration.
        let registration = await this.getRegistration();
        // Await the outcome of the unregistration attempt
        // so that the UI update is not superceded by a
        // returning Promise.
        await registration.unregister();
        // updateUI();
      },

      // Create and send a test notification to the service worker.
      async sendNotification() {
        // Use a random number as part of the notification data
        // (so you can tell the notifications apart during testing!)
        let randy = Math.floor(Math.random() * 100);
        let notification = {
          title: 'Test ' + randy,
          options: { body: 'Test body ' + randy }
        };
        // Get a reference to the service worker registration.
        let registration = await this.getRegistration();
        // Check that the service worker registration exists.
        if (registration) {
          // Check that a service worker controller exists before
          // trying to access the postMessage method.
          if (navigator.serviceWorker.controller) {
            navigator.serviceWorker.controller.postMessage(notification);
          } else {
            console.log('No service worker controller found. Try a soft reload.');
          }
        } else {
          console.log('No service worker registration found. Try a soft reload.');
        }
      },
    },
    watch: {
        '$vuetify.locale.current': {
            handler: function (newValue, oldValue) {
                this.setTopTitle();
            },
        },
    },
    mounted() {
        this.setTopTitle();
    },
    beforeUnmount() {
      setTitle(null);
      this.chatStore.title = null;
    }
  }
</script>
