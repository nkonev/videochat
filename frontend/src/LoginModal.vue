<template>
    <!--
    https://vuetifyjs.com/en/components/dialogs/#dialogs
    https://vuetifyjs.com/en/components/forms/
    -->
    <v-row justify="center">
        <v-dialog persistent v-model="show" max-width="300">
            <v-card>
                <v-card-title class="headline">Login</v-card-title>

                <v-card-text>
                    <v-form
                        ref="form"
                        v-model="valid"
                        lazy-validation
                    >
                        <v-text-field
                                v-model="username"
                                :rules="usernameRules"
                                label="Login"
                                required
                        ></v-text-field>

                        <v-text-field
                                v-model="password"
                                :rules="passwordRules"
                                label="Password"
                                required
                                type="password"
                        ></v-text-field>

                        <v-btn
                                :disabled="!valid"
                                color="success"
                                class="mr-4"
                                @click="validateAndSend"
                        >
                            Login
                        </v-btn>

                </v-form>
                </v-card-text>

            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {UNAUTHORIZED} from "./bus";
    import axios from "axios";

    export default {
        data() {
            return {
                show: false,

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
                        })
                        .catch((error) => {
                            // handle error
                            console.log("Handling error on login", error.response);
                        });
                }
            }
        }
    }
</script>