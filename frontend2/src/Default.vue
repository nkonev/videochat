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


    const cssStr= (el) => {
      return el.tagName.toLowerCase() + (el.id ? '#' + el.id : "") + '.' + (Array.from(el.classList)).join('.')
    };


    const PAGE_SIZE = 10;

    export default {
      data() {
        return {
          page: 0,
          chats: [],


          loadedTop: false,
          loadedBottom: false,
        }
      },

      methods: {
        loadChats(page) {
          return fetch( `/api/chat?page=${page}&size=${PAGE_SIZE}`)
            .then(res => res.json())
            .then(res => res.data);
        },

        load() {
          this.loadChats(this.page).then((chats) => {
            console.log("Get chats", chats, "page", this.page);
            this.chats.push(...chats);
            if (chats.length < PAGE_SIZE) {
              this.loadedTop = true;
            } else {
              this.page += 1;
            }
            // const state = action.loaded(chats.length, PAGE_SIZE);
            // console.log("Got state", state);
          })
        }
      },
      mounted() {
        // https://developer.mozilla.org/en-US/docs/Web/API/Intersection_Observer_API
        let options = {
          root: document.querySelector(".my-scroller"),
          rootMargin: "0px",
          threshold: 0.0,
        };
        const observerCallback = (entries, observer) => {
          const mappedEntries = entries.map((entry) => {
            return {
              entry,
              elementName: cssStr(entry.target)
            }
          });
          const lastElementEntries = mappedEntries.filter(en => en.elementName.includes(".last-element"));
          const lastElementEntry = lastElementEntries.length ? lastElementEntries[lastElementEntries.length-1] : null;

          const firstElementEntries = mappedEntries.filter(en => en.elementName.includes(".first-element"));
          const firstElementEntry = firstElementEntries.length ? firstElementEntries[firstElementEntries.length-1] : null;

          if (lastElementEntry && lastElementEntry.entry.isIntersecting && !this.loadedTop) {
            console.log("load top");
            this.load();
          }
          if (firstElementEntry && firstElementEntry.entry.isIntersecting && !this.loadedBottom) {
            console.log("load bottom");
          }
        };

        const observer = new IntersectionObserver(observerCallback, options);
        observer.observe(document.querySelector(".first-element"));
        observer.observe(document.querySelector(".last-element"));

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
