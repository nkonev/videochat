import {hasLength} from "@/utils";
import {getTopMessagePosition} from "@/store/localStore";

// expects methods: doDefaultScroll(), getPositionFromStore(), getScrollerName(). isTopDirection() - from infiniteScrollMixin.js
export default () => {
    return {
        data() {
            return {
                startingFromItemIdTop: null,
                startingFromItemIdBottom: null,
                // those two doesn't play in reset() in order to survive after reload()
                hasInitialHash: false, // do we have hash in address line (message id)
                loadedHash: null, // keeps loaded message id from localstore the most top visible message - preserves scroll between page reload or switching between chats
            }
        },
        computed: {
            highlightItemId() {
                return this.getIdFromRouteHash(this.$route.hash);
            },
        },
        methods: {
            getDefaultItemId() {
                return this.isTopDirection() ? this.startingFromItemIdTop : this.startingFromItemIdBottom;
            },
            setHashes() {
                this.hasInitialHash = hasLength(this.highlightItemId);
                this.loadedHash = this.getPositionFromStore();
            },
            prepareHashesForLoad() {
                let startingFromItemId;
                let hasHash;
                if (this.hasInitialHash) { // we need it here - it shouldn't be computable in order to be reset. The resetted value is need when we press "arrow down" after reload
                    // how to check:
                    // 1. click on hash
                    // 2. reload page
                    // 3. press "arrow down" (Scroll down)
                    // 4. It is going to invoke this load method which will use cashed and reset hasInitialHash = false
                    startingFromItemId = this.highlightItemId;
                    hasHash = true;
                } else if (this.loadedHash) {
                    startingFromItemId = this.loadedHash;
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
                } else if (this.loadedHash) {
                    await this.scrollTo(prefix + this.loadedHash);
                } else {
                    await this.doDefaultScroll(); // we need it to prevent browser's scrolling
                }
                this.loadedHash = null;
                this.hasInitialHash = false;
            },
            updateTopAndBottomIds() {
                this.startingFromItemIdTop = this.getMinimumItemId();
                this.startingFromItemIdBottom = this.getMaximumItemId();
            },
            async scrollTo(newValue) {
                return await this.$nextTick(()=>{
                    const el = document.querySelector(newValue);
                    el?.scrollIntoView({behavior: 'instant', block: "start"});
                    return el
                })
            },
            async scrollToOrLoad(newValue) {
                const res = await this.scrollTo(newValue);
                if (!res) {
                    console.log("Didn't scrolled, resetting");
                    await this.setHashAndReloadItems();
                }
            },
            getMaximumItemId() {
                return this.items.length ? Math.max(...this.items.map(it => it.id)) : null
            },
            getMinimumItemId() {
                return this.items.length ? Math.min(...this.items.map(it => it.id)) : null
            },
            saveScroll(top) {
                this.preservedScroll = top ? this.getMinimumItemId() : this.getMaximumItemId();
                console.log("Saved scroll", this.preservedScroll, "in ", this.getScrollerName());
            },
            clearRouteHash() {
                // console.log("Cleaning hash");
                this.$router.push({ hash: null, query: this.$route.query })
            },
            async setHashAndReloadItems() {
                this.setHashes();
                await this.reloadItems();
            },
        }
    }
}
