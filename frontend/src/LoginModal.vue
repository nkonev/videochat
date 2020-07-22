<template>
    <!--
    https://vuetifyjs.com/en/components/dialogs/#dialogs
    https://vuetifyjs.com/en/components/forms/
    -->
    <v-row justify="center">
        <v-dialog persistent v-model="show" max-width="400">
            <v-card>
                <v-card-title class="headline">Login</v-card-title>

                <v-card-text>
                    <v-form
                        ref="form"
                        v-model="valid"
                        lazy-validation
                        @keyup.native.enter="validateAndSend"
                    >
                        <v-text-field
                                v-model="username"
                                :rules="usernameRules"
                                label="Login"
                                required
                                @input="hideAlert()"
                        ></v-text-field>

                        <v-text-field
                                v-model="password"
                                :rules="passwordRules"
                                label="Password"
                                required
                                type="password"
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
                                :disabled="!valid"
                                color="success"
                                class="mr-4"
                                @click="validateAndSend"
                        >
                            Login
                        </v-btn>
                        <v-btn class="mr-4 c-btn-vk"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'" @click="loginVk()"></font-awesome-icon></v-btn>
                        <v-btn class="mr-4 c-btn-fb"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook' }" :size="'2x'" @click="loginFb()"></font-awesome-icon></v-btn>
                    </v-form>
                </v-card-text>

            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {UNAUTHORIZED} from "./bus";
    import axios from "axios";
    import {FETCH_USER_PROFILE} from "./store";

    export default {
        data() {
            return {
                show: false,
                showAlert: false,
                loginError: "",

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
                window.location.href = '/api/login/oauth2/vkontakte';
            },
            loginFb() {
                window.location.href = '/api/login/oauth2/facebook';
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
            validateAndSend() {
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
                        });
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
    $fbColor=#3B5998
    $vkColor=#45668e

    .c-btn-fb {
        border-color: $fbColor !important
        background: $fbColor !important
        color: white !important
    }
    .c-btn-vk {
        border-color: $vkColor !important
        background: $vkColor !important
        color: white !important
    }
</style>