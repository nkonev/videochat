<template>

    <v-container class="ma-0 pa-0" style="height: 100%" fluid>
        <div class="d-flex flex-wrap align-start" id="blog-post-list">

                <v-card
                    v-for="(item, index) in items"
                    :key="item.id"
                    class="mb-2 mr-2 myclass"
                    min-width="300"
                    max-width="500"
                >
                    <v-card-text class="pb-0">
                    <v-card>
                    <v-img
                        class="white--text align-end"
                        height="200px"
                        :src="item.imageUrl"
                    >
                        <v-container class="post-title ma-0 pa-0">
                            <v-card-title>
                                <router-link :to="getBlogPostLink(item)" class="post-title-text" v-html="item.title"></router-link>
                            </v-card-title>
                        </v-container>
                    </v-img>
                    </v-card>
                    </v-card-text>

                    <v-card-text class="text--primary pb-0">
                        {{ item.preview }}
                    </v-card-text>

                    <v-card-actions v-if="item?.owner != null">
                        <v-list-item class="grow">
                            <a @click.prevent="onParticipantClick(item.owner)" :href="getProfileLink(item.owner)">
                                <v-list-item-avatar>
                                    <v-img
                                        class="elevation-6"
                                        alt=""
                                        :src="item?.owner?.avatar"
                                    ></v-img>
                                </v-list-item-avatar>
                            </a>

                            <v-list-item-content>
                                <v-list-item-title><a @click.prevent="onParticipantClick(item)" :href="getProfileLink(item)">{{ item?.owner?.login }}</a></v-list-item-title>
                                <v-list-item-subtitle>
                                    {{ getDate(item) }}
                                </v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>
                    </v-card-actions>
                </v-card>

        </div>

        <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId">
            <template slot="no-more"><span/></template>
            <template slot="no-results"><span/></template>
        </infinite-loading>

    </v-container>

</template>

<script>
    import Vue from 'vue';
    import {getHumanReadableDate, hasLength, replaceOrAppend} from "@/utils";
    import axios from "axios";
    import InfiniteLoading from './lib/vue-infinite-loading/src/components/InfiniteLoading.vue';
    import {GET_SEARCH_STRING, SET_SEARCH_NAME, SET_SHOW_SEARCH} from "@/blogStore";
    import debounce from "lodash/debounce";
    import bus, {SEARCH_STRING_CHANGED} from "@/blogBus";
    import Mark from "mark.js";
    import {blog_post_name} from "@/blogRoutes";
    import {profile, profile_name} from "@/routes";

    const pageSize = 40;

    export default {
        data() {
            return {
                items: [],
                page: 0,
                infiniteId: +new Date(),
                markInstance: null,
            }
        },
        methods: {
            infiniteHandler($state) {
                axios.get('/api/blog', {
                    params: {
                        page: this.page,
                        size: pageSize,
                        searchString: this.searchString,
                    },
                }).then(({data}) => {
                    const list = data;
                    if (list.length) {
                        this.page += 1;
                        replaceOrAppend(this.items, list);
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                    this.performMarking();
                });
            },
            getDate(item) {
                return getHumanReadableDate(item.createDateTime)
            },
            reloadItems() {
                this.infiniteId += 1;
                console.log("Resetting infinite loader", this.infiniteId);
            },
            resetVariables() {
                this.items = [];
                this.page = 0;
            },
            searchStringChangedDebounced(searchString) {
                this.searchStringChangedStraight(searchString);
            },
            searchStringChangedStraight(searchString) {
                this.resetVariables();
                this.reloadItems();
            },
            performMarking() {
                Vue.nextTick(() => {
                    if (hasLength(this.searchString)) {
                        this.markInstance.unmark();
                        this.markInstance.mark(this.searchString);
                    }
                })
            },
            getBlogPostLink(item) {
                return {
                    name: blog_post_name,
                    params: {
                        id: item.id
                    }
                }
            },
            onParticipantClick(user) {
                const routeDto = { name: profile_name, params: { id: user.id }};
                this.$router.push(routeDto);
            },
            getProfileLink(user) {
                let url = profile + "/" + user.id;
                return url;
            },
        },
        components: {
            InfiniteLoading
        },
        computed: {
            searchString: {
                get() {
                    return this.$store.getters[GET_SEARCH_STRING];
                },
            }
        },
        mounted() {
            this.markInstance = new Mark("div#blog-post-list");

            this.$store.commit(SET_SEARCH_NAME, 'Search by posts');
            this.$store.commit(SET_SHOW_SEARCH, true);
        },
        created() {
            this.searchStringChangedDebounced = debounce(this.searchStringChangedDebounced, 700, {
                leading: false,
                trailing: true
            });
            bus.$on(SEARCH_STRING_CHANGED, this.searchStringChangedDebounced);
        },
        destroyed() {
            bus.$off(SEARCH_STRING_CHANGED, this.searchStringChangedDebounced);
        },
    }
</script>

<style lang="stylus">
    .post-title {
        background rgba(0, 0, 0, 0.5);

        .post-title-text {
            color white
            text-decoration none
        }
    }
    .myclass {
        flex: 1 1 300px;
    }
</style>
