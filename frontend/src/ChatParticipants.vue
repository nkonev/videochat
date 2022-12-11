<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="700" scrollable>
            <v-card>
                <v-card-title class="pl-0 ml-0">
                  <v-btn class="mx-2" icon @click="closeModal()"><v-icon>mdi-close</v-icon></v-btn>
                  {{ $vuetify.lang.t('$vuetify.participants_modal_title') }}
                  <v-autocomplete
                      class="ml-4"
                      v-if="dto.canEdit"
                      v-model="newParticipantIds"
                      :disabled="newParticipantIdsIsLoading"
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
                  <v-btn v-if="dto.canEdit" :disabled="newParticipantIds.length == 0" color="primary" class="ma-2 ml-4" @click="addSelectedParticipants()">
                      {{ $vuetify.lang.t('$vuetify.add') }}
                  </v-btn>
                </v-card-title>

                <v-card-text  class="ma-0 pa-0">
                    <v-pagination
                        v-if="shouldShowPagination"
                        v-model="participantsPage"
                        :length="participantsPagesCount"
                    ></v-pagination>

                    <v-list v-if="dto.participants && dto.participants.length > 0">
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
                                    <a @click.prevent="onParticipantClick(item)" :href="getLink(item)">
                                        <v-list-item-avatar class="ma-0 pa-0">
                                            <v-img :src="item.avatar"></v-img>
                                        </v-list-item-avatar>
                                    </a>
                                </v-badge>
                                <v-list-item-content class="ml-4">
                                    <v-row no-gutters align="center">
                                        <v-col>
                                            <v-list-item-title><a @click.prevent="onParticipantClick(item)" :href="getLink(item)">{{item.login + (item.id == currentUser.id ? $vuetify.lang.t('$vuetify.you_brackets') : '' )}}</a></v-list-item-title>
                                        </v-col>
                                        <v-col>
                                            <v-progress-linear
                                                v-if="item.callingTo"
                                                color="success"
                                                buffer-value="0"
                                                indeterminate
                                                rounded
                                                reverse
                                            ></v-progress-linear>
                                        </v-col>
                                    </v-row>
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
                                              <v-icon v-bind="attrs" v-on="on" color="primary">mdi-crown</v-icon>
                                          </span>
                                        </template>
                                    </template>
                                    <template v-if="dto.canChangeChatAdmins && (item.id != currentUser.id)">
                                        <span>{{ item.admin ? $vuetify.lang.t('$vuetify.revoke_admin') : $vuetify.lang.t('$vuetify.grant_admin') }}</span>
                                    </template>
                                    <template v-else-if="item.admin">
                                        <span>{{ $vuetify.lang.t('$vuetify.admin') }}</span>
                                    </template>
                                </v-tooltip>

                                <v-tooltip bottom v-if="dto.canEdit && item.id != currentUser.id">
                                    <template v-slot:activator="{ on, attrs }">
                                        <v-btn v-bind="attrs" v-on="on" icon @click="deleteParticipant(item)" color="error"><v-icon dark>mdi-delete</v-icon></v-btn>
                                    </template>
                                    <span>{{ $vuetify.lang.t('$vuetify.delete_from_chat') }}</span>
                                </v-tooltip>


                                <v-tooltip bottom v-if="dto.canVideoKick && item.id != currentUser.id">
                                    <template v-slot:activator="{ on, attrs }">
                                        <v-btn v-bind="attrs" v-on="on" icon @click="kickFromVideoCall(item.id)"><v-icon color="error">mdi-block-helper</v-icon></v-btn>
                                    </template>
                                    <span>{{ $vuetify.lang.t('$vuetify.kick') }}</span>
                                </v-tooltip>
                                <v-tooltip bottom v-if="dto.canAudioMute && item.id != currentUser.id">
                                    <template v-slot:activator="{ on, attrs }">
                                        <v-btn v-bind="attrs" v-on="on" icon @click="forceMute(item.id)"><v-icon color="error">mdi-microphone-off</v-icon></v-btn>
                                    </template>
                                    <span>{{ $vuetify.lang.t('$vuetify.force_mute') }}</span>
                                </v-tooltip>
                                <v-tooltip bottom v-if="item.id != currentUser.id">
                                    <template v-slot:activator="{ on, attrs }">
                                        <v-btn v-bind="attrs" v-on="on" icon @click="startCalling(item)"><v-icon :class="{'call-blink': item.callingTo}" color="success">mdi-phone</v-icon></v-btn>
                                    </template>
                                    <span>{{ item.callingTo ? $vuetify.lang.t('$vuetify.stop_call') : $vuetify.lang.t('$vuetify.call') }}</span>
                                </v-tooltip>
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
        OPEN_SIMPLE_MODAL, VIDEO_DIAL_STATUS_CHANGED,
    } from "./bus";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import {chat, chat_name, profile, profile_name, videochat_name} from "./routes";
    import debounce from "lodash/debounce";
    import userOnlinePollingMixin from "./userOnlinePollingMixin";
    import queryMixin from "@/queryMixin";

    const firstPage = 1;
    const pageSize = 20;

    const dtoFactory = ()=>{
        return {
            id: null,
            name: "",
            participantIds: [ ],
            participants: [ ],
        }
    };

    export default {
        mixins: [userOnlinePollingMixin(), queryMixin()],
        data () {
            return {
                show: false,
                dto: dtoFactory(),
                chatId: null,

                newParticipantIdsIsLoading: false,
                newParticipantIds: [],
                people: [  ], // available person to chat with
                search: null,
                participantsPage: firstPage,
            }
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
            participantsPagesCount() {
                const count = Math.ceil(this.dto.participantsCount / pageSize);
                console.debug("Calc pages count", count);
                return count;
            },
            shouldShowPagination() {
                return this.dto != null && this.dto.participantsCount > pageSize
            }
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
                if (tmp.participants != null) {
                    tmp.participants.forEach(item => {
                        item.adminLoading = false;
                        item.online = false;
                        item.callingTo = false;
                    });
                }
            },
            translatePage() {
                return this.participantsPage - 1;
            },
            loadData() {
                this.stopPolling();
                console.log("Getting info about chat id in modal, chatId=", this.chatId);
                axios.get('/api/chat/' + this.chatId, {
                    params: {
                        page: this.translatePage(),
                        size: pageSize,
                    },
                })
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
                axios.put(`/api/chat/${this.dto.id}/user/${item.id}`, null, {
                    params: {
                        admin: !item.admin,
                        page: this.translatePage(),
                        size: pageSize,
                    },
                });
            },
            startCalling(dto) {
                const call = !dto.callingTo;
                axios.put(`/api/video/${this.dto.id}/dial?userId=${dto.id}&call=${call}`).then(value => {
                    console.log("Inviting to video chat", call);
                    if (this.$route.name != videochat_name && call) {
                        const routerNewState = { name: videochat_name};
                        this.navigateToWithPreservingSearchStringInQuery(routerNewState);
                    }
                    for (const participant of this.dto.participants) {
                        if (participant.id == dto.id) {
                            participant.callingTo = call;
                            break
                        }
                    }
                })
            },
            kickFromVideoCall(userId) {
                axios.put(`/api/video/${this.dto.id}/kick?userId=${userId}`)
            },
            forceMute(userId) {
                axios.put(`/api/video/${this.dto.id}/mute?userId=${userId}`)
            },
            onChatChange(dto) {
                if (this.show && dto.id == this.chatId) {
                    const oldParticipants = this.dto.participants;
                    const oldParticipantIds = this.dto.participantIds;

                    const clonedDto = Object.assign({}, dto);
                    delete clonedDto.participants;
                    delete clonedDto.participantIds;
                    delete clonedDto.changingParticipantsPage;

                    const serverPage = this.translatePage();

                    this.dto = dtoFactory();
                    this.$nextTick(()=> {
                        this.dto = clonedDto;
                        if (dto.changingParticipantsPage == serverPage) {
                            if (dto.participants) {
                                const tmp = dto;
                                this.transformParticipants(tmp);
                                this.dto = tmp;
                            } else { // no participants means that we need switch page back
                                if (this.participantsPage > firstPage) {
                                    this.participantsPage--;
                                    this.loadData();
                                }
                            }
                        } else { // restore old participants - keep untouched
                            this.dto.participants = oldParticipants;
                            this.dto.participantIds = oldParticipantIds;
                        }
                    });
                }
            },
            deleteParticipant(participant) {
                bus.$emit(OPEN_SIMPLE_MODAL, {
                    buttonName: this.$vuetify.lang.t('$vuetify.delete_btn'),
                    title: this.$vuetify.lang.t('$vuetify.delete_participant', participant.id),
                    text: this.$vuetify.lang.t('$vuetify.delete_participant_text', participant.id, participant.login),
                    actionFunction: ()=> {
                        axios.delete(`/api/chat/${this.dto.id}/user/${participant.id}`, {
                                params: {
                                    page: this.translatePage(),
                                    size: pageSize,
                                },
                            })
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
                this.participantsPage = firstPage;
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

                axios.get(`/api/chat/${this.dto.id}/user?searchString=${searchString}`)
                    .then((response) => {
                      console.debug("Fetched users", response.data);
                      this.people = response.data;
                    })
                    .finally(() => (this.newParticipantIdsIsLoading = false))
            },
            addSelectedParticipants() {
                axios.put(`/api/chat/${this.dto.id}/users`, {
                  addParticipantIds: this.newParticipantIds
                }, {
                    params: {
                        page: this.translatePage(),
                        size: pageSize,
                    },
                }).then(value => {
                    this.newParticipantIds = [];
                    this.search = null;
                })
            },
            onChatDelete(dto) {
                if (this.show && dto.id == this.chatId) {
                    this.closeModal();
                }
            },
            onUserOnlineChanged(dtos) {
                if (this.dto.participants) {
                    this.dto.participants.forEach(item => {
                        dtos.forEach(dtoItem => {
                            if (dtoItem.userId == item.id) {
                                item.online = dtoItem.online;
                            }
                        })
                    })
                    this.$forceUpdate();
                }
            },
            onChatDialStatusChange(dto) {
                if (!this.show || dto.chatId != this.chatId || !this.dto.participants) {
                    return;
                }
                for (const participant of this.dto.participants) {
                    innerLoop:
                    for (const videoDialChanged of dto.dials) {
                        if (participant.id == videoDialChanged.userId) {
                            participant.callingTo = videoDialChanged.status;
                            break innerLoop
                        }
                    }
                }
            },
            onParticipantClick(user) {
                const routeDto = { name: profile_name, params: { id: user.id }};
                this.$router.push(routeDto).then(()=> {
                    this.closeModal();
                })
            },
            getLink(user) {
                let url = profile + "/" + user.id;
                return url;
            },
        },
        watch: {
            search (searchString) {
              this.doNewSearch(searchString);
            },
            participantsPage(newValue) {
                if (this.show) {
                    console.debug("SettingNewPage", newValue);
                    this.dto = dtoFactory();
                    this.loadData();
                }
            },
            show(newValue) {
                if (!newValue) {
                    this.closeModal();
                }
            }
        },

        created() {
            this.doNewSearch = debounce(this.doNewSearch, 700);
            bus.$on(OPEN_PARTICIPANTS_DIALOG, this.showModal);
            bus.$on(CHAT_EDITED, this.onChatChange);
            bus.$on(CHAT_DELETED, this.onChatDelete);
            bus.$on(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
        },
        destroyed() {
            bus.$off(OPEN_PARTICIPANTS_DIALOG, this.showModal);
            bus.$off(CHAT_EDITED, this.onChatChange);
            bus.$off(CHAT_DELETED, this.onChatDelete);
            bus.$off(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
        },
    }
</script>

<style lang="stylus" scoped>

    .call-blink {
        animation: blink 0.5s infinite;
    }

    @keyframes blink {
        50% { opacity: 30% }
    }

</style>
