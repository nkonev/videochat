import axios from 'axios';
import InfiniteLoading from 'vue-infinite-loading';

export const findIndex = (array, element) => {
    return array.findIndex(value => value.id === element.id);
};

export const replaceInArray = (array, element) => {
    const foundIndex = findIndex(array, element);
    if (foundIndex === -1) {
        return false;
    } else {
        array[foundIndex] = element;
        return true;
    }
};

export const replaceOrAppend = (array, newArray) => {
    newArray.forEach((element, index) => {
        const replaced = replaceInArray(array, element);
        if (!replaced) {
            array.push(element);
        }
    });
};

export const pageSize = 20;

const ACTION_CREATE = 'actionCreate';
const ACTION_EDIT = 'actionEdit';
const ACTION_DELETE = 'actionDelete';

export  {
    ACTION_CREATE,
    ACTION_EDIT,
    ACTION_DELETE,
}


export default () => {
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
            // not working until you will change this.items list
            reloadItems() {
                this.infiniteId += 1;
                console.log("Resetting infinite loader", this.infiniteId);
            },

            isLastPage() {
                const pagesTotal = Math.ceil(this.itemsTotal / pageSize);
                console.log("isLastPage pagesTotal=", pagesTotal, "this.page=", this.page, "this.itemsTotal=", this.itemsTotal);
                return this.page === pagesTotal;
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