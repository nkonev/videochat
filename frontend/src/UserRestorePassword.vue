<template>
  <v-sheet width="640" class="pl-2 pt-2">
    <v-form fast-fail @submit.prevent="onSubmit()">
      <v-text-field
        @input="hideAlert()"
        v-model="email"
        :label="$vuetify.locale.t('$vuetify.email')"
        :rules="[rules.required, rules.email]"
        variant="underlined"
      ></v-text-field>

      <v-alert
        v-if="showError"
        density="compact"
        type="error"
        :text="error"
      ></v-alert>

      <v-btn type="submit" color="primary" block class="mt-2">{{ $vuetify.locale.t('$vuetify.request_password_reset') }}</v-btn>
    </v-form>
  </v-sheet>
</template>

<script>
import userProfileValidationRules from "@/mixins/userProfileValidationRules";
import {hasLength, setTitle} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import axios from "axios";
import {confirmation_pending_name, password_restore_check_email_name} from "@/router/routes";

export default {
  mixins: [userProfileValidationRules()],
  data: () => ({
    email: null,
    error: "",
  }),
  computed: {
    ...mapStores(useChatStore),
    showError() {
      return hasLength(this.error)
    }
  },
  methods: {
    onSubmit() {
      axios.post("/api/aaa/request-password-reset", null, { params: {
              email: this.email,
          }})
        .then(() => {
          this.$router.push({name: password_restore_check_email_name} )
        })
        .catch(e => {
          this.error = e.message
        })
    },
    hideAlert() {
      this.error = "";
    },
  },
  mounted() {
    this.chatStore.title = this.$vuetify.locale.t('$vuetify.password_restoration');
    setTitle(this.$vuetify.locale.t('$vuetify.password_restoration'));
  },
  beforeUnmount() {
    this.chatStore.title = null;
    setTitle(null);
    this.error = "";
  }
}
</script>
