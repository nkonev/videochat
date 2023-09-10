import {deepCopy, hasLength} from "@/utils";
import bus, {SEARCH_STRING_CHANGED} from "@/bus/bus";

export const SEARCH_MODE_CHATS = "qc"
export const SEARCH_MODE_MESSAGES = "qm"

export const goToPreserving = (route, router, to) => {
    const prev = deepCopy(route.query);
    router.push({ ...to, query: prev })
}

export const searchStringFacade = () => {
    return {
        computed: {
            searchStringFacade: {
                get() {
                    return this.$route.query[this.chatStore.searchType];
                },
                set(newVal) {
                    const prev = deepCopy(this.$route.query);

                    let newQuery;
                    if (hasLength(newVal)) {
                        prev[this.chatStore.searchType] = newVal;
                    } else {
                        delete prev[this.chatStore.searchType]
                    }
                    newQuery = prev;

                    this.$router.push({query: newQuery})
                }

            }
        },
        watch: {
            ['$route.query.'+SEARCH_MODE_CHATS]: {
                handler: function (newValue, oldValue) {
                    console.debug("Route changed from q", SEARCH_MODE_CHATS, oldValue, "->", newValue);
                    bus.emit(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_CHATS, {oldValue: oldValue, newValue: newValue});
                }
                ,
            },
            ['$route.query.'+SEARCH_MODE_MESSAGES]: {
                handler: function (newValue, oldValue) {
                    console.debug("Route changed from q", SEARCH_MODE_MESSAGES, oldValue, "->", newValue);
                    bus.emit(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_MESSAGES, {oldValue: oldValue, newValue: newValue});
                }
                ,
            },
        }
    }
}

export const searchString = (name) => {
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
