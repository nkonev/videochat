<template>
  <v-sheet max-width="800" class="px-2 pt-2">
    <v-form fast-fail @submit.prevent="onSubmit()">
      <v-text-field
        @input="hideAlert()"
        v-model="login"
        :label="$vuetify.locale.t('$vuetify.login')"
        :rules="[rules.required]"
        variant="underlined"
      ></v-text-field>

      <v-text-field
        @input="hideAlert()"
        v-model="password"
        :type="showInputablePassword ? 'text' : 'password'"
        :label="$vuetify.locale.t('$vuetify.password')"
        :rules="[rules.required, rules.min]"
        variant="underlined"
      >
        <template v-slot:append>
          <v-icon @click="showInputablePassword = !showInputablePassword" class="mx-1 ml-3">{{showInputablePassword ? 'mdi-eye' : 'mdi-eye-off'}}</v-icon>
        </template>
      </v-text-field>

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

      <v-btn type="submit" color="primary" block class="mt-2">{{ $vuetify.locale.t('$vuetify.registration_submit') }}</v-btn>
    </v-form>

    <div class="mt-2">
        {{ $vuetify.locale.t('$vuetify.request_resend_confirmation_email_text') }}
        <a class="colored-link" :href="resend()" @click.prevent="onResendClick()">{{$vuetify.locale.t('$vuetify.request_resend_confirmation_email_full')}}</a>
    </div>
  </v-sheet>
</template>

<script>
import userProfileValidationRules from "@/mixins/userProfileValidationRules";
import {hasLength, setTitle, tryExtractMeaningfulError} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import axios from "axios";
import {confirmation_pending_name, registration_resend_email, registration_resend_email_name} from "@/router/routes";

export default {
  mixins: [userProfileValidationRules()],
  data: () => ({
    login: null,
    email: null,
    password: null,
    error: "",
    showInputablePassword: false,
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
      axios.post("/api/aaa/register", data, { params: {
              language: this.$vuetify.locale.current,
              referer: this.$route.query.referer
          }})
        .then(() => {
          this.$router.push({name: confirmation_pending_name} )
        })
        .catch(e => {
          this.error = tryExtractMeaningfulError(e)
        })
    },
    hideAlert() {
      this.error = "";
    },
    resend() {
        return registration_resend_email
    },
    onResendClick() {
        this.$router.push({name: registration_resend_email_name} )
    },
    setTopTitle() {
        this.chatStore.title = this.$vuetify.locale.t('$vuetify.registration');
        setTitle(this.$vuetify.locale.t('$vuetify.registration'));
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
    this.chatStore.title = null;
    setTitle(null);
    this.hideAlert();
    this.showInputablePassword = false;
  }
}
</script>
