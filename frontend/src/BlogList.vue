<template>

    <div class="d-flex flex-wrap align-start">

            <v-card
                v-for="(item, index) in items"
                :key="item.id"
                class="mb-2 mr-2 myclass"
                min-width="300"
                max-width="500"
            >
                <v-img
                    class="white--text align-end"
                    height="200px"
                    :src="item.imageUrl"
                >
                    <v-container class="post-title ma-0 pa-0">
                        <v-card-title class="text-h5 font-weight-bold">{{ item.title }}</v-card-title>
                    </v-container>
                </v-img>

                <v-card-subtitle class="pb-0">
                    {{ getDate(item) }}
                </v-card-subtitle>

                <v-card-text class="text--primary pb-0">
                    {{ item.preview }}
                </v-card-text>

                <v-card-actions>
                    <v-list-item class="grow">
                        <v-list-item-avatar color="grey darken-3">
                            <v-img
                                class="elevation-6"
                                alt=""
                                :src="item?.owner?.avatar"
                            ></v-img>
                        </v-list-item-avatar>

                        <v-list-item-content>
                            <v-list-item-title>{{ item?.owner?.login }}</v-list-item-title>
                        </v-list-item-content>
                    </v-list-item>
                </v-card-actions>
            </v-card>

    </div>

</template>

<script>
import {getHumanReadableDate, replaceOrAppend} from "@/utils";
    import axios from "axios";

    export default {
        data() {
            return {
                items: [],
                page: 0,
                infiniteId: +new Date(),
                itemsLoaded: false,
            }
        },
        methods: {
            loadItems() {
                this.itemsLoaded = false;
                axios.get('/api/blog', {
                    // params: {
                    //     page: this.page,
                    //     size: pageSize,
                    //     searchString: this.searchString,
                    // },
                }).then(({ data }) => {
                    const list = data;
                    if (list.length) {
                        this.page += 1;
                        replaceOrAppend(this.items, list);
                        // $state.loaded();
                    } else {
                        // $state.complete();
                    }
                    this.itemsLoaded = true;
                });
            },
            getDate(item) {
                return getHumanReadableDate(item.createDateTime)
            },
        },
        mounted() {
            this.loadItems()
        }
    }
</script>

<style lang="stylus">
    .post-title {
        background rgba(0, 0, 0, 0.5);
    }
    .myclass {
        flex: 1 1 300px;
    }
</style>
