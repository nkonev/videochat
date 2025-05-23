<template>
    <!--
    https://vuetifyjs.com/en/components/dialogs/#dialogs
    https://vuetifyjs.com/en/components/forms/
    -->
    <v-row justify="center">
        <v-dialog persistent v-model="show" max-width="440">
            <v-card>
                <v-card-title class="d-flex flex-row align-center">
                  <span class="d-flex flex-grow-1">
                      {{ $vuetify.locale.t('$vuetify.login_title') }}
                  </span>
                  <span class="d-flex">
                      <v-btn
                          class="ml-2"
                          @click="onLanguageClick"
                          variant="plain"
                          icon
                          :title="$vuetify.locale.t('$vuetify.language')"
                      >
                          <v-icon>mdi-translate</v-icon>
                      </v-btn>
                  </span>

                </v-card-title>

                <v-card-text :class="isMobile() ? 'pa-4 py-0' : 'pl-4 py-0'">
                    <v-form
                        ref="form"
                        lazy-validation
                        @keyup.native.enter="loginWithLogin"
                        class="pb-2"
                    >
                        <v-text-field
                                id="login-text"
                                v-model="login"
                                :rules="loginRules"
                                :label="$vuetify.locale.t('$vuetify.login')"
                                required
                                :disabled="disable"
                                @input="hideAlert()"
                                variant="underlined"
                        ></v-text-field>

                        <v-text-field
                                id="password-text"
                                v-model="password"
                                :append-icon="showInputablePassword ? 'mdi-eye' : 'mdi-eye-off'"
                                @click:append="showInputablePassword = !showInputablePassword"
                                :rules="passwordRules"
                                :label="$vuetify.locale.t('$vuetify.password')"
                                required
                                :type="showInputablePassword ? 'text' : 'password'"
                                :disabled="disable"
                                @input="hideAlert()"
                                variant="underlined"
                        ></v-text-field>

                        <v-alert
                                dismissible
                                v-model="isShowAlert"
                                type="error"
                                class="mb-4"
                        >
                            <v-row align="center">
                                <v-col class="grow">{{loginError}}</v-col>
                            </v-row>
                        </v-alert>

                        <v-btn
                                id="login-btn"
                                :disabled="disable"
                                color="success"
                                class="mr-2 mb-2"
                                @click="loginWithLogin"
                                min-width="80px"
                                :loading="loadingLogin"
                        >
                            {{ $vuetify.locale.t('$vuetify.login_action') }}
                        </v-btn>

                        <v-btn v-if="chatStore.availableOAuth2Providers.includes('vkontakte')" class="mr-2 mb-2 c-btn-vk" :disabled="disable" :loading="loadingVk" min-width="80px" @click="loginVk()">
                            <font-awesome-icon :icon="[ 'fab', 'vk']" :size="'2x'"></font-awesome-icon>
                        </v-btn>
                        <v-btn v-if="chatStore.availableOAuth2Providers.includes('facebook')" class="mr-2 mb-2 c-btn-fb" :disabled="disable" :loading="loadingFb" min-width="80px" @click="loginFb()">
                            <font-awesome-icon :icon="[ 'fab', 'facebook' ]" :size="'2x'"></font-awesome-icon>
                        </v-btn>
                        <v-btn v-if="chatStore.availableOAuth2Providers.includes('google')" class="mr-2 mb-2 c-btn-google" :disabled="disable" :loading="loadingGoogle" min-width="80px" @click="loginGoogle()">
                            <font-awesome-icon :icon="[ 'fab', 'google' ]" :size="'2x'"></font-awesome-icon>
                        </v-btn>
                        <v-btn v-if="chatStore.availableOAuth2Providers.includes('keycloak')" class="mr-2 mb-2 c-btn-keycloak" :disabled="disable" :loading="loadingKeycloak" min-width="80px" @click="loginKeycloak()">
                            <font-awesome-icon :icon="['fa', 'key' ]" :size="'2x'"></font-awesome-icon>
                        </v-btn>
                    </v-form>

                    <v-divider/>
                    <div class="my-2">
                      <a :href="registration()" class="colored-link" @click.prevent="onRegisterClick">{{ $vuetify.locale.t('$vuetify.registration') }}</a>
                      <span>{{ $vuetify.locale.t('$vuetify.or') }}</span>
                      <a :href="forgot_password()" class="colored-link" @click.prevent="onForgotPasswordClick">{{ $vuetify.locale.t('$vuetify.forgot_password') }}</a>
                    </div>
                    <v-btn class="mb-4" @click.prevent="goBlog()" text variant="outlined" :href="getBlog()">
                      <template v-slot:prepend>
                        <v-icon >mdi-postage-stamp</v-icon>
                      </template>
                      <template v-slot:default>
                        {{ $vuetify.locale.t('$vuetify.blogs') }}
                      </template>
                    </v-btn>

                </v-card-text>

            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {LOGGED_IN, LOGGED_OUT, OPEN_SETTINGS} from "./bus/bus";
    import axios from "axios";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import {
      forgot_password,
      forgot_password_name,
      registration,
      registration_name, blog
    } from "@/router/routes";
    import {setLanguageToVuetify, tryExtractMeaningfulError} from "@/utils";
    import {getStoredLanguage} from "@/store/localStore";

    export default {
        data() {
            return {
                showInputablePassword: false,
                show: false,
                isShowAlert: false,
                loginError: "",

                disable: false,

                loadingLogin: false,
                loadingVk: false,
                loadingFb: false,
                loadingGoogle: false,
                loadingKeycloak: false,

                login: '',
                loginRules: [
                    v => !!v || 'Login is required',
                ],
                password: '',
                passwordRules: [
                    v => !!v || 'Password is required',
                ],

            }
        },
        mounted() {
            bus.on(LOGGED_OUT, this.showLoginModal);
        },
        beforeUnmount() {
            bus.off(LOGGED_OUT, this.showLoginModal);
        },
        computed: {
            ...mapStores(useChatStore),
        },
        methods: {
            registration() {
              return registration
            },
            forgot_password() {
              return forgot_password
            },
            showLoginModal() {
                this.$data.show = true;
            },
            hideLoginModal() {
                this.$data.show = false;
            },

            loginVk() {
                this.loadingVk = true;
                this.disable = true;
                window.location.href = '/api/aaa/login/oauth2/vkontakte';
            },
            loginFb() {
                this.loadingFb = true;
                this.disable = true;
                window.location.href = '/api/aaa/login/oauth2/facebook';
            },
            loginGoogle() {
                this.loadingGoogle = true;
                this.disable = true;
                window.location.href = '/api/aaa/login/oauth2/google';
            },
            loginKeycloak() {
                this.loadingKeycloak = true;
                this.disable = true;
                window.location.href = '/api/aaa/login/oauth2/keycloak';
            },
            validate () {
                return this.$refs.form.validate()
            },
            reset () {
                this.$refs.form.reset()
            },
            resetValidation () {
                this.$refs.form.resetValidation()
            },
            loginWithLogin() {
                this.disable = true;
                this.loadingLogin = true;
                const valid = this.validate();
                console.log("Valid", valid);
                if (valid) {
                    this.hideAlert();
                    const dto = {
                        login: this.$data.login,
                        password: this.$data.password
                    };
                    const params = new URLSearchParams();
                    Object.keys(dto).forEach((key) => {
                        params.append(key, dto[key])
                    });

                    axios.post(`/api/aaa/login`, params)
                        .then((value) => {
                            // store.dispatch(replayPreviousUrl());
                            console.log("You successfully logged in");
                            this.hideLoginModal();
                            this.chatStore.fetchUserProfile().then(()=>{
                                bus.emit(LOGGED_IN, null);
                                setLanguageToVuetify(this, getStoredLanguage());
                            })
                        })
                        .catch((error) => {
                            // handle error
                            console.log("Handling error on login", error.response);
                            this.$data.isShowAlert = true;
                            this.$data.loginError = tryExtractMeaningfulError(error)
                        }).finally(() => {
                            this.loadingLogin = false;
                            this.disable = false;
                        });
                } else {
                    this.loadingLogin = false;
                    this.disable = false;
                }
            },
            onRegisterClick() {
              this.show = false;
              this.$router.push({name: registration_name, query: { referer: window.location.href }} )
            },
            onForgotPasswordClick() {
              this.show = false;
              this.$router.push({name: forgot_password_name} )
            },
            hideAlert() {
                this.$data.isShowAlert = false;
                this.$data.loginError = "";
            },
            onLanguageClick() {
                bus.emit(OPEN_SETTINGS)
            },
            goBlog() {
              window.location.href = blog
            },
            getBlog() {
              return blog
            },
        }
    }
</script>

<style lang="stylus" scoped>
    @import "oAuth2.styl"
</style>
