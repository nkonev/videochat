<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" :persistent="isNew">
            <v-card>
                <v-card-title v-if="!isNew">{{ $vuetify.lang.t('$vuetify.edit_chat') }} #{{editChatId}}</v-card-title>
                <v-card-title v-else>{{ $vuetify.lang.t('$vuetify.create_chat') }}</v-card-title>

                <v-container fluid class="pb-0">
                    <v-form
                        ref="form"
                        v-model="valid"
                        lazy-validation
                        @keydown.native.enter.prevent="saveChat"
                    >
                        <v-text-field
                            id="new-chat-text"
                            :label="$vuetify.lang.t('$vuetify.chat_name')"
                            v-model="editDto.name"
                            required
                            :rules="chatNameRules"
                        ></v-text-field>
                        <v-autocomplete
                                v-model="editDto.participantIds"
                                :disabled="isLoading"
                                :items="people"
                                filled
                                chips
                                color="blue-grey lighten-2"
                                :label="$vuetify.lang.t('$vuetify.select_users_to_add_to_chat')"
                                item-text="login"
                                item-value="id"
                                multiple
                                :hide-selected="true"
                                hide-details
                                :search-input.sync="search"
                                dense
                                outlined
                        >
                            <template v-slot:selection="data">
                                <v-chip
                                        v-bind="data.attrs"
                                        :input-value="data.selected"
                                        close
                                        small
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

                        <v-checkbox
                            v-model="editDto.canResend"
                            :label="$vuetify.lang.t('$vuetify.can_resend')"
                            hide-details
                            dense
                        ></v-checkbox>

                        <v-checkbox
                            v-model="editDto.availableToSearch"
                            :label="$vuetify.lang.t('$vuetify.available_to_search')"
                            hide-details
                            dense
                        ></v-checkbox>

                        <template v-if="!isNew">
                            <v-container class="pb-0 px-0 pt-1">
                                <v-img v-if="editDto.avatarBig || editDto.avatar"
                                       :src="ava"
                                       :aspect-ratio="16/9"
                                       min-width="200"
                                       min-height="200"
                                       @click="openAvatarDialog"
                                >
                                </v-img>
                                <v-btn v-else color="primary" @click="openAvatarDialog()">{{ $vuetify.lang.t('$vuetify.choose_avatar_btn') }}</v-btn>
                            </v-container>
                        </template>
                    </v-form>
                </v-container>

                <v-card-actions class="pa-4">
                    <template>
                        <v-btn color="primary" class="mr-4" @click="saveChat" id="chat-save-btn">{{ $vuetify.lang.t('$vuetify.ok') }}</v-btn>
                    </template>
                    <v-btn color="error" class="mr-4" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";
    import debounce from "lodash/debounce";
    import bus, {OPEN_CHAT_EDIT, OPEN_CHOOSE_AVATAR} from "./bus";
    import {chat_name} from "@/routes";

    const dtoFactory = ()=>{
        return {
            id: null,
            name: "",
            participantIds: [ ],
            canResend: false,
            availableToSearch: true // it's default for all the new chats, excluding tet-a-tet
        }
    };

    export default {
        data () {
            const requiredMessage = this.$vuetify.lang.t('$vuetify.chat_name_required');
            return {
                show: false,
                editChatId: null,
                search: null,
                editDto: dtoFactory(),
                isLoading: false,
                people: [  ], // available person to chat with
                chatNameRules: [
                    v => !!v || requiredMessage,
                ],
                valid: true
            }
        },
        computed: {
            isNew() {
                return !this.editChatId;
            },
            ava() {
                if (this.editDto.avatarBig) {
                    return this.editDto.avatarBig
                } else if (this.editDto.avatar) {
                    return this.editDto.avatar
                } else {
                    return null
                }
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
                this.editChatId = chatId;
                if (this.editChatId) {
                    this.loadData();
                } else {
                    this.editDto = dtoFactory();
                }

            },
            loadData() {
                console.log("Getting info about chat id", this.editChatId);
                axios.get('/api/chat/'+this.editChatId)
                    .then((response) => {
                        this.editDto = {
                            id: response.data.id,
                            name: response.data.name,
                            avatar: response.data.avatar,
                            avatarBig: response.data.avatarBig,
                            canResend: response.data.canResend,
                            availableToSearch: response.data.availableToSearch,
                        };
                    })
            },
            removeSelected(item) {
                const index = this.editDto.participantIds.indexOf(item.id);
                if (index >= 0) this.editDto.participantIds.splice(index, 1)
            },
            doSearch(searchString) {
                if (this.isLoading) return;

                if (!searchString) {
                    return;
                }

                this.isLoading = true;

                if (this.isNew) {
                    axios.get(`/api/user?searchString=${searchString}`)
                        .then((response) => {
                            console.log("Fetched users", response.data.data);
                            this.people = [...this.people, ...response.data.data];
                        })
                        .finally(() => (this.isLoading = false))
                } else {
                    axios.get(`/api/chat/${this.editChatId}/user-candidate?searchString=${searchString}`)
                        .then((response) => {
                            console.log("Fetched users", response.data);
                            this.people = [...this.people, ...response.data];
                        })
                        .finally(() => (this.isLoading = false))
                }
            },
            saveChat() {
                const valid = this.validate();
                if (valid) {
                    const dtoToPost = {
                        id: this.editDto.id,
                        name: this.editDto.name,
                        participantIds: this.isNew ? this.editDto.participantIds : null,
                        avatar: this.editDto.avatar,
                        avatarBig: this.editDto.avatarBig,
                        canResend: this.editDto.canResend,
                        availableToSearch: this.editDto.availableToSearch,
                    };

                    if (this.isNew) {
                        axios.post(`/api/chat`, dtoToPost).then(({data}) => {
                            const routeDto = { name: chat_name, params: { id: data.id }};
                            this.$router.push(routeDto);
                        }).then(()=>this.closeModal());
                    } else {
                        axios.put(`/api/chat`, dtoToPost).then(()=>{
                            if (this.editDto.participantIds && this.editDto.participantIds.length) {
                                // we firstly add users...
                                return axios.put(`/api/chat/${this.editChatId}/user`, {
                                    addParticipantIds: this.editDto.participantIds
                                })
                            } else {
                                return Promise.resolve()
                            }
                        }).then(()=>this.closeModal());
                    }
                }
            },
            validate () {
                return this.$refs.form.validate()
            },
            closeModal() {
                console.debug("Closing ChatEditModal");
                this.show = false;
                // this.editChatId = null;
                this.search = null;
                this.editDto = dtoFactory();
                this.isLoading = false;
                this.people = [  ];
                this.valid = true;
            },
            openAvatarDialog() {
                bus.$emit(OPEN_CHOOSE_AVATAR, {
                    initialAvatarCallback: () => {
                        return this.ava;
                    },
                    uploadAvatarFileCallback: (blob) => {
                        if (!blob) {
                            return Promise.resolve(false);
                        }
                        const config = {
                            headers: { 'content-type': 'multipart/form-data' }
                        }
                        const formData = new FormData();
                        formData.append('data', blob);
                        return axios.post(`/api/storage/chat/${this.editDto.id}/avatar`, formData, config)
                    },
                    removeAvatarUrlCallback: () => {
                        this.editDto.avatar = null;
                        this.editDto.avatarBig = null;
                        return axios.put(`/api/chat`, this.editDto);
                    },
                    storeAvatarUrlCallback: (res) => {
                        this.editDto.avatar = res.data.relativeUrl;
                        this.editDto.avatarBig = res.data.relativeBigUrl;
                        return axios.put(`/api/chat`, this.editDto);
                    },
                    onSuccessCallback: () => {
                        this.loadData();
                    }
                });
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
