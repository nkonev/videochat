<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="320">
            <v-card v-if="show">
                <v-card-title>{{ $vuetify.lang.t('$vuetify.language') }}</v-card-title>

                <v-card-text class="px-4 py-0">

                    <v-btn-toggle
                        v-model="language"
                        tile
                        color="primary accent-3"
                        group
                        mandatory
                        @change="changeLanguage"
                    >
                        <v-btn value="ru">
                          Русский
                        </v-btn>

                        <v-btn value="en">
                          English
                        </v-btn>

                    </v-btn-toggle>
                  
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
        created() {
            bus.$on(OPEN_LANGUAGE_MODAL, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_LANGUAGE_MODAL, this.showModal);
        },
    }
</script>