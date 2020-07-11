import axios from 'axios';
import InfiniteLoading from 'vue-infinite-loading';

const replaceInArray = (array, element) => {
    const foundIndex = array.findIndex(value => value.id === element.id);
    if (foundIndex === -1) {
        return false;
    } else {
        array[foundIndex] = element;
        return true;
    }
};

const replaceOrAppend = (array, newArray) => {
    newArray.forEach((element, index) => {
        const replaced = replaceInArray(array, element);
        if (!replaced) {
            array.push(element);
        }
    });
};

const pageSize = 20;

export default (urlFunction) => {
    return  {
        data () {
            return {
                page: 0,
                lastPageActualSize: 0,
                items: [],
                infiniteId: new Date(),
                searchString: ""
            }
        },
        components:{
            InfiniteLoading,
        },
        methods:{
            infiniteHandler($state) {
                axios.get(urlFunction(), {
                    params: {
                        page: this.page,
                        size: pageSize,
                        searchString: this.searchString
                    },
                }).then(({ data }) => {
                    if (data.length) {
                        this.page += 1;
                        //this.chats.push(...data);
                        replaceOrAppend(this.items, data);
                        this.lastPageActualSize = data.length;
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                });
            },
            reloadItems() {
                this.infiniteId += 1;
                console.log("Resetting infinite loader", this.infiniteId);
            },
            /**
             * Appends on replaces entity
             * @param dto
             */
            rerenderItem(dto) {
                console.log("Rerendering chat", dto);
                const replaced = replaceInArray(this.items, dto);
                console.debug("Replaced:", replaced);
                if (!replaced) {
                    this.reloadLastPage();
                }
                this.$forceUpdate();
            },
            reloadLastPage() {
                console.log("this.lastPageActualSize", this.lastPageActualSize);
                if (this.lastPageActualSize > 0) {
                    this.page--;
                    // remove lastPageActualSize
                    this.items.splice(-1, this.lastPageActualSize);
                    console.log("removing last", this.lastPageActualSize);
                } else {
                    this.page--;
                    // remove 20
                    this.items.splice(-1, pageSize);
                    console.log("removing last", pageSize);
                }
                this.reloadItems();
            },
            setSearchString(searchString) {
                this.searchString = searchString;
                this.items = [];
                this.page = 0;
                this.reloadItems();
            },
        },

    }
}