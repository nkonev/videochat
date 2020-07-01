<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title v-if="isEdit()">Edit chat #{{editChatId}}</v-card-title>
                <v-card-title v-else>Create chat</v-card-title>

                <v-container fluid>
                    <v-text-field label="Chat name"></v-text-field>
                <v-autocomplete
                        v-model="participantIds"
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
                    <template>
                        <v-btn color="primary" class="mr-4" @click="saveChat" v-if="isEdit()">Edit</v-btn>
                        <v-btn color="primary" class="mr-4" @click="saveChat" v-else>Create</v-btn>
                    </template>
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
    import bus, {CHAT_SAVED} from "./bus";

    export default {
        props: {
            value: Boolean,
            editChatId: Number,
            editParticipantIds: Array
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
            return {
                search: null,
                participantIds: [ ],
                isLoading: false,
                people: [  ],
            }
        },

        watch: {
            search (searchString) {
                this.doSearch(searchString);
            },
            editChatId(val) {
                if (val) {
                    console.log("Getting info about chat id", val)
                }
            },
            editParticipantIds(val) {
                console.log("on editParticipantIds", val);
                if (val.length) {
                    axios.get('/api/user/list', {
                        params: {userId: [...val] + ''}
                    }).then((response) => {
                        this.people = response.data;
                        this.participantIds = this.people.map(e => e.id);
                    })
                }
            },
            /*'editParticipantIds': {
                handler: function (val, oldVal) {
                    console.log("on editParticipantIds", val);
                    if (val.length) {
                        axios.get('/api/user/list', {
                            params: {userId: [...val] + ''}
                        }).then((response) => {
                            this.people = response.data;
                            this.participantIds = this.people.map(e => e.id);
                        })
                    }
                },
                deep: true
            }*/

        },

        methods: {
            removeSelected (item) {
                console.debug("Removing", item, this.participantIds);
                const index = this.participantIds.indexOf(item.id);
                if (index >= 0) this.participantIds.splice(index, 1)
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
            isEdit() {
                if (this.editChatId) {
                    return true
                } else {
                    return false
                }
            },
            saveChat() {
                const dtoToPost = {id: this.editChatId, participantIds: this.participantIds};
                (dtoToPost.id ? axios.put(`/api/chat`, dtoToPost) : axios.post(`/api/chat`, dtoToPost))
                    .then(() => {
                        bus.$emit(CHAT_SAVED, null);
                    })
                    .then(() => {
                        this.show=false;
                    })
            },
        },
        created() {
            // https://forum-archive.vuejs.org/topic/5174/debounce-replacement-in-vue-2-0
            this.doSearch = debounce(this.doSearch, 700);
        },
    }
</script>