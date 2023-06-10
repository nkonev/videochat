<template>

    <v-container class="fill-height">
      <v-responsive class="align-center text-center fill-height">
        <v-img height="300" src="@/assets/logo.svg" />

        <div class="text-body-2 font-weight-light mb-n1">Welcome to</div>

        <h1 class="text-h2 font-weight-bold">Vuetify</h1>

        <div class="py-14" />

        <v-row class="d-flex align-center justify-center">

          <v-col cols="auto">
            <v-btn
              color="primary"
              min-width="228"
              size="x-large"
              variant="flat"
              :to="{ name: 'list'}"
            >
              <v-icon
                icon="mdi-speedometer"
                size="large"
                start
              />
              Get Started
            </v-btn>
          </v-col>

        </v-row>

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

        <vue-eternal-loading :load="load"></vue-eternal-loading>
      </v-responsive>
    </v-container>

</template>

<script>
    import { VueEternalLoading } from '@ts-pro/vue-eternal-loading';

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

        load(action) {
          this.loadChats(this.page).then((chats) => {
            console.log("Get chats", chats);
            this.chats.push(...chats);
            this.page += 1;
            const state = action.loaded(chats.length, PAGE_SIZE);
            console.log("Got state", state);
          })
        }
      },
      components: {
        VueEternalLoading
      }
    }
</script>

<style lang="css">
    .vue-eternal-loading>div {
      margin: 10px;
      text-align: center;
    }

</style>
