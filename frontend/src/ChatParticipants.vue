<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable @click:outside="closeModal()">
            <v-card>
                <v-card-title class="pl-0 ml-0">
                  <v-btn class="mx-2" icon @click="closeModal()"><v-icon>mdi-arrow-left</v-icon></v-btn>
                  Participants
                  <v-autocomplete
                      class="ml-4"
                      v-if="dto.canEdit"
                      v-model="newParticipantIds"
                      :disabled="newParticipantIdsIsLoading"
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
                      dense
                      outlined
                      autofocus
                  >
                    <template v-slot:selection="data">
                      <v-chip
                          v-bind="data.attrs"
                          :input-value="data.selected"
                          close
                          small
                          @click="data.select"
                          @click:close="removeNewSelected(data.item)"
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
                  <v-btn v-if="dto.canEdit" :disabled="newParticipantIds.length == 0" color="primary" class="ma-2 ml-4" @click="addSelectedParticipants()">Add</v-btn>
                </v-card-title>

                <v-card-text  class="ma-0 pa-0">
                    <v-list v-if="dto.participants.length > 0">
                        <template v-for="(item, index) in dto.participants">
                            <v-list-item class="pl-2 ml-1 pr-0 mr-1 mb-1 mt-1">
                                <v-badge
                                    v-if="item.avatar"
                                    color="success accent-4"
                                    dot
                                    bottom
                                    overlap
                                    bordered
                                    :value="item.online"
                                >
                                    <router-link :to="{ name: 'profileUser', params: { id: item.id }}">
                                        <v-list-item-avatar class="ma-0 pa-0">
                                            <v-img :src="item.avatar"></v-img>
                                        </v-list-item-avatar>
                                    </router-link>
                                </v-badge>
                                <v-list-item-content class="ml-4">
                                    <v-list-item-title>{{item.login}}<template v-if="item.id == currentUser.id"> (you)</template></v-list-item-title>
                                </v-list-item-content>
                                <v-tooltip bottom v-if="item.admin || dto.canChangeChatAdmins">
                                    <template v-slot:activator="{ on, attrs }">
                                        <template v-if="dto.canChangeChatAdmins && (item.id != currentUser.id)">
                                            <v-btn
                                                v-bind="attrs" v-on="on"
                                                :color="item.admin ? 'primary' : 'disabled'"
                                                :loading="item.adminLoading ? true : false"
                                                @click="changeChatAdmin(item)"
                                                icon
                                            >
                                                <v-icon>mdi-crown</v-icon>
                                            </v-btn>
                                        </template>
                                        <template v-else-if="item.admin">
                                          <span class="pl-1 pr-1">
                                              <v-icon v-bind="attrs" v-on="on">mdi-crown</v-icon>
                                          </span>
                                        </template>
                                    </template>
                                    <span>Admin</span>
                                </v-tooltip>

                                <v-btn v-if="dto.canEdit && item.id != currentUser.id" icon @click="deleteParticipant(item)" color="error"><v-icon dark>mdi-delete</v-icon></v-btn>
                                <v-tooltip bottom v-if="dto.canVideoKick && item.id != currentUser.id">
                                    <template v-slot:activator="{ on, attrs }">
                                        <v-btn v-bind="attrs" v-on="on" icon @click="kickFromVideoCall(item.id)"><v-icon color="error">mdi-block-helper</v-icon></v-btn>
                                    </template>
                                    <span>Kick</span>
                                </v-tooltip>
                                <v-btn v-if="item.id != currentUser.id" icon @click="inviteToVideoCall(item.id)"><v-icon color="success">mdi-phone</v-icon></v-btn>
                            </v-list-item>
                            <v-divider></v-divider>
                        </template>
                    </v-list>
                    <v-progress-circular
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                </v-card-text>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";
    import bus, {
        CHAT_DELETED,
        CHAT_EDITED,
        CLOSE_SIMPLE_MODAL,
        OPEN_PARTICIPANTS_DIALOG,
        OPEN_SIMPLE_MODAL,
    } from "./bus";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import {chat_name, videochat_name} from "./routes";
    import debounce from "lodash/debounce";
    import userOnlinePollingMixin from "./userOnlinePollingMixin";

    const dtoFactory = ()=>{
        return {
            id: null,
            name: "",
            participantIds: [ ],
            participants: [ ],
        }
    };

    export default {
        mixins: [userOnlinePollingMixin()],
        data () {
            return {
                show: false,
                dto: dtoFactory(),
                chatId: null,

                newParticipantIdsIsLoading: false,
                newParticipantIds: [],
                people: [  ], // available person to chat with
                search: null,
            }
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        },

        methods: {
            showModal(chatId) {
                this.chatId = chatId;

                this.show = true;
                if (this.chatId && this.show) {
                    this.loadData();
                } else {
                    this.dto = dtoFactory();
                }
            },
            transformParticipants(tmp) {
                tmp.participants.forEach(item => {
                    item.adminLoading = false;
                    item.online = false;
                });
            },
            loadData() {
                console.log("Getting info about chat id in modal, chatId=", this.chatId);
                axios.get('/api/chat/' + this.chatId)
                    .then((response) => {
                        const tmp = response.data;
                        this.transformParticipants(tmp);
                        this.dto = tmp;
                    }).then(value => {
                        this.startPolling(
                            ()=>{ return this.dto.participantIds},
                            (v) => this.onUserOnlineChanged(v)
                        );
                    })
            },
            changeChatAdmin(item) {
                item.adminLoading = true;
                this.$forceUpdate();
                axios.put(`/api/chat/${this.dto.id}/user/${item.id}?admin=${!item.admin}`);
            },
            inviteToVideoCall(userId) {
                axios.put(`/api/chat/${this.dto.id}/video/invite?userId=${userId}`).then(value => {
                    console.log("Inviting to video chat");
                    if (this.$route.name != videochat_name) {
                        this.$router.push({name: videochat_name});
                    }
                })
            },
            kickFromVideoCall(userId) {
                axios.put(`/api/chat/${this.dto.id}/video/kick?userId=${userId}`)
            },
            onChatChange(dto) {
                if (this.show && dto.id == this.chatId) {
                    this.dto = dtoFactory();
                    this.$nextTick(()=> {
                        const tmp = dto;
                        this.transformParticipants(tmp);
                        this.dto = tmp;
                    });
                }
            },
            deleteParticipant(participant) {
                bus.$emit(OPEN_SIMPLE_MODAL, {
                    buttonName: 'Remove',
                    title: `Remove participant #${participant.id}`,
                    text: `Are you sure to remove user #${participant.id} '${participant.login}' from this chat ?`,
                    actionFunction: ()=> {
                        axios.delete(`/api/chat/${this.dto.id}/user/${participant.id}`)
                            .then(() => {
                                bus.$emit(CLOSE_SIMPLE_MODAL);
                            })
                    }
                });
            },
            closeModal() {
                console.debug("Closing ChatParticipants");
                this.show = false;
                this.chatId = null;
                this.newParticipantIds = [];
                this.people = [];
                this.dto = dtoFactory();
                this.newParticipantIdsIsLoading = false;
                this.search = null;
                this.stopPolling();
            },
            removeNewSelected (item) {
                console.debug("Removing", item, this.newParticipantIds);
                const index = this.newParticipantIds.indexOf(item.id);
                if (index >= 0) this.newParticipantIds.splice(index, 1)
            },
            doNewSearch(searchString) {
                if (this.newParticipantIdsIsLoading) return;

                if (!searchString) {
                    return;
                }

                this.newParticipantIdsIsLoading = true;

                axios.get(`/api/user?searchString=${searchString}`)
                    .then((response) => {
                      console.log("Fetched users", response.data.data);
                      this.people = [...this.people, ...response.data.data].filter(value => !this.dto.participantIds.includes(value.id));
                    })
                    .finally(() => (this.newParticipantIdsIsLoading = false))
            },
            addSelectedParticipants() {
                axios.put(`/api/chat/${this.dto.id}/users`, {
                  addParticipantIds: this.newParticipantIds
                }).then(value => {
                    this.newParticipantIds = [];
                    this.search = null;
                })
            },
            onChatDelete(dto) {
                if (dto.id == this.chatId) {
                    this.closeModal();
                }
            },
            onUserOnlineChanged(dtos) {
                this.dto.participants.forEach(item => {
                    dtos.forEach(dtoItem => {
                        if (dtoItem.userId == item.id) {
                            item.online = dtoItem.online;
                        }
                    })
                })
                this.$forceUpdate()
            }
        },
        watch: {
            search (searchString) {
              this.doNewSearch(searchString);
            },
            '$route' (to, from) {
                console.debug("Listening route from", from, "to", to);
                if (from.name == chat_name || from.name == videochat_name) {
                    this.closeModal();
                }
            }
        },

        created() {
            this.doNewSearch = debounce(this.doNewSearch, 700);
            bus.$on(OPEN_PARTICIPANTS_DIALOG, this.showModal);
            bus.$on(CHAT_EDITED, this.onChatChange);
            bus.$on(CHAT_DELETED, this.onChatDelete);
        },
        destroyed() {
            bus.$off(OPEN_PARTICIPANTS_DIALOG, this.showModal);
            bus.$off(CHAT_EDITED, this.onChatChange);
            bus.$off(CHAT_DELETED, this.onChatDelete);
        },
    }
</script>
