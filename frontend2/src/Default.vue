<template>

    <v-container style="height: calc(100vh - 64px); background: darkgrey">
        <div class="my-scroller">
          <div class="first-element" style="min-height: 1px; background: #9cffa1"></div>
          <div v-for="chat in chats" :key="chats.id" class="card mb-3">
            <div class="row g-0">
              <div class="col">
                <img :src="chat.avatar" class="rounded-start img-thumbnail m-1" alt="...">
              </div>
              <div class="col">
                <div class="card-body">
                  <h5 class="card-title">{{ chat.name }}</h5>
                </div>
              </div>
            </div>
          </div>
          <div class="last-element" style="min-height: 1px; background: #c62828"></div>

        </div>

    </v-container>

</template>

<script>
    // import "v3-infinite-loading/lib/style.css"; //required if you're not going to override default slots


    const css_str= (el) => {
      return el.tagName.toLowerCase() + (el.id ? '#' + el.id : "") + '.' + (Array.from(el.classList)).join('.')
    };


    const PAGE_SIZE = 5;

    export default {
      data() {
        return {
          page: 0,
          chats: [],
        }
      },

      methods: {
        loadChats(page) {
          return fetch( `/api/chat?page=${page}&size=${PAGE_SIZE}`)
            .then(res => res.json())
            .then(res => res.data);
        },

        load($state) {
          this.loadChats(this.page).then((chats) => {
            console.log("Get chats", chats);
            this.chats.push(...chats);
            this.page += 1;
            if (chats.length < PAGE_SIZE) {
              $state.complete();
            } else {
              $state.loaded();
            }
            // const state = action.loaded(chats.length, PAGE_SIZE);
            // console.log("Got state", state);
          })
        }
      },
      mounted() {
        // for(let i=0; i<1000; ++i) {
        //   this.chats.push({id: 1, avatar: "", name: "adsa" + i});
        // }

        // https://developer.mozilla.org/en-US/docs/Web/API/Intersection_Observer_API
        let options = {
          root: document.querySelector(".my-scroller"),
          rootMargin: "0px",
          threshold: 1.0,
        };
        let callback = (entries, observer) => {
          entries.forEach((entry) => {
            // Each entry describes an intersection change for one observed
            // target element:
            //   entry.boundingClientRect
            //   entry.intersectionRatio
            //   entry.intersectionRect
            //   entry.isIntersecting
            //   entry.rootBounds
            //   entry.target
            //   entry.time
            console.log("Entry: intersecting=", entry.isIntersecting, css_str(entry.target));
          });
        };
        let observer = new IntersectionObserver(callback, options);

        let target1 = document.querySelector(".first-element");
        observer.observe(target1);

        let target2 = document.querySelector(".last-element");
        observer.observe(target2);

        this.load({
          complete() {
            console.log("Complete");
          },
          loaded() {
            console.log("Loaded");
          }
        });
        console.log("this.chats", this.chats)
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
