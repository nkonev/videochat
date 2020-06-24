<template>
    <v-app>
        <v-main>
            <v-container>
                <v-card
                        max-width="500"
                        class="mx-auto"
                >
                    <v-toolbar
                            color="indigo"
                            dark
                    >
                        <v-app-bar-nav-icon></v-app-bar-nav-icon>

                        <v-toolbar-title>Inbox</v-toolbar-title>

                        <v-spacer></v-spacer>

                        <v-btn icon>
                            <v-icon>mdi-magnify</v-icon>
                        </v-btn>

                        <v-btn icon>
                            <v-icon>mdi-dots-vertical</v-icon>
                        </v-btn>
                    </v-toolbar>
                    <v-list>
                        <v-list-item
                                v-for="item in items"
                                :key="item.title"
                                @click=""
                        >
                            <v-list-item-icon>
                                <v-icon v-if="item.icon" color="pink">mdi-star</v-icon>
                            </v-list-item-icon>

                            <v-list-item-content>
                                <v-list-item-title v-text="item.title"></v-list-item-title>
                            </v-list-item-content>

                            <v-list-item-avatar>
                                <v-img :src="item.avatar"></v-img>
                            </v-list-item-avatar>
                        </v-list-item>
                    </v-list>
                </v-card>

            </v-container>
        </v-main>
    </v-app>
</template>

<script>
    import Vue from 'vue'

    export default {
        data () {
            return {
                items: [
                    {icon: true, title: 'Jason Oner', avatar: 'https://cdn.vuetifyjs.com/images/lists/1.jpg'},
                    {title: 'Travis Howard', avatar: 'https://cdn.vuetifyjs.com/images/lists/2.jpg'},
                    {title: 'Ali Connors', avatar: 'https://cdn.vuetifyjs.com/images/lists/3.jpg'},
                    {title: 'Cindy Baker', avatar: 'https://cdn.vuetifyjs.com/images/lists/4.jpg'},
                ],
            }
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
