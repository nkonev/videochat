<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640">
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.find_user') }}</v-card-title>

                <v-card-text class="px-4 py-0">
                    <v-autocomplete
                        :disabled="isLoading"
                        :items="people"
                        filled
                        color="blue-grey lighten-2"
                        :label="$vuetify.lang.t('$vuetify.type_to_find_user')"
                        item-text="login"
                        item-value="id"
                        hide-details
                        :search-input.sync="search"
                        dense
                        outlined
                        autofocus
                    >
                        <template v-slot:item="data">
                            <v-list-item @click="onUserClicked(data.item)">
                                <v-list-item-avatar v-if="data.item.avatar">
                                    <img :src="data.item.avatar">
                                </v-list-item-avatar>
                                <v-list-item-content>
                                    <v-list-item-title v-html="data.item.login"></v-list-item-title>
                                </v-list-item-content>
                            </v-list-item>
                        </template>
                    </v-autocomplete>
                </v-card-text>

                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn color="error" class="my-1" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
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
            show(newValue) {
                if (!newValue) {
                    this.closeModal();
                }
            }
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

                axios.post(`/api/user/search`, {
                    searchString: searchString
                })
                    .then((response) => {
                        const users = response.data.users;
                        console.log("Fetched users", users);
                        this.people = [...this.people, ...users];
                    })
                    .finally(() => {
                        this.isLoading = false;
                    })
            },
            onUserClicked(item) {
                this.$router.push(({ name: profile_name, params: { id: item.id}})).then(()=>this.closeModal());
            },
            closeModal() {
                console.debug("Closing FindUserModal");
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
