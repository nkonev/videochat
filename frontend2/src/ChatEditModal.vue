<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" :persistent="isNew" scrollable>
            <v-card :title="getTitle()">
                <v-card-text class="pb-0">
                    <v-form
                        ref="form"
                        v-model="valid"
                        lazy-validation
                        @keydown.native.enter.prevent="saveChat"
                    >
                        <v-text-field
                            id="new-chat-text"
                            :label="$vuetify.locale.t('$vuetify.chat_name')"
                            v-model="editDto.name"
                            variant="outlined"
                            :rules="chatNameRules"
                        ></v-text-field>
                        <v-autocomplete
                                v-model="editDto.participantIds"
                                :loading="isLoading"
                                :items="people"
                                chips
                                closable-chips
                                color="blue-grey lighten-2"
                                :label="$vuetify.locale.t('$vuetify.select_users_to_add_to_chat')"
                                item-title="login"
                                item-value="id"
                                multiple
                                :hide-selected="true"
                                hide-details
                                @update:search="onUpdateSearch"
                                density="compact"
                                variant="outlined"
                        >
                            <template v-slot:chip="{ props, item }">
                                <v-chip
                                    v-bind="props"
                                    :prepend-avatar="item.raw.avatar"
                                    :text="item.raw.login"
                                ></v-chip>
                            </template>

                            <template v-slot:item="{ props, item }">
                                <v-list-item
                                    v-bind="props"
                                    :prepend-avatar="item?.raw?.avatar"
                                    :title="item?.raw?.login"
                                ></v-list-item>
                            </template>
                        </v-autocomplete>

                        <v-checkbox
                            v-model="editDto.canResend"
                            :label="$vuetify.locale.t('$vuetify.can_resend')"
                            hide-details
                            density="compact"
                        ></v-checkbox>

                        <v-checkbox
                            v-model="editDto.availableToSearch"
                            :label="$vuetify.locale.t('$vuetify.available_to_search')"
                            hide-details
                            density="compact"
                        ></v-checkbox>

                        <v-checkbox
                            v-model="editDto.blog"
                            :label="$vuetify.locale.t('$vuetify.blog')"
                            hide-details
                            density="compact"
                        ></v-checkbox>

                        <template v-if="!isNew">
                            <v-container class="pa-0 ma-0 mt-2">
                                <v-img v-if="hasAva"
                                       :src="ava"
                                       :aspect-ratio="16/9"
                                       min-width="600"
                                       min-height="600"
                                       max-height="800"
                                       @click="openAvatarDialog"
                                >
                                </v-img>
                            </v-container>
                        </template>
                    </v-form>
                </v-card-text>

                <v-card-actions>
                    <v-btn v-if="!hasAva" variant="outlined" @click="openAvatarDialog()"><v-icon>mdi-image-outline</v-icon> {{ $vuetify.locale.t('$vuetify.choose_avatar_btn') }}</v-btn>
                    <v-spacer></v-spacer>
                    <v-btn color="primary" variant="flat" @click="saveChat" id="chat-save-btn">{{ $vuetify.locale.t('$vuetify.ok') }}</v-btn>
                    <v-btn color="red" variant="flat" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";
    import debounce from "lodash/debounce";
    import bus, {OPEN_CHAT_EDIT, OPEN_CHOOSE_AVATAR} from "./bus/bus";
    import {chat_name} from "@/router/routes";
    import {hasLength} from "@/utils";

    const dtoFactory = ()=>{
        return {
            id: null,
            name: "",
            participantIds: [ ],
            canResend: false,
            availableToSearch: false, // it's default for all the new chats, excluding tet-a-tet
        }
    };

    export default {
        data () {
            // const requiredMessage = this.$vuetify.locale.t('$vuetify.chat_name_required');
            return {
                show: false,
                editChatId: null,
                search: null,
                editDto: dtoFactory(),
                isLoading: false,
                people: [  ], // available person to chat with
                // chatNameRules: [
                //     v => !!v || requiredMessage,
                // ],
                valid: true,
            }
        },
        computed: {
            chatNameRules() {
                return [
                    v => !!v || this.$vuetify.locale.t('$vuetify.chat_name_required'),
                ]
            },
            isNew() {
                return !this.editChatId;
            },
            ava() {
                if (hasLength(this.editDto.avatarBig)) {
                    return this.editDto.avatarBig
                } else if (hasLength(this.editDto.avatar)) {
                    return this.editDto.avatar
                } else {
                    return null
                }
            },
            hasAva() {
                return hasLength(this.editDto.avatarBig) || hasLength(this.editDto.avatar)
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
                            blog: response.data.blog,
                        };
                    })
            },
            removeSelected(item) {
                const index = this.editDto.participantIds.indexOf(item.id);
                if (index >= 0) this.editDto.participantIds.splice(index, 1)
            },
            onUpdateSearch(value) {
                console.log("on update search", value)
                this.doSearch(value);
            },
            doSearch(searchString) {
                if (this.isLoading) return;

                if (!searchString) {
                    return;
                }

                this.isLoading = true;

                if (this.isNew) {
                    axios.post(`/api/user/search`, {
                        searchString: searchString
                    })
                        .then((response) => {
                            console.log("Fetched users", response.data.users);
                            this.people = response.data.users;
                        })
                        .finally(() => {
                            this.isLoading = false;
                        })
                } else {
                    axios.get(`/api/chat/${this.editChatId}/user-candidate?searchString=${searchString}`)
                        .then((response) => {
                            console.log("Fetched users", response.data);
                            this.people = response.data;
                        })
                        .finally(() => {
                            this.isLoading = false;
                        })
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
                        blog: this.editDto.blog,
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
                bus.emit(OPEN_CHOOSE_AVATAR, {
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
            getTitle() {
                if (!this.isNew) {
                    return this.$vuetify.locale.t('$vuetify.edit_chat') + " #" + this.editChatId;
                } else {
                    return this.$vuetify.locale.t('$vuetify.create_chat')
                }
            },
        },
        created() {
            // https://forum-archive.vuejs.org/topic/5174/debounce-replacement-in-vue-2-0
            this.doSearch = debounce(this.doSearch, 700);
            bus.on(OPEN_CHAT_EDIT, this.showModal);
        },
        destroyed() {
            bus.off(OPEN_CHAT_EDIT, this.showModal);
        },
    }
</script>
