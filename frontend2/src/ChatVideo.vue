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
    this.chatStore.showDrawer = false;
    this.chatStore.showCallButton = false;
    this.chatStore.showHangButton = true;
  },
  beforeUnmount() {
    this.chatStore.showCallButton = true;
    this.chatStore.showHangButton = false;
    this.chatStore.showDrawer = true;
  }
}
</script>
