<template>

    <v-container style="height: calc(100vh - 64px); background: darkgrey">
        <div class="my-scroller" @scroll.passive="onScroll">
          <div class="first-element" style="min-height: 1px; background: #9cffa1"></div>
          <div v-for="item in items" :key="item.id" class="card mb-3" :id="getItemId(item.id)">
            <div class="row g-0">
              <div class="col">
                <img :src="item.owner.avatar" style="max-width: 64px; max-height: 64px">
              </div>
              <div class="col">
                <div class="card-body">
                  <h5 class="card-title">{{ item.text }}</h5>
                </div>
              </div>
            </div>
          </div>
          <div class="last-element" style="min-height: 1px; background: #c62828"></div>

        </div>

    </v-container>

</template>

<script>
    import throttle from "lodash/throttle";
    import debounce from "lodash/debounce";

    const directionTop = 'top';
    const directionBottom = 'bottom';

    const maxItemsLength = 200;
    const reduceToLength = 100;

    const PAGE_SIZE = 40;

    export default {
      data() {
        return {
          startingFromItemIdTop: 287,
          startingFromItemIdBottom: 287 - 1,
          items: [],


          isFirstLoad: true,

          scrollerDiv: null,

          loadedTop: false,
          loadedBottom: false,

          aDirection: directionTop,

          scrollerProbeCurrent: 0,
          scrollerProbePrevious: 0,
          scrollerProbePreviousPrevious: 0,

          preservedScroll: 0
        }
      },

      methods: {
        async load() {
          const startingFromItemId = this.isTopDirection() ? this.startingFromItemIdTop : this.startingFromItemIdBottom;
          return fetch( `/api/chat/1/message?startingFromItemId=${startingFromItemId}&size=${PAGE_SIZE}&reverse=${this.isTopDirection()}`)
            .then(res => res.json())
            .then((items) => {
            console.log("Get items", items, "page", this.startingFromItemIdTop, this.startingFromItemIdBottom, "chosen", startingFromItemId);

            if (this.isTopDirection()) {
              this.items = this.items.concat(items);
            } else {
              this.items = items.reverse().concat(this.items);
            }

            if (items.length < PAGE_SIZE) {
              if (this.isTopDirection()) {
                this.loadedTop = true;
              } else {
                this.loadedBottom = true;
              }
            } else {
              if (this.isTopDirection()) {
                this.startingFromItemIdTop = this.getMinimumItemId();
              } else {
                this.startingFromItemIdBottom = this.getMaximumItemId();
              }
            }
          }).then(()=>{
            return this.$nextTick()
          })
        },

        getMaximumItemId() {
          return Math.max(...this.items.map(it => it.id))
        },
        getMinimumItemId() {
          return Math.min(...this.items.map(it => it.id))
        },

        bottomElementSelector() {
          return ".first-element"
        },
        topElementSelector() {
          return ".last-element"
        },
        getItemId(id) {
          return 'item-' + id
        },

        cssStr(el) {
          return el.tagName.toLowerCase() + (el.id ? '#' + el.id : "") + '.' + (Array.from(el.classList)).join('.')
        },

        async reduceListIfNeed() {
          if (this.items.length > maxItemsLength) {
            return this.$nextTick(() => {
              console.log("Reducing to", maxItemsLength);
              if (this.isTopDirection()) {
                this.items = this.items.slice(-reduceToLength);
                this.startingFromItemIdBottom = this.getMaximumItemId();
                this.loadedBottom = false;
              } else {
                this.items = this.items.slice(0, reduceToLength);
                this.startingFromItemIdTop = this.getMinimumItemId();
                this.loadedTop = false;
              }
            });
          }
        },
        onScroll(e) {
          this.scrollerProbePreviousPrevious = this.scrollerProbePrevious;
          this.scrollerProbePrevious = this.scrollerProbeCurrent;
          this.scrollerProbeCurrent = this.scrollerDiv.scrollTop;
          console.debug("onScroll prevPrev=", this.scrollerProbePreviousPrevious , " prev=", this.scrollerProbePrevious, "cur=", this.scrollerProbeCurrent);

          this.trySwitchDirection();
        },
        trySwitchDirection() {
          if (this.scrollerProbeCurrent != 0 && this.scrollerProbeCurrent > this.scrollerProbePrevious && this.scrollerProbePrevious > this.scrollerProbePreviousPrevious && this.isTopDirection()) {
            this.aDirection = directionBottom;
            console.debug("Infinity scrolling direction has been changed to bottom");
          } else if (this.scrollerProbeCurrent != 0 && this.scrollerProbePreviousPrevious > this.scrollerProbePrevious && this.scrollerProbePrevious > this.scrollerProbeCurrent && !this.isTopDirection()) {
            this.aDirection = directionTop;
            console.debug("Infinity scrolling direction has been changed to top");
          } else {
            console.debug("Infinity scrolling direction has been remained untouched", this.aDirection);
          }
        },
        isTopDirection() {
          return this.aDirection === directionTop
        },

        saveScroll(bottom) {
          this.preservedScroll = bottom ? this.getMaximumItemId() : this.getMinimumItemId();
          console.log("Saved scroll", this.preservedScroll);
        },
        restoreScroll(bottom) {
          const restored = this.preservedScroll;
          console.log("Restored scrollTop to element id", restored);
          document.querySelector("#"+this.getItemId(restored)).scrollIntoView({behavior: 'instant', block: bottom ? "end" : "start"});
        },
        scrollDown() {
          this.$nextTick(() => {
            this.scrollerDiv.scrollTop = 0;
          });
        },
      },

      created() {
        this.onScroll = throttle(this.onScroll, 300, {leading:true, trailing:true});
      },

      mounted() {

        this.scrollerDiv = document.querySelector(".my-scroller");

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

          if (lastElementEntry && lastElementEntry.entry.isIntersecting) {
            console.log("going to load top");
            if (!this.loadedTop && this.isTopDirection()) {
              if (!this.isFirstLoad) {
                this.saveScroll(false);
              }
              await this.load();
              if (this.isFirstLoad) {
                this.scrollDown();
                this.isFirstLoad = false;
              } else {
                await this.reduceListIfNeed();
                this.restoreScroll(false);
              }
            }
          }
          if (firstElementEntry && firstElementEntry.entry.isIntersecting) {
            console.log("going to load bottom");
            if (!this.loadedBottom && !this.isTopDirection()) {
              this.saveScroll(true);
              await this.load();
              await this.reduceListIfNeed();
              this.restoreScroll(true);
            }
          }
        };

        const observerCallback = debounce(observerCallback0, 500, {leading:false, trailing:true});

        const observer = new IntersectionObserver(observerCallback, options);
        observer.observe(document.querySelector(this.bottomElementSelector()));
        observer.observe(document.querySelector(this.topElementSelector()));
      }
    }
</script>

<style lang="css">
    .my-scroller {
      height: 100%;
      overflow-y: scroll !important;
      display: flex;
      flex-direction: column-reverse;
    }

</style>
