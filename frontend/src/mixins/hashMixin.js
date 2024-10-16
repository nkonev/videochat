import {hasLength} from "@/utils";
import {directionBottom} from "@/mixins/infiniteScrollMixin.js";

// expects methods: doDefaultScroll(), getPositionFromStore(), conditionToSaveLastVisible(), itemSelector(), doSaveTheFirstItem(), setPositionToStore(), scrollerSelector(), itemSelector(), initialDirection(), isAppropriateHash()
// isTopDirection() - from infiniteScrollMixin.js
export default () => {
    return {
        data() {
            return {
                startingFromItemIdTop: null,
                startingFromItemIdBottom: null,
                // those two doesn't play in reset() in order to survive after reload()
                hasHashFromRoute: false, // do we have hash in address line (message id)
                loadedFromStoreHash: null, // keeps loaded message id from localstore the most top visible message - preserves scroll between page reload or switching between chats
            }
        },
        computed: {
            highlightItemId() {
                if (this.isAppropriateHash(this.$route.hash)) {
                    return this.getIdFromRouteHash(this.$route.hash);
                } else {
                    return null
                }
            },
        },
        methods: {
            getDefaultItemId() {
                return this.isTopDirection() ? this.startingFromItemIdTop : this.startingFromItemIdBottom;
            },
            initializeHashVariables() {
                this.hasHashFromRoute = hasLength(this.highlightItemId);
                this.loadedFromStoreHash = this.getPositionFromStore();
            },
            prepareHashesForRequest() {
                let startingFromItemId;
                let hasHash;
                if (this.hasHashFromRoute) { // we need it here - it shouldn't be computable in order to be reset. The resetted value is need when we press "arrow down" after reload
                    // how to check:
                    // 1. click on hash
                    // 2. reload page
                    // 3. press "arrow down" (Scroll down)
                    // 4. It is going to invoke this load method which will use cashed and reset hasHashFromRoute = false
                    startingFromItemId = this.highlightItemId;
                    hasHash = true;
                } else if (this.loadedFromStoreHash) {
                    startingFromItemId = this.loadedFromStoreHash;
                    hasHash = true;
                } else {
                    startingFromItemId = this.getDefaultItemId();
                    hasHash = false;
                }
                return {startingFromItemId, hasHash}
            },
            async doScrollOnFirstLoad(prefix) {
                if (this.highlightItemId) {
                    await this.scrollTo(prefix + this.highlightItemId);
                } else if (this.loadedFromStoreHash) {
                    await this.scrollTo(prefix + this.loadedFromStoreHash);
                } else {
                    await this.doDefaultScroll(); // we need it to prevent browser's scrolling
                }
                this.loadedFromStoreHash = null;
                this.hasHashFromRoute = false;
            },
            async scrollTo(newValue) {
                return await this.$nextTick(()=>{
                    const el = document.querySelector(newValue);
                    el?.scrollIntoView({behavior: 'instant', block: "start"});
                    return el
                })
            },
            async scrollToOrLoad(newValue, isTheSameQuery) {
                let res;
                if (isTheSameQuery) {
                    res = await this.scrollTo(newValue);
                }
                if (!res) {
                    console.log("Didn't scrolled or different queries, resetting");
                    await this.initializeHashVariablesAndReloadItems();
                }
            },
            clearRouteHash() {
                // console.log("Cleaning hash");
                this.$router.push({ hash: null, query: this.$route.query })
            },
            async initializeHashVariablesAndReloadItems() {
                this.initializeHashVariables();
                await this.reloadItems();
            },
            saveLastVisibleElement(obj) {
                console.log("saveLastVisibleElement", this.conditionToSaveLastVisible());
                if (this.conditionToSaveLastVisible()) {
                    const elems = [...document.querySelectorAll(this.scrollerSelector() + " " + this.itemSelector())].map((item) => {
                        const visible = item.getBoundingClientRect().top > 10 // 10 only for ChatList (on mobile Chrome), 0 is enough for UserList amd MessageList
                        return {item, visible}
                    });

                    const visible = elems.filter((el) => el.visible);
                    // console.log("visible", visible, "elems", elems);
                    if (visible.length == 0) {
                        console.warn("Unable to get desiredVisible")
                        return
                    }
                    const desiredVisible = this.doSaveTheFirstItem() ?  visible[0].item : visible[visible.length - 1].item;

                    const iid = this.getIdFromRouteHash(desiredVisible.id);
                    console.log("For storing to localstore found desiredVisible", desiredVisible, "itemId", iid, "obj", obj);

                    this.setPositionToStore(iid, obj)
                } else {
                    console.log("Skipped saved desiredVisible because we are already scrolled")
                }
            },
            doSaveTheFirstItem() {
                return this.initialDirection() == directionBottom
            }
        },
    }
}
