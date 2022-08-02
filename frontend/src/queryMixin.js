import {GET_SEARCH_STRING, SET_SEARCH_STRING} from "@/store";
import {videochat_name} from "@/routes";

export const searchQueryParameter = 'q';


// it expects searchStringChanged() method in receivee
export default () => {
    let unsubscribe;

    return {
        methods: {
            initQueryAndWatcher() {
                // Initialize store from query
                const gotQuery = this.$route.query[searchQueryParameter];
                console.debug("gotQuery", gotQuery);
                this.searchString = gotQuery ? gotQuery : "";
                console.debug("this.searchString", this.searchString);

                // set watcher on store change - trigger server request
                unsubscribe = this.$store.subscribe((mutation, state) => {
                    // console.debug("mutation.type", mutation.type);
                    // console.debug("mutation.payload", mutation.payload);
                    if (mutation.type == SET_SEARCH_STRING) {
                        this.searchStringChanged(mutation.payload);
                    }
                });
            },
            closeQueryWatcher() {
                unsubscribe();
            },
            navigateToWithPreservingSearchStringInQuery(routerNewState) {
                if (this.searchString && this.searchString != "") {
                    routerNewState.query = {[searchQueryParameter]: this.searchString};
                }
                this.$router.push(routerNewState);
            }
        },
        computed: {
            searchString: {
                get(){
                    return this.$store.getters[GET_SEARCH_STRING];
                },
                set(newVal){
                    this.$store.commit(SET_SEARCH_STRING, newVal);
                    return newVal;
                }
            }
        }
    }
}