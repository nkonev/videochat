<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="320">
            <v-card v-if="show">
                <v-card-title>Language</v-card-title>

                <v-card-text class="px-4 py-0">

                    <v-select
                        messages="Language"
                        :items="languageItems"
                        label="Language"
                        dense
                        solo
                        @change="changeLanguage"
                        v-model="language"
                    ></v-select>
                  
                </v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn color="error" class="mr-4" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                    <v-spacer/>
                </v-card-actions>

            </v-card>

        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {
      OPEN_LANGUAGE_MODAL,
    } from "./bus";
    import {getStoredLanguage, setStoredLanguage} from "@/utils";

    export default {
        data () {
            return {
                language: null,
                show: false,
            }
        },
        methods: {
            showModal() {
                this.show = true;
                this.language = getStoredLanguage();
                this.setToVuetify(this.language);
            },
            closeModal() {
                this.show = false;
            },
            setToVuetify(newLanguage) {
                this.$vuetify.lang.current = newLanguage;
            },

            changeLanguage(newLanguage) {
                console.log("Setting lang", newLanguage);
                setStoredLanguage(newLanguage);
                this.setToVuetify(newLanguage);
            },
        },
        computed: {
            languageItems() {
                return [{text:'USA', value: 'en'}, {text:'Russia', value: 'ru'}]
            },
        },
        created() {
            bus.$on(OPEN_LANGUAGE_MODAL, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_LANGUAGE_MODAL, this.showModal);
        },
    }
</script>