<template>
    <!--
    https://vuetifyjs.com/en/components/dialogs/#dialogs
    https://vuetifyjs.com/en/components/forms/
    -->
    <v-row justify="center">
        <v-dialog persistent v-model="show" max-width="440">
            <v-card>
                <v-card-title class="headline">Login</v-card-title>

                <v-card-text>
                    <v-form
                        ref="form"
                        v-model="valid"
                        lazy-validation
                        @keyup.native.enter="loginWithUsername"
                    >
                        <v-text-field
                                v-model="username"
                                :rules="usernameRules"
                                label="Login"
                                required
                                :disabled="disable"
                                @input="hideAlert()"
                        ></v-text-field>

                        <v-text-field
                                v-model="password"
                                :rules="passwordRules"
                                label="Password"
                                required
                                type="password"
                                :disabled="disable"
                                @input="hideAlert()"
                        ></v-text-field>

                        <v-alert
                                dismissible
                                v-model="showAlert"
                                type="error"
                        >
                            <v-row align="center">
                                <v-col class="grow">{{loginError}}</v-col>
                            </v-row>
                        </v-alert>

                        <v-btn
                                :disabled="!valid || disable"
                                color="success"
                                class="mr-2"
                                @click="loginWithUsername"
                                min-width="80px"
                                :loading="loadingLogin"
                        >
                            Login
                        </v-btn>
                        <v-btn class="mr-2 c-btn-vk" :disabled="disable" :loading="loadingVk" min-width="80px" @click="loginVk()"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon></v-btn>
                        <v-btn class="mr-2 c-btn-fb" :disabled="disable" :loading="loadingFb" min-width="80px" @click="loginFb()"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook' }" :size="'2x'"></font-awesome-icon></v-btn>
                        <v-btn class="mr-2 c-btn-google" :disabled="disable" :loading="loadingGoogle" min-width="80px" @click="loginGoogle()"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google' }" :size="'2x'"></font-awesome-icon></v-btn>
                    </v-form>
                </v-card-text>

            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {LOGGED_IN, UNAUTHORIZED} from "./bus";
    import axios from "axios";
    import {FETCH_USER_PROFILE} from "./store";

    export default {
        data() {
            return {
                show: false,
                showAlert: false,
                loginError: "",

                disable: false,

                loadingLogin: false,
                loadingVk: false,
                loadingFb: false,
                loadingGoogle: false,

                valid: true,
                username: '',
                usernameRules: [
                    v => !!v || 'Login is required',
                ],
                password: '',
                passwordRules: [
                    v => !!v || 'Password is required',
                ],

            }
        },
        created() {
            bus.$on(UNAUTHORIZED, this.showLoginModal);
        },
        destroyed() {
            bus.$off(UNAUTHORIZED, this.showLoginModal);
        },
        methods: {
            showLoginModal() {
                this.$data.show = true;
            },
            hideLoginModal() {
                this.$data.show = false;
            },

            loginVk() {
                this.loadingVk = true;
                this.disable = true;
                window.location.href = '/api/login/oauth2/vkontakte';
            },
            loginFb() {
                this.loadingFb = true;
                this.disable = true;
                window.location.href = '/api/login/oauth2/facebook';
            },
            loginGoogle() {
                this.loadingGoogle = true;
                this.disable = true;
                window.location.href = '/api/login/oauth2/google';
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
            loginWithUsername() {
                this.disable = true;
                this.loadingLogin = true;
                const valid = this.validate();
                console.log("Valid", valid);
                if (valid) {
                    const dto = {
                        username: this.$data.username,
                        password: this.$data.password
                    };
                    const params = new URLSearchParams();
                    Object.keys(dto).forEach((key) => {
                        params.append(key, dto[key])
                    });

                    axios.post(`/api/login`, params)
                        .then((value) => {
                            // store.dispatch(replayPreviousUrl());
                            console.log("You successfully logged in");
                            this.hideLoginModal();
                            this.$store.dispatch(FETCH_USER_PROFILE);
                            bus.$emit(LOGGED_IN, null);
                        })
                        .catch((error) => {
                            // handle error
                            console.log("Handling error on login", error.response);
                            this.$data.showAlert = true;
                            if (error.response.status === 401) {
                                this.$data.loginError = "Wrong login or password";
                            } else {
                                this.$data.loginError = "Unknown error " + error.response.status;
                            }
                        }).finally(() => {
                            this.loadingLogin = false;
                            this.disable = false;
                        });
                } else {
                    this.loadingLogin = false;
                    this.disable = false;
                }
            },
            hideAlert() {
                this.$data.showAlert = false;
                this.$data.loginError = "";
            }
        }
    }
</script>

<style lang="stylus">
    @import "OAuth2.styl"
</style>