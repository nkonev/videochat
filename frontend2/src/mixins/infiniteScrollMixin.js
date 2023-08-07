import debounce from "lodash/debounce";

export const directionTop = 'top';
export const directionBottom = 'bottom';

export const maxItemsLength = 200;
export const reduceToLength = 100;

// expects bottomElementSelector(), topElementSelector(), getItemId(id),
// load(), onFirstLoad(), initialDirection(), saveScroll(), scrollerSelector(),
// reduceTop(), reduceBottom()
// onScroll() should be called from template
export default (name) => {
  let observer;
  return {
    data() {
      return {
        items: [],


        isFirstLoad: true,

        scrollerDiv: null,

        loadedTop: false,
        loadedBottom: false,

        aDirection: this.initialDirection(),

        scrollerProbeCurrent: 0,
        scrollerProbePrevious: 0,
        scrollerProbePreviousPrevious: 0,

        preservedScroll: 0,

      }
    },
    methods: {
      cssStr(el) {
        return el.tagName.toLowerCase() + (el.id ? '#' + el.id : "") + '.' + (Array.from(el.classList)).join('.')
      },

      async reduceListIfNeed() {
        if (this.items.length > maxItemsLength) {
          return this.$nextTick(() => {
            if (this.isTopDirection()) {
                this.reduceBottom();
                this.loadedBottom = false;
            } else {
                this.reduceTop();
                this.loadedTop = false;
            }
            console.log("Reduced to", maxItemsLength, this.loadedBottom, this.loadedTop, "in", name);
          });
        }
      },
      onScroll(e) {
        this.scrollerProbePreviousPrevious = this.scrollerProbePrevious;
        this.scrollerProbePrevious = this.scrollerProbeCurrent;
        this.scrollerProbeCurrent = this.scrollerDiv.scrollTop;
        // console.debug("onScroll prevPrev=", this.scrollerProbePreviousPrevious , " prev=", this.scrollerProbePrevious, "cur=", this.scrollerProbeCurrent);

        this.trySwitchDirection();
      },
      trySwitchDirection() {
        if (this.scrollerProbeCurrent != 0 && this.scrollerProbeCurrent > this.scrollerProbePrevious && this.scrollerProbePrevious > this.scrollerProbePreviousPrevious && this.isTopDirection()) {
          this.aDirection = directionBottom;
          // console.debug("Infinity scrolling direction has been changed to bottom");
          if (this.onChangeDirection) {
            this.onChangeDirection();
          }
        } else if (this.scrollerProbeCurrent != 0 && this.scrollerProbePreviousPrevious > this.scrollerProbePrevious && this.scrollerProbePrevious > this.scrollerProbeCurrent && !this.isTopDirection()) {
          this.aDirection = directionTop;
          // console.debug("Infinity scrolling direction has been changed to top");
          if (this.onChangeDirection) {
            this.onChangeDirection();
          }
        } else {
          // console.debug("Infinity scrolling direction has been remained untouched", this.aDirection);
        }
      },
      isTopDirection() {
        return this.aDirection === directionTop
      },

      restoreScroll(top) {
        const restored = this.preservedScroll;
        const q = this.scrollerSelector() + " " + "#"+this.getItemId(restored);
        const el = document.querySelector(q);
        console.debug("Restored scroll to element id", restored, "in", name, "selector", q, "element", el);
        el?.scrollIntoView({behavior: 'instant', block: top ? "start": "end"});
      },

      resetInfiniteScrollVars() {
          this.items = [];
          this.isFirstLoad = true;
          this.loadedTop = false;
          this.loadedBottom = false;
          this.aDirection = this.initialDirection();
      },

      async loadTop() {
          console.log("going to load top in", name);
          if (!this.isFirstLoad) {
              this.saveScroll(this.initialDirection());
          }
          await this.load();
          if (this.isFirstLoad) {
              this.onFirstLoad();
              this.isFirstLoad = false;
          } else {
              await this.reduceListIfNeed();
              this.restoreScroll(this.initialDirection());
          }
      },

      async loadBottom() {
          console.log("going to load bottom in", name);
          if (!this.isFirstLoad) {
              this.saveScroll(!this.initialDirection());
          }
          await this.load();
          if (this.isFirstLoad) {
              this.onFirstLoad();
              this.isFirstLoad = false;
          } else {
              await this.reduceListIfNeed();
              this.restoreScroll(!this.initialDirection());
          }
      },

      initScroller() {
        this.scrollerDiv = document.querySelector(this.scrollerSelector());

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

          if (lastElementEntry && lastElementEntry.entry.isIntersecting) {
            console.debug("attempting to load top", !this.loadedTop, this.isTopDirection(), "in", name);
            if (!this.loadedTop && this.isTopDirection()) {
              await this.loadTop();
            }
          }
          if (firstElementEntry && firstElementEntry.entry.isIntersecting) {
            console.debug("attempting to load bottom", !this.loadedBottom, !this.isTopDirection(), "in", name);
            if (!this.loadedBottom && !this.isTopDirection()) {
              await this.loadBottom();
            }
          }
        };

        const observerCallback = debounce(observerCallback0, 100, {leading:false, trailing:true});

        observer = new IntersectionObserver(observerCallback, options);
        observer.observe(document.querySelector(this.scrollerSelector() + " " + this.bottomElementSelector()));
        observer.observe(document.querySelector(this.scrollerSelector() + " " + this.topElementSelector()));
      },
      destroyScroller() {
        observer.disconnect()
      }
    }
  }
}
