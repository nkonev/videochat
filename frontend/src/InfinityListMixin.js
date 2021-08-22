import InfiniteLoading from 'vue-infinite-loading';

export const pageSize = 40;

export default () => {
    return  {
        data () {
            return {
                page: 0,
                startingFromItemId: null,
                items: [],
                itemsTotal: 0,
                infiniteId: +new Date(),
            }
        },
        components:{
            InfiniteLoading,
        },
        methods:{
            // not working until you will change this.items list
            reloadItems() {
                this.infiniteId += 1;
                console.log("Resetting infinite loader", this.infiniteId);
            },

            searchStringChanged() {
                this.items = [];
                this.page = 0;
                this.startingFromItemId = null;
                this.reloadItems();
            },
        },

    }
}