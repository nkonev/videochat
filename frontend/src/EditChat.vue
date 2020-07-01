<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title>Create chat</v-card-title>

                <v-container fluid>
                <v-autocomplete
                        v-model="participants"
                        :disabled="isLoading"
                        :items="people"
                        filled
                        chips
                        color="blue-grey lighten-2"
                        label="Select users for add to chat"
                        item-text="login"
                        item-value="id"
                        multiple
                        :hide-selected="true"
                        :search-input.sync="search"
                >
                    <template v-slot:selection="data">
                        <v-chip
                                v-bind="data.attrs"
                                :input-value="data.selected"
                                close
                                @click="data.select"
                                @click:close="removeSelected(data.item)"
                        >
                            <v-avatar left v-if="data.item.avatar">
                                <v-img :src="data.item.avatar"></v-img>
                            </v-avatar>
                            {{ data.item.login }}
                        </v-chip>
                    </template>
                    <template v-slot:item="data">
                        <v-list-item-avatar v-if="data.item.avatar">
                            <img :src="data.item.avatar">
                        </v-list-item-avatar>
                        <v-list-item-content>
                            <v-list-item-title v-html="data.item.login"></v-list-item-title>
                        </v-list-item-content>
                    </template>
                </v-autocomplete>
                </v-container>

                <v-card-actions class="pa-4">
                    <v-btn color="primary" class="mr-4" @click="show=false">Create</v-btn>
                    <v-btn color="error" class="mr-4" @click="show=false">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";
    import debounce from "lodash/debounce";

    export default {
        props: {
            value: Boolean
        },
        computed: {
            show: {
                get() {
                    return this.value
                },
                set(value) {
                    this.$emit('input', value)
                }
            }
        },
        data () {
            const srcs = {
                1: 'https://cdn.vuetifyjs.com/images/lists/1.jpg',
                2: 'https://cdn.vuetifyjs.com/images/lists/2.jpg',
                3: 'https://cdn.vuetifyjs.com/images/lists/3.jpg',
                4: 'https://cdn.vuetifyjs.com/images/lists/4.jpg',
                5: 'https://cdn.vuetifyjs.com/images/lists/5.jpg',
            };

            return {
                search: null,
                participants: [ ],
                isLoading: false,
                people: [  ],
            }
        },


        watch: {
            search (searchString) {
                this.doSearch(searchString);
            },

        },

        methods: {
            removeSelected (item) {
                console.log("Removing", item, this.participants);
                const index = this.participants.indexOf(item.id);
                if (index >= 0) this.participants.splice(index, 1)
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
            }
        },
        created() {
            // https://forum-archive.vuejs.org/topic/5174/debounce-replacement-in-vue-2-0
            this.doSearch = debounce(this.doSearch, 700);
        },

    }
</script>