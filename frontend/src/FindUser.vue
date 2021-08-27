<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" @click:outside="closeModal()">
            <v-card>
                <v-card-title>Find user</v-card-title>

                <v-card-text class="px-4 py-0">
                    <v-autocomplete
                        :disabled="isLoading"
                        :items="people"
                        filled
                        color="blue-grey lighten-2"
                        label="Type to search"
                        item-text="login"
                        item-value="id"
                        :hide-selected="true"
                        hide-details
                        :search-input.sync="search"
                        dense
                        outlined
                        autofocus
                    >
                        <template v-slot:item="data">
                            <v-list-item-avatar v-if="data.item.avatar" @click="onUserClicked(data.item)">
                                <img :src="data.item.avatar">
                            </v-list-item-avatar>
                            <v-list-item-content @click="onUserClicked(data.item)">
                                <v-list-item-title v-html="data.item.login"></v-list-item-title>
                            </v-list-item-content>
                        </template>
                    </v-autocomplete>
                </v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn color="error" class="mr-4" @click="closeModal()">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";
    import debounce from "lodash/debounce";
    import bus, {OPEN_FIND_USER} from "./bus";
    import {profile_name} from "./routes";

    export default {
        data () {
            return {
                show: false,
                search: null,
                isLoading: false,
                people: [  ], // available person to chat with
            }
        },
        watch: {
            search (searchString) {
                this.doSearch(searchString);
            },
        },
        methods: {
            showModal() {
                this.$data.show = true;
            },
            doSearch(searchString) {
                if (this.isLoading) return;

                if (!searchString) {
                    return;
                }

                this.isLoading = true;

                axios.get(`/api/user?searchString=${searchString}`)
                    .then((response) => {
                        console.log("Fetched users", response.data.data);
                        this.people = [...this.people, ...response.data.data];
                    })
                    .finally(() => (this.isLoading = false))
            },
            onUserClicked(item) {
                console.log("onUserClicked", item);
                this.$router.push(({ name: profile_name, params: { id: item.id}}));
                this.closeModal();
            },
            closeModal() {
                console.debug("Closing FindUser");
                this.show = false;
                this.search = null;
                this.isLoading = false;
                this.people = [  ];
            }
        },
        created() {
            // https://forum-archive.vuejs.org/topic/5174/debounce-replacement-in-vue-2-0
            this.doSearch = debounce(this.doSearch, 700);
            bus.$on(OPEN_FIND_USER, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_FIND_USER, this.showModal);
        },
    }
</script>