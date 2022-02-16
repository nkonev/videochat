<template>
  <tiptap v-model="content" />
</template>

<script>
  import Tiptap from "@/TipTapEditor";

  export default {
    data() {
      return {
        content: "a texto"
      }
    },
    components: {
      Tiptap
    },
    watch: {
      content(newValue) {
        console.log("New content: ", newValue);
      }
    }
  }
</script>