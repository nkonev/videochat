<template>
    <v-sheet max-width="640" class="px-2 pt-2">
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

            <v-btn type="submit" color="primary" block class="mt-2">{{ $vuetify.locale.t('$vuetify.request_resend_confirmation_email') }}</v-btn>
        </v-form>
    </v-sheet>
</template>

<script>
import userProfileValidationRules from "@/mixins/userProfileValidationRules";
import {hasLength, setTitle} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import axios from "axios";
import {check_email_name} from "@/router/routes";

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
            axios.post("/api/aaa/resend-confirmation-email", null, { params: {
                    email: this.email,
                    language: this.$vuetify.locale.current
                }})
                .then(() => {
                    this.$router.push({name: check_email_name} )
                })
                .catch(e => {
                    this.error = e.message
                })
        },
        hideAlert() {
            this.error = "";
        },
        setTopTitle() {
            this.chatStore.title = this.$vuetify.locale.t('$vuetify.resending_confirmation_email');
            setTitle(this.$vuetify.locale.t('$vuetify.resending_confirmation_email'));
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
        this.error = "";
    }
}
</script>
