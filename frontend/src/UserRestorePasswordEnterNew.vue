<template>
  <v-sheet width="640" class="pl-2 pt-2">
    <v-form fast-fail @submit.prevent="onSubmit()">
        <v-text-field
            disabled
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

      <v-alert
        v-if="showError"
        density="compact"
        type="error"
        :text="error"
      ></v-alert>

      <v-btn type="submit" color="primary" block class="mt-2">{{ $vuetify.locale.t('$vuetify.set_new_password') }}</v-btn>
    </v-form>
  </v-sheet>
</template>

<script>
import userProfileValidationRules from "@/mixins/userProfileValidationRules";
import {hasLength, setTitle} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import axios from "axios";
import {confirmation_pending_name, check_email_name, root_name} from "@/router/routes";

export default {
  mixins: [userProfileValidationRules()],
  data: () => ({
    password: null,
    showInputablePassword: false,
    error: "",
  }),
  computed: {
    ...mapStores(useChatStore),
    showError() {
      return hasLength(this.error)
    },
    login() {
      return this.$route.query.login
    },
  },
  methods: {
    onSubmit() {
      axios.post("/api/aaa/password-reset-set-new",{
          passwordResetToken: this.$route.query.uuid,
          newPassword: this.password,
      })
        .then(() => {
          this.$router.push({name: root_name} );
          this.chatStore.fetchUserProfile();
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
    this.showInputablePassword = false;
    this.error = "";
  }
}
</script>
