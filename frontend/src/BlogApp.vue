<template>
    <v-app>
        <v-app-bar app color='indigo' dark dense>
            <v-breadcrumbs
                :items="items"
            >
            </v-breadcrumbs>

            <v-spacer></v-spacer>
            <v-card light :width="isMobile() ? '100%' : ''">
                <v-text-field :autofocus="isMobile()" prepend-icon="mdi-magnify" hide-details single-line clearable clear-icon="mdi-close-circle"></v-text-field>
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
                href: '/blog',
            },
            {
                text: 'nkonev',
                disabled: false,
                href: '/blog/post',
            },
        ],
    }),
}
</script>

<style lang="stylus">
    .v-breadcrumbs {
        li > a {
            color white
        }
    }
</style>
