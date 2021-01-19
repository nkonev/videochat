<template>
    <v-card v-if="currentUser"
            class="mr-auto"
            max-width="640"
    >
        <v-list-item three-line>
            <v-list-item-content class="d-flex justify-space-around">
                <div class="overline mb-4">User profile #{{ currentUser.id }}</div>
                <v-img v-if="currentUser.avatar"
                       :src="ava"
                       :aspect-ratio="16/9"
                       min-width="200"
                       min-height="200"
                       @click="openAvatarDialog"
                >
                </v-img>
                <v-btn v-else color="primary" @click="chooseAvatar()">Choose avatar</v-btn>
                <v-list-item-title class="headline mb-1 mt-2">{{ currentUser.login }}</v-list-item-title>
                <v-list-item-subtitle v-if="currentUser.email">{{ currentUser.email }}</v-list-item-subtitle>
            </v-list-item-content>
        </v-list-item>

        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">Bound OAuth2 providers</v-card-title>
        <v-card-actions class="mx-2">
            <v-chip
                v-if="currentUser.oauth2Identifiers.vkontakteId"
                min-width="80px"
                label
                close
                class="c-btn-vk py-5 mr-2"
                text-color="white"
                close-icon="mdi-delete"
                @click:close="removeVk"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="currentUser.oauth2Identifiers.facebookId"
                min-width="80px"
                label
                close
                class="c-btn-fb py-5 mr-2"
                text-color="white"
                close-icon="mdi-delete"
                @click:close="removeFb"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="currentUser.oauth2Identifiers.googleId"
                min-width="80px"
                label
                close
                class="c-btn-google py-5 mr-2"
                text-color="white"
                close-icon="mdi-delete"
                @click:close="removeGoogle"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google'}" :size="'2x'"></font-awesome-icon>
            </v-chip>
        </v-card-actions>

        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">Not bound OAuth2 providers</v-card-title>
        <v-card-actions class="mx-2">
            <v-chip
                v-if="!currentUser.oauth2Identifiers.vkontakteId"
                @click="submitOauthVkontakte"
                min-width="80px"
                label
                class="c-btn-vk py-5 mr-2"
                text-color="white"
                close-icon="mdi-delete"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="!currentUser.oauth2Identifiers.facebookId"
                @click="submitOauthFacebook"
                min-width="80px"
                label
                class="c-btn-fb py-5 mr-2"
                text-color="white"
                close-icon="mdi-delete"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="!currentUser.oauth2Identifiers.googleId"
                @click="submitOauthGoogle"
                min-width="80px"
                label
                class="c-btn-google py-5 mr-2"
                text-color="white"
                close-icon="mdi-delete"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google'}" :size="'2x'"></font-awesome-icon>
            </v-chip>
        </v-card-actions>


        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">Login</v-card-title>
        <v-btn v-if="!showLoginInput" class="mx-4 mb-4" color="primary" dark @click="showLoginInput = !showLoginInput; loginPrevious = currentUser.login">
            Change login
            <v-icon dark right>mdi-account</v-icon>
        </v-btn>
        <v-row v-if="showLoginInput" no-gutters>
            <v-col cols="12" >
                <v-row :align="'center'" no-gutters>
                    <v-col class="ml-4">
                        <v-text-field
                            v-model="currentUser.login"
                            :rules="[rules.required]"
                            label="Login"
                            @keyup.native.enter="sendLogin()"
                        ></v-text-field>
                    </v-col>
                    <v-col md="auto" class="ml-1 mr-4">
                        <v-row :align="'center'" no-gutters>
                            <v-icon @click="sendLogin()" color="primary">mdi-check-bold</v-icon>
                            <v-icon @click="showLoginInput = false; currentUser.login = loginPrevious">mdi-cancel</v-icon>
                        </v-row>
                    </v-col>
                </v-row>
            </v-col>
        </v-row>


        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">Password</v-card-title>
        <v-btn v-if="!showPasswordInput" class="mx-4 mb-4" color="primary" dark
               @click="showPasswordInput = !showPasswordInput">Change password
            <v-icon dark right>mdi-lock</v-icon>
        </v-btn>
        <v-row v-if="showPasswordInput" no-gutters>
            <v-col cols="12" >
                <v-row :align="'center'" no-gutters>
                    <v-col class="ml-4">
                        <v-text-field
                            v-model="password"
                            :append-icon="showInputablePassword ? 'mdi-eye' : 'mdi-eye-off'"
                            @click:append="showInputablePassword = !showInputablePassword"
                            :type="showInputablePassword ? 'text' : 'password'"
                            :rules="[rules.required, rules.min]"
                            label="Password"
                            hint="At least 8 characters"
                            @keyup.native.enter="sendPassword()"
                        ></v-text-field>
                    </v-col>
                    <v-col md="auto" class="ml-1 mr-4">
                        <v-row :align="'center'" no-gutters>
                            <v-icon @click="sendPassword()" color="primary">mdi-check-bold</v-icon>
                            <v-icon @click="showPasswordInput = false">mdi-cancel</v-icon>
                        </v-row>
                    </v-col>
                </v-row>
            </v-col>
        </v-row>


        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">Email</v-card-title>
        <v-btn v-if="!showEmailInput" class="mx-4 mb-4" color="primary" dark @click="showEmailInput = !showEmailInput; emailPrevious = currentUser.email">
            Change email
            <v-icon dark right>mdi-email</v-icon>
        </v-btn>
        <v-row v-if="showEmailInput" no-gutters>
            <v-col cols="12" >
                <v-row :align="'center'" no-gutters>
                    <v-col class="ml-4">
                        <v-text-field
                            v-model="currentUser.email"
                            :rules="[rules.required, rules.email]"
                            label="E-mail"
                            @keyup.native.enter="sendEmail()"
                        ></v-text-field>
                    </v-col>
                    <v-col md="auto" class="ml-1 mr-4">
                        <v-row :align="'center'" no-gutters>
                            <v-icon @click="sendEmail()" color="primary">mdi-check-bold</v-icon>
                            <v-icon @click="showEmailInput = false; currentUser.email = emailPrevious">mdi-cancel</v-icon>
                        </v-row>
                    </v-col>
                </v-row>
            </v-col>
        </v-row>

    </v-card>
    <v-alert type="warning" v-else>
        You are not logged in
    </v-alert>
