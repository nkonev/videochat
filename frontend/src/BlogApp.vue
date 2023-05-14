<template>
    <v-app>
        <v-app-bar app color='indigo' dark dense>
            <v-breadcrumbs
                :items="getBreadcrumbs()"
            >
            </v-breadcrumbs>

            <v-spacer></v-spacer>

            <v-btn v-if="isShowSearch && showSearchButton && isMobile()" icon :title="searchName" @click="onOpenSearch()">
                <v-icon>{{ hasSearchString ? 'mdi-magnify-close' : 'mdi-magnify'}}</v-icon>
            </v-btn>
            <v-card v-if="isShowSearch && !showSearchButton || !isMobile()" light :width="isMobile() ? '100%' : ''">
                <v-text-field prepend-icon="mdi-magnify" hide-details single-line v-model="searchString" :label="searchName" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput" @blur="showSearchButton=true"></v-text-field>
            </v-card>
        </v-app-bar>

        <!-- Sizes your content based upon application components -->
        <v-main>

            <!-- Provides the application the proper gutter -->
            <v-container fluid>

                <!-- If using vue-router -->
                <router-view></router-view>

            </v-container>
        </v-main>
    </v-app>
</template>

<script>
import {GET_SEARCH_NAME, GET_SEARCH_STRING, GET_SHOW_SEARCH, SET_SEARCH_STRING} from "@/blogStore";
import {mapGetters} from 'vuex'
import bus, {SEARCH_STRING_CHANGED} from "@/blogBus";
import {blog, blog_post_name} from "@/blogRoutes";
import {hasLength} from "@/utils";

let unsubscribe;

export default {
    data: () => ({
        showSearchButton: true,
    }),
    methods: {
        resetInput() {
            this.searchString = null;
            this.showSearchButton = true;
        },
        getBreadcrumbs() {
            const ret = [
                {
                    text: 'Videochat',
                    disabled: false,
                    href: '/',
                },
                {
                    text: 'Blog',
                    disabled: false,
                    exactPath: true,
                    to: blog,
                },
            ];
            if (this.$route.name == blog_post_name) {
                ret.push(
                    {
                        text: 'Post',
                        disabled: false,
                        to: '/blog/post',
                    },
                )
            }
            return ret
        },
        onOpenSearch() {
            this.showSearchButton = false;
        },
    },
    computed: {
        searchString: {
            get() {
                return this.$store.getters[GET_SEARCH_STRING];
            },
            set(newVal) {
                this.$store.commit(SET_SEARCH_STRING, newVal);
                return newVal;
            }
        },
        ...mapGetters({
            searchName: GET_SEARCH_NAME,
            isShowSearch: GET_SHOW_SEARCH,
        }),
        hasSearchString() {
            return hasLength(this.searchString)
        }
    },
    created() {
        unsubscribe = this.$store.subscribe((mutation, state) => {
            if (mutation.type == SET_SEARCH_STRING) {
                bus.$emit(SEARCH_STRING_CHANGED, mutation.payload);
                if (!hasLength(mutation.payload)) {
                    this.showSearchButton = true;
                }
            }
        });
    },
    beforeDestroy() {
        unsubscribe();
    }
}
</script>

<style lang="stylus">
    .v-breadcrumbs {
        li > a {
            color white
        }
    }
</style>
