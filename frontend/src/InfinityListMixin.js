import axios from 'axios';
import InfiniteLoading from 'vue-infinite-loading';

export const findIndex = (array, element) => {
    return array.findIndex(value => value.id === element.id);
};

const replaceInArray = (array, element) => {
    const foundIndex = findIndex(array, element);
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

const ACTION_CREATE = 'actionCreate';
const ACTION_EDIT = 'actionEdit';
const ACTION_DELETE = 'actionDelete';

export  {
    ACTION_CREATE,
    ACTION_EDIT,
    ACTION_DELETE,
}


export default (urlFunction, shouldChangeFunction) => {
    return  {
        data () {
            return {
                page: 0,
                items: [],
                itemsTotal: 0,
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
                    const list = data.data;
                    this.itemsTotal = data.totalCount;
                    if (list.length) {
                        this.page += 1;
                        this.items = [...this.items, ...list];
                        //replaceOrAppend(this.items, list);
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                });
            },
            // not working until you will change this.items list
            reloadItems() {
                this.infiniteId += 1;
                console.log("Resetting infinite loader", this.infiniteId);
            },
            // does should change items list (new item added to visible part or not for example)
            shouldChange(dto, action) {
                return shouldChangeFunction(this.$data, dto, action, this.isLastPage())
            },
            isLastPage() {
                const pagesTotal = Math.ceil(this.itemsTotal / pageSize);
                console.log("isLastPage pagesTotal=", pagesTotal, "this.page=", this.page, "this.itemsTotal=", this.itemsTotal);
                return this.page === pagesTotal;
            },
            addItem(dto) {
                if (this.shouldChange(dto, ACTION_CREATE)) {
                    console.log("Adding item", dto);
                    this.items.push(dto);
                    this.$forceUpdate();
                } else {
                    console.log("Item was not be added", dto);
                }
            },
            changeItem(dto) {
                if (this.shouldChange(dto, ACTION_EDIT)) {
                    console.log("Replacing item", dto);
                    replaceInArray(this.items, dto);
                    this.$forceUpdate();
                } else {
                    console.log("Item was not be replaced", dto);
                }
            },
            removeItem(dto) {
                if (this.shouldChange(dto, ACTION_DELETE)) {
                    console.log("Removing item", dto);
                    const idxToRemove = findIndex(this.items, dto);
                    this.items.splice(idxToRemove, 1);
                    this.$forceUpdate();
                } else {
                    console.log("Item was not be removed", dto);
                }
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