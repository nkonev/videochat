<template>
  <v-sheet width="640" class="pl-2 pt-2">
    {{ $vuetify.locale.t('$vuetify.registration_pending_confirmation') }}
  </v-sheet>
</template>

<script>
import {setTitle} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";

export default {
  computed: {
    ...mapStores(useChatStore),
  },
  mounted() {
    this.chatStore.title = this.$vuetify.locale.t('$vuetify.registration_pending_confirmation_title');
    setTitle(this.$vuetify.locale.t('$vuetify.registration_pending_confirmation_title'));
  },
  beforeUnmount() {
    this.chatStore.title = null;
    setTitle(null);
  }
}
</script>
