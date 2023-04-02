export const searchQueryParameter = 'q';

export const querySetter = (router, newVal) => {
    let currentRoute = router.currentRoute;
    let newRoute = {
        name: currentRoute.name
    }
    if (newVal) {
        newRoute.query = {[searchQueryParameter]: newVal }
    }
    router.push(newRoute);
}

// it expects searchStringChanged() method in receivee
export default () => {

    return {
        methods: {
            navigateToWithPreservingSearchStringInQuery(routerNewState) {
                if (this.searchString && this.searchString != "") {
                    routerNewState.query = {[searchQueryParameter]: this.searchString};
                }
                this.$router.push(routerNewState);
            },
            getSearchString() {
                return this.$router.currentRoute.query?.[searchQueryParameter];
            },
            setSearchString(newVal) {
                querySetter(this.$router, newVal)
            },
        },
        // TODO planned for removal, removed from only App.vue
        computed: {
            searchString: {
                get(){
                    return this.$router.currentRoute.query?.[searchQueryParameter];
                },
                set(newVal){
                    querySetter(this.$router, newVal)
                }
            }
        },
        watch: {
            '$route': {
                handler: function(newRoute, oldRoute) {
                    console.debug("Watched on newRoute in queryMixin", newRoute, " oldRoute", oldRoute);
                    let value = this.$router.currentRoute.query?.[searchQueryParameter];
                    if (value) {
                        this.searchStringChanged(value);
                    }
                },
                immediate: true,
                deep: true
            },
        },

    }
}
