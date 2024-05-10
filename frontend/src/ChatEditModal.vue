<template>
    <v-row justify="center">
        <input id="image-input-chat-avatar" type="file" style="display: none;" accept="image/*"/>

        <v-dialog v-model="show" max-width="640" :persistent="isNew" scrollable>
            <v-card :title="getTitle()">
                <v-card-text class="pb-0">
                    <v-form
                        v-if="!loading"
                        ref="form"
                        v-model="valid"
                        lazy-validation
                        @keydown.native.enter.prevent="saveChat"
                    >
                        <v-text-field
                            id="test-chat-text"
                            :label="$vuetify.locale.t('$vuetify.chat_name')"
                            v-model="editDto.name"
                            variant="outlined"
                            density="compact"
                            :rules="chatNameRules"
                            class="mt-2"
                        ></v-text-field>
                        <v-autocomplete
                                v-model="editDto.participantIds"
                                :loading="searchLoading"
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
                            color="primary"
                        ></v-checkbox>

                        <v-checkbox
                            v-model="editDto.availableToSearch"
                            :label="$vuetify.locale.t('$vuetify.available_to_search')"
                            hide-details
                            density="compact"
                            color="primary"
                        ></v-checkbox>

                        <v-checkbox
                            v-if="canCreateBlog"
                            v-model="editDto.blog"
                            :label="$vuetify.locale.t('$vuetify.blog')"
                            hide-details
                            density="compact"
                            color="primary"
                        ></v-checkbox>

                        <template v-if="!isNew">
                            <v-container class="pa-0 ma-0 mt-2">
                                <v-img v-if="hasAva"
                                       :src="ava"
                                       :aspect-ratio="16/9"
                                       :min-width="isMobile() ? null : 600"
                                       :min-height="isMobile() ? null : 600"
                                       :max-height="isMobile() ? null : 800"
                                >
                                </v-img>
                            </v-container>
                        </template>
                    </v-form>

                    <v-progress-circular
                      class="ma-4"
                      v-else
                      indeterminate
                      color="primary"
                    ></v-progress-circular>
                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">
                    <template v-if="!isNew && !loading">
                      <v-btn v-if="hasAva" variant="outlined" @click="removeAvatarFromChat()">
                        <template v-slot:prepend>
                          <v-icon>mdi-image-remove</v-icon>
                        </template>
                        <template v-slot:default>
                          {{ $vuetify.locale.t('$vuetify.remove_avatar_btn') }}
                        </template>
                      </v-btn>
                      <v-btn v-if="!hasAva" variant="outlined" @click="openAvatarDialog()">
                        <template v-slot:prepend>
                          <v-icon>mdi-image-outline</v-icon>
                        </template>
                        <template v-slot:default>
                          {{ $vuetify.locale.t('$vuetify.choose_avatar_btn') }}
                        </template>
                      </v-btn>
                    </template>
                    <v-spacer></v-spacer>
                    <div :class="isMobile() ? 'mt-2' : ''">
                      <v-btn v-if="!loading" color="primary" variant="flat" @click="saveChat" id="test-chat-save-btn">{{ $vuetify.locale.t('$vuetify.ok') }}</v-btn>
                      <v-btn color="red" variant="flat" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                    </div>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";
    import debounce from "lodash/debounce";
    import bus, {OPEN_CHAT_EDIT} from "./bus/bus";
    import {chat_name} from "@/router/routes";
    import {hasLength} from "@/utils";
    import {v4 as uuidv4} from "uuid";
    import {isNumber, isObject, isString} from "lodash";

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
                search: null,
                editDto: dtoFactory(),
                searchLoading: false,
                people: [  ], // available person to chat with
                valid: true,
                fileInput: null,
                canCreateBlog: false,
                loading: false,
            }
        },
        computed: {
            chatNameRules() {
                return [
                    v => !!v || this.$vuetify.locale.t('$vuetify.chat_name_required'),
                ]
            },
            isNew() {
                return !this.editDto.id;
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
            showModal(input) {
                this.$data.show = true;

                if (isNumber(input) || isString(input)) {
                  this.loadData(input).then((data) => {
                    this.editDto = this.extractNecessaryFields(data);
                  })
                } else if (isObject(input)) {
                  this.editDto = this.extractNecessaryFields(input);
                } else {
                  this.editDto = dtoFactory()
                }

                this.loadCanCreateBlog();
            },
            extractNecessaryFields(chatDto) {
              return {
                id: chatDto.id,
                name: chatDto.name,
                avatar: chatDto.avatar,
                avatarBig: chatDto.avatarBig,
                canResend: chatDto.canResend,
                availableToSearch: chatDto.availableToSearch,
                blog: chatDto.blog,
              }
            },
            loadData(editChatId) {
              console.log("Getting info about chat id", editChatId);
              this.loading = true;
              return axios.get('/api/chat/'+editChatId)
                .then((response) => {
                  return response.data
                }).finally(()=>{
                  this.loading = false;
                })
            },

            loadCanCreateBlog() {
                axios.get("/api/chat/can-create-blog")
                    .then((response) => {
                        this.canCreateBlog = response.data.canCreateBlog;
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
                if (this.searchLoading) return;

                if (!searchString) {
                    return;
                }

                this.searchLoading = true;

                if (this.isNew) {
                    axios.post(`/api/aaa/user/search`, {
                        searchString: searchString
                    })
                        .then((response) => {
                            console.log("Fetched users", response.data);
                            this.people = response.data;
                        })
                        .finally(() => {
                            this.searchLoading = false;
                        })
                } else {
                    axios.get(`/api/chat/${this.editDto.id}/user-candidate?searchString=${searchString}`)
                        .then((response) => {
                            console.log("Fetched users", response.data);
                            this.people = response.data;
                        })
                        .finally(() => {
                            this.searchLoading = false;
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

                    this.loading = true;
                    if (this.isNew) {
                        axios.post(`/api/chat`, dtoToPost).then(({data}) => {
                            const routeDto = { name: chat_name, params: { id: data.id }};
                            this.$router.push(routeDto);
                        }).then(()=>this.closeModal()).finally(()=>{
                          this.loading = false;
                        });
                    } else {
                        axios.put(`/api/chat`, dtoToPost).then(()=>{
                            if (this.editDto.participantIds && this.editDto.participantIds.length) {
                                // we firstly add users...
                                return axios.put(`/api/chat/${this.editDto.id}/participant`, {
                                    addParticipantIds: this.editDto.participantIds
                                })
                            } else {
                                return Promise.resolve()
                            }
                        }).then(()=>this.closeModal()).finally(()=>{
                          this.loading = false;
                        });
                    }
                }
            },
            validate () {
                return this.$refs.form.validate()
            },
            closeModal() {
                console.debug("Closing ChatEditModal");
                this.show = false;
                this.search = null;
                this.editDto = dtoFactory();
                this.searchLoading = false;
                this.people = [  ];
                this.valid = true;
                this.canCreateBlog = false;
                this.loading = false;
            },
            openAvatarDialog() {
                this.fileInput.click();
            },
            getTitle() {
                if (!this.isNew) {
                    return this.$vuetify.locale.t('$vuetify.edit_chat') + " #" + this.editDto.id;
                } else {
                    return this.$vuetify.locale.t('$vuetify.create_chat')
                }
            },
            setAvatarToChat(file) {
              const config = {
                headers: { 'content-type': 'multipart/form-data' }
              }
              this.loading = true;
              const formData = new FormData();
              formData.append('data', file);
              return axios.post(`/api/storage/chat/${this.editDto.id}/avatar`, formData, config)
                .then((res) => {
                  this.editDto.avatar = res.data.relativeUrl;
                  this.editDto.avatarBig = res.data.relativeBigUrl;
                  return axios.put(`/api/chat`, this.editDto);
                }).finally(()=>{
                  this.loading = false;
                })
            },
            removeAvatarFromChat() {
                this.editDto.avatar = null;
                this.editDto.avatarBig = null;

                this.loading = true;
                return axios.put(`/api/chat`, this.editDto).finally(()=>{
                  this.loading = false;
                });
            },
        },
        created() {
            // https://forum-archive.vuejs.org/topic/5174/debounce-replacement-in-vue-2-0
            this.doSearch = debounce(this.doSearch, 700);
            bus.on(OPEN_CHAT_EDIT, this.showModal);
        },
        beforeUnmount() {
            if (this.fileInput) {
              this.fileInput.onchange = null;
            }
            this.fileInput = null;
            bus.off(OPEN_CHAT_EDIT, this.showModal);
        },
        mounted() {
          this.fileInput = document.getElementById('image-input-chat-avatar');
          this.fileInput.onchange = (e) => {
            if (e.target.files.length) {
              const files = Array.from(e.target.files);
              const file = files[0];
              this.setAvatarToChat(file);
            }
          }
        }
    }
</script>
