import debounce from "lodash/debounce";
import {isFireFox} from "@/utils.js";

export const directionTop = 'top';
export const directionBottom = 'bottom';

// expects getMaxItemsLength(),
// bottomElementSelector(), topElementSelector(), getItemId(id),
// load(), onFirstLoad(), initialDirection(), saveScroll(), scrollerSelector(),
// reduceTop(), reduceBottom()
// onScrollCallback(), afterScrollRestored()
// onScroll() should be called from template
// updateLastUpdateDateTime() (optionally)

// stop-scrolling class (in App.vue)
export default (name) => {
  return {
    data() {
      return {
        items: [],
        observer: null,

        isFirstLoad: true,

        scrollerDiv: null,

        aDirection: this.initialDirection(),

        scrollerProbeCurrent: 0,
        scrollerProbePrevious: 0,

        preservedScroll: 0,
      }
    },
    methods: {
      cssStr(el) {
        return el.tagName.toLowerCase() + (el.id ? '#' + el.id : "") + '.' + (Array.from(el.classList)).join('.')
      },

      async reduceListIfNeed() {
        if (this.items.length > this.getMaxItemsLength()) {
          return this.$nextTick(() => {
            if (this.isTopDirection()) {
                this.reduceBottom();
            } else {
                this.reduceTop();
            }
            console.log("Reduced to", this.getMaxItemsLength(), "in", name);
          });
        }
      },
      reduceListAfterAdd(fromTop) {
        if (this.items.length > this.getMaxItemsLength()) {
            if (fromTop) {
                this.reduceTop();
            } else {
                this.reduceBottom();
            }
            console.log("Reduced after add to", this.getMaxItemsLength(), "in", name);
        }
      },
      onScroll(e) {
        if (this.onScrollCallback) {
          this.onScrollCallback();
        }

        this.scrollerProbePrevious = this.scrollerProbeCurrent;
        this.scrollerProbeCurrent = this.scrollerDiv.scrollTop;
        // console.debug("onScroll in", name, " prev=", this.scrollerProbePrevious, "cur=", this.scrollerProbeCurrent);

        this.trySwitchDirection();
      },
      trySwitchDirection() {
        if (this.scrollerProbeCurrent != 0 && this.scrollerProbeCurrent > this.scrollerProbePrevious && this.isTopDirection()) {
          this.aDirection = directionBottom;
          // console.debug("Infinity scrolling direction has been changed to bottom");
        } else if (this.scrollerProbeCurrent != 0 && this.scrollerProbePrevious > this.scrollerProbeCurrent && !this.isTopDirection()) {
          this.aDirection = directionTop;
          // console.debug("Infinity scrolling direction has been changed to top");
        } else {
          // console.debug("Infinity scrolling direction has been remained untouched", this.aDirection);
        }
      },
      isTopDirection() {
        return this.aDirection === directionTop
      },

      async restoreScroll(top) {
          return this.$nextTick(()=>{
            const restored = this.preservedScroll;
            const q = this.scrollerSelector() + " " + "#"+this.getItemId(restored);
            const el = document.querySelector(q);
            console.debug("Restored scroll to element id", restored, "in", name, "selector", q, "element", el);
            el?.scrollIntoView({behavior: 'instant', block: top ? "start": "end"});
            if (this.afterScrollRestored) {
              this.afterScrollRestored(el)
            }
          })
      },

      // invoked from reset()
      resetInfiniteScrollVars() {
          this.items = [];
          this.isFirstLoad = true;
          this.aDirection = this.initialDirection();
          this.scrollerProbePrevious = 0;
          this.scrollerProbeCurrent = 0;
          this.preservedScroll = null;
      },
      async setNoScroll() {
          return this.$nextTick(()=>{
              if (isFireFox()) { // This works well only on Firefox, in case Chrome it both doesn't needed and breaks pagination in a random manner
                  this.scrollerDiv.classList.add("stop-scrolling");
              }
          })
      },
      async unsetNoScroll() {
          return this.$nextTick(()=>{
              if (isFireFox()) {
                  this.scrollerDiv.classList.remove("stop-scrolling");
              }
          })
      },
      async initialLoad() {
        await this.$nextTick(()=>{
            if (this.scrollerDiv == null) {
                this.scrollerDiv = document.querySelector(this.scrollerSelector());
            }
        })
        await this.setNoScroll();
        const loadedResult = await this.load();
        await this.unsetNoScroll()
        await this.$nextTick();
        await this.onFirstLoad(loadedResult);
        this.isFirstLoad = false;
      },

      async loadTop() {
          console.log("going to load top in", name);
          this.saveScroll(true); // saves scroll between new portion load
          await this.setNoScroll();
          await this.load(); // restores scroll after new portion load
          await this.$nextTick();
          await this.reduceListIfNeed();
          await this.restoreScroll(true);
          await this.unsetNoScroll()
      },

      async loadBottom() {
          console.log("going to load bottom in", name);
          this.saveScroll(false);
          await this.setNoScroll();
          await this.load();
          await this.$nextTick();
          await this.reduceListIfNeed();
          await this.restoreScroll(false);
          await this.unsetNoScroll()
      },
      isReady() {
          return this.scrollerDiv != null
      },
      initScroller() {
        if (!this.isReady()) {
          throw "You have to invoke initialLoad() first"
        }

        // https://developer.mozilla.org/en-US/docs/Web/API/Intersection_Observer_API
        const options = {
            root: this.scrollerDiv,
            rootMargin: "0px",
            threshold: 0.0,
        };
        const observerCallback0 = async (entries, observer) => {
          const mappedEntries = entries.map((entry) => {
            return {
              entry,
              elementName: this.cssStr(entry.target)
            }
          });
          const lastElementEntries = mappedEntries.filter(en => en.entry.intersectionRatio > 0 && en.elementName.includes(this.topElementSelector()));
          const lastElementEntry = lastElementEntries.length ? lastElementEntries[lastElementEntries.length-1] : null;

          const firstElementEntries = mappedEntries.filter(en => en.entry.intersectionRatio > 0 && en.elementName.includes(this.bottomElementSelector()));
          const firstElementEntry = firstElementEntries.length ? firstElementEntries[firstElementEntries.length-1] : null;

          console.log("Invoking callback in", name, mappedEntries);

          if (this.items.length && lastElementEntry && lastElementEntry.entry.isIntersecting) {
            console.debug("attempting to load top", this.isTopDirection(), "in", name);
            if (this.isTopDirection()) {
              await this.loadTop();
            }
          }
          if (this.items.length && firstElementEntry && firstElementEntry.entry.isIntersecting) {
            console.debug("attempting to load bottom", !this.isTopDirection(), "in", name);
            if (!this.isTopDirection()) {
              await this.loadBottom();
            }
          }
        };

        const observerCallback = debounce(observerCallback0, 200, {leading:false, trailing:true});

        this.observer = new IntersectionObserver(observerCallback, options);
        this.observer.observe(document.querySelector(this.scrollerSelector() + " " + this.bottomElementSelector()));
        this.observer.observe(document.querySelector(this.scrollerSelector() + " " + this.topElementSelector()));
      },
      async destroyScroller() {
          return this.$nextTick(()=>{
              this.observer?.disconnect();
              this.observer = null;
              this.scrollerDiv = null;
          })
      },
      async installScroller() {
        return this.$nextTick(()=>{
            this.initScroller();
            console.log("Scroller", name, "has been installed");
        })
        // tests in Firefox
        // a) refresh page 30 times
        // b) refresh page 30 times when the hash is present (#message-523)
        // c) input search string - search by messages
      },
      async uninstallScroller() {
        await this.destroyScroller();
        this.reset();
        console.log("Scroller", name, "has been uninstalled");
      },
      async reloadItems() {
        await this.uninstallScroller();
        await this.$nextTick();
        if (this.updateLastUpdateDateTime) {
              this.updateLastUpdateDateTime();
        }
        await this.initialLoad();
        await this.$nextTick(async () => {
          await this.installScroller();
        });
      },
    }
  }
}
