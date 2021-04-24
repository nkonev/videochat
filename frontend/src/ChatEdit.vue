<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title v-if="isEdit()">Edit chat #{{editChatId}}</v-card-title>
                <v-card-title v-else>Create chat</v-card-title>

                <v-container fluid>
                    <v-form
                        ref="form"
                        v-model="valid"
                        lazy-validation
                        @keyup.native.enter="saveChat"
                    >
                        <v-text-field
                            label="Chat name"
                            v-model="dto.name"
                            required
                            :rules="chatNameRules"
                        ></v-text-field>
                        <v-autocomplete
                                v-model="dto.participantIds"
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
                                hide-details
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
                    </v-form>
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
    import bus, {CHAT_ADD, CHAT_EDITED, OPEN_CHAT_EDIT} from "./bus";

    const dtoFactory = ()=>{
        return {
            id: null,
            name: "",
            participantIds: [ ],
        }
    };

    export default {
        data () {
            return {
                show: false,
                editChatId: null,
                search: null,
                dto: dtoFactory(),
                isLoading: false,
                people: [  ], // available person to chat with
                chatNameRules: [
                    v => !!v || 'Chat name is required',
                ],
                valid: true
            }
        },

        watch: {
            search (searchString) {
                this.doSearch(searchString);
            },
        },
        methods: {
            showModal(chatId) {
                this.$data.show = true;
                const val = chatId;
                this.editChatId = chatId;
                if (val) {
                    console.log("Getting info about chat id", val);
                    axios.get('/api/chat/'+val)
                        .then((response) => {
                            this.dto = response.data;
                        }).then(()=>{
                            axios.get('/api/user/list', {
                                params: {userId: [...this.dto.participantIds] + ''}
                            }).then((response) => {
                                this.people = response.data;
                            })
                        })
                } else {
                    this.dto = dtoFactory();
                }

            },
            removeSelected (item) {
                console.debug("Removing", item, this.dto.participantIds);
                const index = this.dto.participantIds.indexOf(item.id);
                if (index >= 0) this.dto.participantIds.splice(index, 1)
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
                const valid = this.validate();
                if (valid) {
                    const dtoToPost = this.dto;
                    dtoToPost.participants = null;
                    (dtoToPost.id ? axios.put(`/api/chat`, dtoToPost) : axios.post(`/api/chat`, dtoToPost))
                        .then(() => {
                            this.show = false;
                        })
                }
            },
            validate () {
                return this.$refs.form.validate()
            },

        },
        created() {
            // https://forum-archive.vuejs.org/topic/5174/debounce-replacement-in-vue-2-0
            this.doSearch = debounce(this.doSearch, 700);
            bus.$on(OPEN_CHAT_EDIT, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_CHAT_EDIT, this.showModal);
        },
    }
</script>