</template>

<script>
import {mapGetters} from "vuex";
import {FETCH_USER_PROFILE, GET_USER} from "./store";
import axios from "axios";
import bus, {CHANGE_TITLE, OPEN_CHOOSE_AVATAR} from "./bus";
import {titleFactory} from "./changeTitle";
import {getCorrectUserAvatar} from "./utils";

export default {
    data() {
        return {
            showInputablePassword: false,
            rules: {
                required: value => !!value || 'Required.',
                min: v => v.length >= 8 || 'Min 8 characters',
                email: value => {
                    const pattern = /^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/
                    return pattern.test(value) || 'Invalid e-mail.'
                },
            },

            showLoginInput: false,
            showPasswordInput: false,
            showEmailInput: false,

            loginPrevious: "",
            password: "",
            emailPrevious: ""
        }
    },
    computed: {
        ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        ava() {
            const maybeUser = this.$store.getters[GET_USER];
            if (maybeUser) {
                return getCorrectUserAvatar(maybeUser.avatar);
            }
        }
    },
    methods: {
        submitOauthVkontakte() {
            window.location.href = '/api/login/oauth2/vkontakte';
        },
        submitOauthFacebook() {
            window.location.href = '/api/login/oauth2/facebook';
        },
        submitOauthGoogle() {
            window.location.href = '/api/login/oauth2/google';
        },

        sendLogin() {
            axios.patch('/api/profile', {login: this.currentUser.login})
                .then((response) => {
                    this.$store.dispatch(FETCH_USER_PROFILE);
                    this.showLoginInput = false;
                })
        },
        sendPassword() {
            axios.patch('/api/profile', {password: this.password})
                .then((response) => {
                    this.showPasswordInput = false;
                })
        },
        sendEmail() {
            axios.patch('/api/profile', {email: this.currentUser.email})
                .then((response) => {
                    this.$store.dispatch(FETCH_USER_PROFILE);
                    this.showEmailInput = false;
                })
        },
        removeVk() {
            axios.delete('/api/profile/vkontakte')
                .then((response) => {
                    this.$store.dispatch(FETCH_USER_PROFILE);
                })
        },
        removeFb() {
            axios.delete('/api/profile/facebook')
                .then((response) => {
                    this.$store.dispatch(FETCH_USER_PROFILE);
                })
        },
        removeGoogle() {
            axios.delete('/api/profile/google')
                .then((response) => {
                    this.$store.dispatch(FETCH_USER_PROFILE);
                })
        },


        openAvatarDialog() {
            bus.$emit(OPEN_CHOOSE_AVATAR);
        },
        // getAvatar() {
        //     return getCorrectUserAvatar(this.currentUser.avatar)
        // },
        chooseAvatar() {
            bus.$emit(OPEN_CHOOSE_AVATAR);
        }
    },
    mounted() {
        bus.$emit(CHANGE_TITLE, titleFactory(`User profile`, false, false, null, null));
    },
}
</script>

<style lang="stylus">
@import "OAuth2.styl"
</style>