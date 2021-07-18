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

export const moveToFirstPosition = (array, element) => {
    const idx = findIndex(array, element);
    if (idx > 0) {
        array.splice(idx, 1);
        array.unshift(element);
    }
}

export const pageSize = 20;

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