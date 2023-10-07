<template>
  <v-sheet width="640" class="pl-2 pt-2">
    <v-form fast-fail @submit.prevent="onSubmit()">
      <v-text-field
        @input="hideAlert()"
        v-model="login"
        :label="$vuetify.locale.t('$vuetify.login')"
        :rules="[rules.required]"
      ></v-text-field>

      <v-text-field
        @input="hideAlert()"
        v-model="email"
        :label="$vuetify.locale.t('$vuetify.email')"
        :rules="[rules.required, rules.email]"
      ></v-text-field>

      <v-text-field
        @input="hideAlert()"
        v-model="password"
        :label="$vuetify.locale.t('$vuetify.password')"
        :rules="[rules.required, rules.min]"
      ></v-text-field>

      <v-alert
        v-if="showError"
        density="compact"
        type="error"
        :text="error"
      ></v-alert>

      <v-btn type="submit" color="primary" block class="mt-2">Submit</v-btn>
    </v-form>
  </v-sheet>
</template>

<script>
import userProfileValidationRules from "@/mixins/userProfileValidationRules";
import {hasLength, setTitle} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import axios from "axios";
import {confirmation_pending, confirmation_pending_name, root_name} from "@/router/routes";

export default {
  mixins: [userProfileValidationRules()],
  data: () => ({
    login: null,
    email: null,
    password: null,
    error: ""
  }),
  computed: {
    ...mapStores(useChatStore),
    showError() {
      return hasLength(this.error)
    }
  },
  methods: {
    onSubmit() {
      const data = {
        login: this.login,
        email: this.email,
        password: this.password,
      }
      axios.post("/api/register", data)
        .then(() => {
          this.$router.push({name: confirmation_pending_name} )
        })
        .catch(e => {
          this.error = e.message
        })
    },
    hideAlert() {
      this.error = ""
    },
  },
  mounted() {
    this.chatStore.title = this.$vuetify.locale.t('$vuetify.registration');
    setTitle(this.$vuetify.locale.t('$vuetify.registration'));
  },
  beforeUnmount() {
    this.chatStore.title = null;
    setTitle(null);
    this.error = "";
  }
}
</script>
