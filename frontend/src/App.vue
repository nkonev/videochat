<template>
    <v-app>
        <!-- https://vuetifyjs.com/en/components/application/ -->
        <v-app-bar
                color="indigo"
                dark
                app
        >
            <v-app-bar-nav-icon></v-app-bar-nav-icon>

            <v-btn icon @click="openModal">
                <v-icon>mdi-plus-circle-outline</v-icon>
            </v-btn>

            <v-spacer></v-spacer>
            <v-toolbar-title>Chats</v-toolbar-title>
            <v-spacer></v-spacer>

            <v-btn icon>
                <v-icon>mdi-magnify</v-icon>
            </v-btn>
        </v-app-bar>

        <v-main>
            <v-container>
                <v-card
                        max-width="1000"
                        class="mx-auto"
                >
                    <EditChat v-model="openEditModal"/>

                    <v-list>
                            <v-list-item
                                    v-for="(item, index) in items"
                                    :key="item.id"
                                    @click=""
                            >
                                <v-list-item-content>
                                    <v-list-item-title v-html="item.name"></v-list-item-title>
                                    <v-list-item-subtitle v-html="item.participantIds"></v-list-item-subtitle>
                                </v-list-item-content>
                            </v-list-item>
                    </v-list>
                    <infinite-loading @infinite="infiniteHandler"></infinite-loading>

                </v-card>
            </v-container>
        </v-main>
    </v-app>
</template>

<script>
    import axios from 'axios';
    import EditChat from "./EditChat";
    import InfiniteLoading from 'vue-infinite-loading';

    export default {
        data () {
            return {
                page: 0,
                items: [],
                openEditModal: false
            }
        },
        components:{
            EditChat,
            InfiniteLoading
        },
        methods:{
            openModal() {
                this.$data.openEditModal = true;
            },
            infiniteHandler($state) {
                axios.get(`/api/chat`, {
                    params: {
                        page: this.page,
                    },
                }).then(({ data }) => {
                    if (data.length) {
                        this.page += 1;
                        this.items.push(...data);
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                });
            },
        }
    }
</script>

<style lang="stylus" scoped>
    .application {
        font-family: Arial, sans-serif;
        -webkit-font-smoothing: antialiased;
        -moz-osx-font-smoothing: grayscale;


        #input-usage .v-input__prepend-outer,
        #input-usage .v-input__append-outer,
        #input-usage .v-input__slot,
        #input-usage .v-messages {
            border: 1px dashed rgba(0,0,0, .4);
        }
    }
</style>
