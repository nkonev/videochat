<template>
  <v-dialog
    v-model="show"
    width="auto"
  >
    <v-card>

        <v-sheet elevation="6">
          <v-tabs
            v-model="tab"
            bg-color="indigo"
            show-arrows
          >
            <v-tab value="choose_language">
              {{ $vuetify.locale.t('$vuetify.language') }}
            </v-tab>
            <v-tab value="a_video_settings">
              {{ $vuetify.locale.t('$vuetify.video_settings') }}
            </v-tab>
          </v-tabs>
        </v-sheet>

      <v-card-text>
        <v-window v-model="tab">
          <v-window-item value="choose_language">
            <LanguageModal/>
          </v-window-item>
          <v-window-item value="a_video_settings">
            Video settings will be here
          </v-window-item>
        </v-window>
      </v-card-text>

      <v-card-actions>
        <v-spacer/>
        <v-btn color="red" variant="flat" @click="hideLoginModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
import bus, { OPEN_SETTINGS} from "@/bus/bus";
import LanguageModal from "@/LanguageModal.vue";

export default {
  data () {
    return {
      tab: null,
      show: false,
    }
  },
  components: {
    LanguageModal
  },
  methods: {
    showLoginModal() {
      this.$data.show = true;
    },
    hideLoginModal() {
      this.$data.show = false;
    },
  },
  created() {
    bus.on(OPEN_SETTINGS, this.showLoginModal);
  },
  destroyed() {
    bus.off(OPEN_SETTINGS, this.showLoginModal);
  },

}
</script>
