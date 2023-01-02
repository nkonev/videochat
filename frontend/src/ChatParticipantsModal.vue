<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="700" scrollable>
            <v-card>
                <v-card-title>
                    {{ $vuetify.lang.t('$vuetify.participants_modal_title') }}
                    <v-text-field class="ml-4 pt-0 mt-0" prepend-icon="mdi-magnify" hide-details single-line v-model="userSearchString" :label="$vuetify.lang.t('$vuetify.search_by_users')" clearable clear-icon="mdi-close-circle"></v-text-field>
                </v-card-title>

                <v-card-text  class="ma-0 pa-0">
                    <v-list v-if="dto.participants && dto.participants.length > 0">
                        <template v-for="(item, index) in dto.participants">
                            <v-list-item class="pl-2 ml-1 pr-0 mr-3 mb-1 mt-1">
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
                                <template v-if="item.admin || dto.canChangeChatAdmins">
                                    <template v-if="dto.canChangeChatAdmins && (item.id != currentUser.id)">
                                        <v-btn
                                            :color="item.admin ? 'primary' : 'disabled'"
                                            :loading="item.adminLoading ? true : false"
                                            @click="changeChatAdmin(item)"
                                            icon
                                            :title="item.admin ? $vuetify.lang.t('$vuetify.revoke_admin') : $vuetify.lang.t('$vuetify.grant_admin')"
                                        >
                                            <v-icon>mdi-crown</v-icon>
                                        </v-btn>
                                    </template>
                                    <template v-else-if="item.admin">
                                          <span class="pl-1 pr-1" :title="$vuetify.lang.t('$vuetify.admin')">
                                              <v-icon color="primary">mdi-crown</v-icon>
                                          </span>
                                    </template>
                                </template>

                                <template v-if="dto.canEdit && item.id != currentUser.id">
                                    <v-btn icon @click="deleteParticipant(item)" color="error" :title="$vuetify.lang.t('$vuetify.delete_from_chat')"><v-icon dark>mdi-delete</v-icon></v-btn>
                                </template>
                                <template v-if="dto.canVideoKick && item.id != currentUser.id">
                                    <v-btn icon @click="kickFromVideoCall(item.id)" :title="$vuetify.lang.t('$vuetify.kick')"><v-icon color="error">mdi-block-helper</v-icon></v-btn>
                                </template>
                                <template v-if="dto.canAudioMute && item.id != currentUser.id">
                                    <v-btn icon @click="forceMute(item.id)" :title="$vuetify.lang.t('$vuetify.force_mute')"><v-icon color="error">mdi-microphone-off</v-icon></v-btn>
                                </template>
                                <template v-if="item.id != currentUser.id">
                                    <v-btn icon @click="startCalling(item)" :title="item.callingTo ? $vuetify.lang.t('$vuetify.stop_call') : $vuetify.lang.t('$vuetify.call')"><v-icon :class="{'call-blink': item.callingTo}" color="success">mdi-phone</v-icon></v-btn>
                                </template>
                            </v-list-item>
                            <v-divider></v-divider>
                        </template>
                    </v-list>
                    <template v-else-if="!loading">
                        <v-card-text>{{ $vuetify.lang.t('$vuetify.participants_not_found') }}</v-card-text>
                    </template>

                    <v-progress-circular
                        v-if="loading"
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-pagination
                        v-if="shouldShowPagination"
                        v-model="participantsPage"
                        :length="participantsPagesCount"
                    ></v-pagination>
                    <v-spacer></v-spacer>
                    <v-btn v-if="dto.canEdit" color="primary" class="ma-2 ml-4" @click="addParticipants()">
                        {{ $vuetify.lang.t('$vuetify.add') }}
                    </v-btn>
                    <v-btn color="error" class="my-1" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>

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
        mixins: [userOnlinePollingMixin()],
        data () {
            return {
                show: false,
                dto: dtoFactory(),
                chatId: null,
                userSearchString: null,
                participantsPage: firstPage,
                loading: false,
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
                this.loading = true;
                axios.get('/api/chat/' + this.chatId, {
                    params: {
                        page: this.translatePage(),
                        size: pageSize,
                        userSearchString: this.userSearchString
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
                    }).finally(() => {
                        this.loading = false;
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
                        this.$router.push(routerNewState);
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
                console.debug("Closing ChatParticipantsModal");
                this.loading = false;
                this.show = false;
                this.chatId = null;
                this.dto = dtoFactory();
                this.userSearchString = null;
                this.participantsPage = firstPage;
                this.stopPolling();
            },
            addParticipants() {
                console.log("Add participants");
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
            doSearch(){
                if (this.show) {
                    this.loadData();
                }
            }
        },
        watch: {
            userSearchString (searchString) {
              this.doSearch();
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
            this.doSearch = debounce(this.doSearch, 700);
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
