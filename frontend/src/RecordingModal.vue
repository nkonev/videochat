<template>

  <v-dialog
    v-model="show"
    scrollable
    persistent
    width="fit-content" max-width="100%"
  >
      <v-card>
          <v-sheet elevation="6">
              <v-tabs
                  v-model="tab"
                  bg-color="indigo"
                  @update:modelValue="onUpdateTab"
              >
                  <v-tab value="video">{{$vuetify.locale.t('$vuetify.video')}}</v-tab>
                  <v-tab value="audio">{{$vuetify.locale.t('$vuetify.audio')}}</v-tab>
              </v-tabs>
          </v-sheet>

          <v-window v-model="tab" style="overflow: auto">
              <v-window-item value="video">
                  <RecordingVideoModal ref="recordingVideoModalRef" @closemodal="onCloseModal"/>
              </v-window-item>
              <v-window-item value="audio">
                  <RecordingAudioModal ref="recordingAudioModalRef"/>
              </v-window-item>
          </v-window>
      </v-card>
  </v-dialog>
</template>

<script>
import bus, {OPEN_RECORDING_MODAL} from "@/bus/bus";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import RecordingVideoModal from "@/RecordingVideoModal.vue";
import RecordingAudioModal from "@/RecordingAudioModal.vue";

export default {
  data () {
    return {
      tab: null,
      show: false,
    }
  },
  components: {
      RecordingVideoModal,
      RecordingAudioModal,
  },
  methods: {
    showModal() {
        this.$data.show = true;
    },
    onUpdateTab(tab) {
        console.debug("Setting tab", tab);
    },
    onCloseModal() {
      this.$data.show = false;
      this.tab = null;
    },
  },
  computed: {
    ...mapStores(useChatStore),
  },
  beforeUnmount() {
    bus.off(OPEN_RECORDING_MODAL, this.showModal);
  },
  mounted() {
    bus.on(OPEN_RECORDING_MODAL, this.showModal);
  }
}
</script>
