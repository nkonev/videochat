import {hasLength} from "@/utils";

export default () => {
  return {
    computed: {
      searchString: {
        get() {
          return this.$route.query.q;
        },
        set(newVal) {
          const newQuery = hasLength(newVal) ?  { q: newVal } : {}
          this.$router.push({ query: newQuery })
        }
      },

    }
  }
}
