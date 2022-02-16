<template>
  <tiptap :value="text" />
</template>

<script>
  import Tiptap from "@/TipTapEditor";

  export default {
    data() {
      return {
        text: "a texto"
      }
    },
    components: {
      Tiptap
    }
  }
</script>