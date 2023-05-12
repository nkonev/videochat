<template>
    <v-app>
        <v-app-bar app color='indigo' dark dense>
            <v-breadcrumbs
                :items="items"
            >
            </v-breadcrumbs>

            <v-spacer></v-spacer>
            <v-card v-if="isShowSearch" light :width="isMobile() ? '100%' : ''">
                <v-text-field prepend-icon="mdi-magnify" hide-details single-line v-model="searchString" :label="searchName" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput"></v-text-field>
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
import {blog} from "@/blogRoutes";

let unsubscribe;

export default {
    data: () => ({
        items: [
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
            }
        ],
    }),
    methods: {
        resetInput() {
            this.searchString = null;
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
    },
    created() {
        unsubscribe = this.$store.subscribe((mutation, state) => {
            if (mutation.type == SET_SEARCH_STRING) {
                bus.$emit(SEARCH_STRING_CHANGED, mutation.payload);
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
