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
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import axios from "axios";
    import {setLanguageToVuetify} from "@/utils";

    export default {
        data () {
            return {
                language: null,
            }
        },
        computed: {
            ...mapStores(useChatStore),
        },
        methods: {
            init() {
                this.language = getStoredLanguage();
            },

            async changeLanguage(newLanguage) {
                console.log("Changing language to", newLanguage)
                if (this.chatStore.currentUser != null) {
                    await axios.put("/api/aaa/settings/language", {language: newLanguage})
                }
                setStoredLanguage(newLanguage);
                setLanguageToVuetify(this, newLanguage);
            },
        },
        mounted() {
          this.init()
        }
    }
</script>
