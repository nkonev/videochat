import {deepCopy, hasLength} from "@/utils";

export const SEARCH_MODE_CHATS = "qc"
export const SEARCH_MODE_MESSAGES = "qm"

export const goToPreserving = (route, router, to) => {
    const prev = deepCopy(route.query);
    router.push({ ...to, query: prev })
}

export default (name) => {
  return {
    computed: {
      searchString: {
          get() {
              return this.$route.query[name];
          },
          set(newVal) {
              const prev = deepCopy(this.$route.query);

              let newQuery;
              if (hasLength(newVal)) {
                  prev[name] = newVal;
              } else {
                  delete prev[name]
              }
              newQuery = prev;

              this.$router.push({ query: newQuery })
          }
      },
    }
  }
}
