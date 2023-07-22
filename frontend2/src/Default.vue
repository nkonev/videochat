<template>

    <v-container>
        <div class="my-scroller">
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

          <InfiniteLoading @infinite="load" top></InfiniteLoading>
        </div>

    </v-container>

</template>

<script>
    import InfiniteLoading from "@/lib/infinite-scrolling/components/InfiniteLoading.vue";
    // import "v3-infinite-loading/lib/style.css"; //required if you're not going to override default slots

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
      components: {
        InfiniteLoading
      }
    }
</script>

<style lang="css">
    .my-scroller {
      display: flex;
      flex-direction: column-reverse;
    }

</style>
