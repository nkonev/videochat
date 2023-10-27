<template>
  <v-sheet max-width="800" class="px-2 pt-2">
      {{ $vuetify.locale.t('$vuetify.registration_pending_confirmation') }}

      <div>
          {{ $vuetify.locale.t('$vuetify.request_resend_confirmation_email_text') }}
          <a class="colored-link" :href="resend()" @click.prevent="onResendClick()">{{$vuetify.locale.t('$vuetify.request_resend_confirmation_email')}}</a>
      </div>

  </v-sheet>
</template>

<script>
import {setTitle} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {registration_resend_email, registration_resend_email_name} from "@/router/routes";

export default {
  computed: {
    ...mapStores(useChatStore),
  },
  methods: {
      resend() {
          return registration_resend_email
      },
      onResendClick() {
          this.$router.push({name: registration_resend_email_name} )
      },
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
