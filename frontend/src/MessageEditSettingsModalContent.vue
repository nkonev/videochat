<template>

  <v-card-text class="pb-0">

      <v-switch
          :label="$vuetify.locale.t('$vuetify.normalize_pasted_text')"
          density="comfortable"
          color="primary"
          hide-details
          class="ma-0 pt-0 ml-4 mr-4 mb-2"
          v-model="normalizeText"
          @update:modelValue="changeNormalizeText"
      ></v-switch>

      <v-divider/>

      <v-radio-group class="mt-4"
                     v-model="sendButtonsType"
                     @update:modelValue="changeSendButtonsType"
                     color="primary"
                     hide-details
      >
          <template v-slot:label>
              <div>{{ $vuetify.locale.t('$vuetify.message_send_buttons_type') }}</div>
          </template>
          <v-radio :label="$vuetify.locale.t('$vuetify.message_send_buttons_type_auto')" value="auto"></v-radio>
          <v-radio :label="$vuetify.locale.t('$vuetify.message_send_buttons_type_full')" value="full"></v-radio>
          <v-radio :label="$vuetify.locale.t('$vuetify.message_send_buttons_type_compact')" value="compact"></v-radio>
      </v-radio-group>

  </v-card-text>

</template>

<script>
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import {
        getStoredMessageEditNormalizeText,
        getStoredMessageEditSendButtonsType,
        setStoredMessageEditNormalizeText, setStoredMessageEditSendButtonsType
    } from "@/store/localStore.js";
    import bus, {ON_MESSAGE_EDIT_SEND_BUTTONS_TYPE_CHANGED} from "@/bus/bus.js";

    export default {
        data () {
            return {
                normalizeText: null,
                sendButtonsType: null,
            }
        },
        computed: {
            ...mapStores(useChatStore),
        },
        methods: {
            showModal() {
                this.normalizeText = getStoredMessageEditNormalizeText();
                this.sendButtonsType = getStoredMessageEditSendButtonsType('auto');
            },
            changeNormalizeText(v) {
                setStoredMessageEditNormalizeText(v);
            },
            changeSendButtonsType(v) {
                setStoredMessageEditSendButtonsType(v);
                bus.emit(ON_MESSAGE_EDIT_SEND_BUTTONS_TYPE_CHANGED)
            },
        },
        mounted() {
            this.showModal()
        }
    }
</script>
