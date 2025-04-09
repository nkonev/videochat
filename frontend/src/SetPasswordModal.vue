<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480" persistent>
            <v-card :title="$vuetify.locale.t('$vuetify.set_password_for', userName)" :disabled="loading">
                <v-progress-linear
                  :active="loading"
                  :indeterminate="loading"
                  absolute
                  bottom
                  color="primary"
                ></v-progress-linear>

                <v-card-text @keyup.native.enter="onSet">
                  <v-text-field
                      v-model="password"
                      :append-icon="showInputablePassword ? 'mdi-eye' : 'mdi-eye-off'"
                      @click:append="showInputablePassword = !showInputablePassword"
                      @input="hideAlert()"
                      :label="$vuetify.locale.t('$vuetify.password')"
                      required
                      :type="showInputablePassword ? 'text' : 'password'"
                      :rules="[rules.required, rules.min]"
                      :disabled="loading"
                      variant="underlined"
                  ></v-text-field>
                  <v-alert
                      dismissible
                      v-model="isShowAlert"
                      type="error"
                      class="mb-4"
                  >
                    <v-row align="center">
                      <v-col class="grow">{{passwordError}}</v-col>
                    </v-row>
                  </v-alert>

                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-spacer></v-spacer>
                    <v-btn color="primary" variant="flat" @click="onSet()">{{ $vuetify.locale.t('$vuetify.ok') }}</v-btn>
                    <v-btn color="red" variant="flat" @click="onClose()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_SET_PASSWORD_MODAL} from "./bus/bus";
    import axios from "axios";
    import {tryExtractMeaningfulError} from "@/utils.js";
    import userProfileValidationRules from "@/mixins/userProfileValidationRules.js";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore.js";

    export default {
        data () {
            return {
                show: false,
                loading: false,
                password: null,
                showInputablePassword: false,
                userId: null,
                userName: null,
                isShowAlert: false,
                passwordError: "",
            }
        },
        mixins: [
          userProfileValidationRules(),
        ],
        computed: {
          ...mapStores(useChatStore),
        },
        methods: {
            showModal(newData) {
                this.$data.show = true;
                this.$data.userId = newData.userId;
                this.$data.userName = newData.userName;
            },
            onClose() {
                this.$data.show = false;
                this.$data.showInputablePassword = false;
                this.$data.loading = false;
                this.$data.password = null;
                this.$data.userId = null;
                this.$data.userName = null;
                this.hideAlert();
            },
            onSet() {
              this.$data.loading = true;
              axios.put(`/api/aaa/user/${this.$data.userId}/password`, {password: this.password})
                  .then(()=>{
                    this.onClose();
                  }).catch(e => {
                    this.$data.isShowAlert = true;
                    this.$data.passwordError = tryExtractMeaningfulError(e);
                  }).finally(()=>{
                    this.$data.loading = false;
                  })
            },
            hideAlert() {
              this.$data.isShowAlert = false;
              this.$data.passwordError = "";
            },
        },
        mounted() {
            bus.on(OPEN_SET_PASSWORD_MODAL, this.showModal);
        },
        beforeUnmount() {
            bus.off(OPEN_SET_PASSWORD_MODAL, this.showModal);
        },
    }
</script>
