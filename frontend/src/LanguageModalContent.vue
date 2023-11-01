<template>
  <v-card-text class="pb-0 d-flex justify-center">
    <v-btn-toggle
      v-model="language"
      @update:modelValue="changeLanguage"
    >
      <v-btn value="ru">
        Русский
      </v-btn>

      <v-btn value="en">
        English
      </v-btn>

    </v-btn-toggle>
  </v-card-text>

</template>

<script>
    import {getStoredLanguage, setStoredLanguage} from "@/store/localStore";

    export default {
        data () {
            return {
                language: null,
            }
        },
        methods: {
            init() {
                this.language = getStoredLanguage();
                this.setToVuetify(this.language);
            },
            setToVuetify(newLanguage) {
                this.$vuetify.locale.current = newLanguage;
            },

            changeLanguage(newLanguage) {
                console.log("Lang has been changed")
                setStoredLanguage(newLanguage);
                this.setToVuetify(newLanguage);
            },
        },
        mounted() {
          this.init()
        }
    }
</script>
