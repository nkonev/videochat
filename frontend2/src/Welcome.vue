<template>
    <div>Welcome!</div>
</template>

<script>
  import {setTitle} from "@/utils";
  import {mapStores} from "pinia";
  import {useChatStore} from "@/store/chatStore";

  export default {
    computed: {
      ...mapStores(useChatStore),
    },

    mounted() {
      this.chatStore.title = this.$vuetify.locale.t('$vuetify.welcome');
      setTitle(this.$vuetify.locale.t('$vuetify.welcome'));
    },
    beforeUnmount() {
      setTitle(null);
      this.chatStore.title = null;
    }
  }
</script>
