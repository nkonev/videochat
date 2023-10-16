<template>
  <div>video here</div>
</template>

<script>
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";

export default {
  computed: {
    ...mapStores(useChatStore),
  },
  mounted() {
    this.chatStore.showCallButton = false;
    this.chatStore.showHangButton = true;
  },
  beforeUnmount() {
    this.chatStore.showCallButton = true;
    this.chatStore.showHangButton = false;
  }
}
</script>
