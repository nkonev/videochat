<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="700" scrollable :persistent="hasSearchString()">
            <v-card>
                <v-card-title>
                    {{ $vuetify.lang.t('$vuetify.participants_modal_title') }}
                    <v-text-field class="ml-4 pt-0 mt-0" prepend-icon="mdi-magnify" hide-details single-line v-model="userSearchString" :label="$vuetify.lang.t('$vuetify.search_by_users')" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput"></v-text-field>
                </v-card-title>

                <v-card-text  class="ma-0 pa-0">
                    <v-list v-if="participantsDto.participants && participantsDto.participants.length > 0">
                        <template v-for="(item, index) in participantsDto.participants">
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
                                    <v-row no-gutters align="center" class="d-flex flex-row">
                                        <v-col class="flex-grow-0 flex-shrink-0">
                                            <v-list-item-title :class="!isMobile() ? 'mr-2' : ''"><a @click.prevent="onParticipantClick(item)" :href="getLink(item)">{{item.login + (item.id == currentUser.id ? $vuetify.lang.t('$vuetify.you_brackets') : '' )}}</a></v-list-item-title>
                                        </v-col>
                                        <v-col v-if="!isMobile()" class="flex-grow-1 flex-shrink-0">
                                            <v-progress-linear
                                                v-if="item.callingTo"
                                                color="success"
                                                buffer-value="0"
                                                height="16"
                                                indeterminate
                                                stream
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
                                <template v-if="dto.canVideoKick && item.id != currentUser.id && isVideo()">
                                    <v-btn icon @click="kickFromVideoCall(item.id)" :title="$vuetify.lang.t('$vuetify.kick')"><v-icon color="error">mdi-block-helper</v-icon></v-btn>
                                </template>
                                <template v-if="dto.canAudioMute && item.id != currentUser.id && isVideo()">
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
        CHAT_DELETED, CHAT_EDITED,
        CLOSE_SIMPLE_MODAL, OPEN_CHAT_EDIT,
        OPEN_PARTICIPANTS_DIALOG,
        OPEN_SIMPLE_MODAL, PARTICIPANT_ADDED, PARTICIPANT_DELETED, PARTICIPANT_EDITED, VIDEO_DIAL_STATUS_CHANGED,
    } from "./bus";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import {profile, profile_name, videochat_name} from "./routes";
    import graphqlSubscriptionMixin from "./graphqlSubscriptionMixin"
    import {findIndex, hasLength, isArrEqual, moveToFirstPosition, replaceInArray} from "@/utils";
    import cloneDeep from "lodash/cloneDeep";
    import debounce from "lodash/debounce";
    const firstPage = 1;
    const pageSize = 20;

    const dtoFactory = ()=>{
        return { }
    };

    const participantsDtoFactory = () => {
        return {
            participants: [],
            participantsCount: 0
        }
    }

    export default {
        mixins: [graphqlSubscriptionMixin('userOnlineInChatParticipants')],
        data () {
            return {
                show: false,
                dto: dtoFactory(),
                participantsDto: participantsDtoFactory(),
                chatId: null,
                userSearchString: null,
                participantsPage: firstPage,
                loading: false,
            }
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
            participantsPagesCount() {
                const count = Math.ceil(this.participantsDto.participantsCount / pageSize);
                console.debug("Calc pages count", count);
                return count;
            },
            shouldShowPagination() {
                return this.participantsDto != null && this.participantsDto.participantsCount > pageSize
            }
        },

        methods: {
            showModal(chatId) {
                this.chatId = chatId;

                this.show = true;
                if (this.chatId && this.show) {
                    this.loadData().then(() => this.loadParticipantsData())
                } else {
                    this.dto = dtoFactory();
                    this.participantsDto = participantsDtoFactory();
                }
            },
            translatePage() {
                return this.participantsPage - 1;
            },
            loadData() {
                console.log("Getting info about chat id in modal, chatId=", this.chatId);
                this.loading = true;
                return axios.get('/api/chat/' + this.chatId)
                    .then((response) => {
                        this.dto = response.data;
                    })
            },
            loadParticipantsData() {
                console.log("Getting info about participants in modal, chatId=", this.chatId);
                this.loading = true;
                return axios.get('/api/chat/' + this.chatId + '/user', {
                            params: {
                                page: this.translatePage(),
                                size: pageSize,
                                searchString: this.userSearchString
                            },
                        })
                        .then((response) => {
                            const tmp = cloneDeep(response.data);
                            this.transformParticipants(tmp.participants);
                            this.participantsDto = tmp;
                        }).finally(() => {
                            this.loading = false;
                            axios.put('/api/video/' + this.chatId + '/ask-dials')
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
                axios.put(`/api/video/${this.dto.id}/dial/invite?userId=${dto.id}&call=${call}`).then(value => {
                    console.log("Inviting to video chat", call);
                    if (this.$route.name != videochat_name && call) {
                        const routerNewState = { name: videochat_name};
                        this.$router.push(routerNewState);
                    }
                    for (const participant of this.participantsDto.participants) {
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
                this.participantsDto = participantsDtoFactory();
                this.userSearchString = null;
                this.participantsPage = firstPage;
            },
            addParticipants() {
                bus.$emit(OPEN_CHAT_EDIT, this.chatId);
            },
            onChatDelete(dto) {
                if (this.show && dto.id == this.chatId) {
                    this.closeModal();
                }
            },
            onChatEdit(dto) {
                if (!this.show) {
                    return
                }

                // actually it is need only to reflect canEdit and friends
                this.dto = dto;
            },
            onUserOnlineChanged(rawData) {
                const dtos = rawData?.data?.userOnlineEvents;
                if (this.participantsDto.participants && dtos) {
                    this.participantsDto.participants.forEach(item => {
                        dtos.forEach(dtoItem => {
                            if (dtoItem.id == item.id) {
                                item.online = dtoItem.online;
                            }
                        })
                    })
                    this.$forceUpdate();
                }
            },
            onChatDialStatusChange(dto) {
                if (!this.show || dto.chatId != this.chatId || !this.participantsDto.participants) {
                    return;
                }
                for (const participant of this.participantsDto.participants) {
                    innerLoop:
                    for (const videoDialChanged of dto.dials) {
                        if (participant.id == videoDialChanged.userId) {
                            participant.callingTo = videoDialChanged.status;
                            break innerLoop
                        }
                    }
                }
                this.$forceUpdate();
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
                    this.loadParticipantsData();
                }
            },
            transformParticipants(participants) {
                if (participants != null) {
                    participants.forEach(item => {
                        item.adminLoading = false;
                        item.online = false;
                        item.callingTo = false;
                    });
                }
            },

            addItem(dto) {
                console.log("Adding item", dto);
                this.participantsDto.participants.unshift(dto);
            },
            changeItem(dto) {
                console.log("Replacing item", dto);
                if (this.hasItem(dto)) {
                    replaceInArray(this.participantsDto.participants, dto);
                    moveToFirstPosition(this.participantsDto.participants, dto)
                } else {
                    this.participantsDto.participants.unshift(dto);
                }
            },
            removeItem(dto) {
                if (this.hasItem(dto)) {
                    console.log("Removing item", dto);
                    const idxToRemove = findIndex(this.participantsDto.participants, dto);
                    this.participantsDto.participants.splice(idxToRemove, 1);
                } else {
                    console.log("Item was not be removed", dto);
                }
            },
            // does should change items list (new item added to visible part or not for example)
            hasItem(item) {
                let idxOf = findIndex(this.participantsDto.participants, item);
                return idxOf !== -1;
            },

            onParticipantAdded(users) {
                if (!this.show) {
                    return
                }

                const tmp = cloneDeep(users);
                this.transformParticipants(tmp);
                for (const user of tmp) {
                    this.addItem(user);
                }
                this.$forceUpdate();
            },
            onParticipantDeleted(users) {
                if (!this.show) {
                    return
                }

                const tmp = cloneDeep(users);
                this.transformParticipants(tmp);
                for (const user of tmp) {
                    this.removeItem(user);
                }
                this.$forceUpdate();
            },
            onParticipantEdited(users) {
                if (!this.show) return

                const tmp = cloneDeep(users);
                this.transformParticipants(tmp);
                for (const user of tmp) {
                    this.changeItem(user);
                }
                this.$forceUpdate();
            },
            hasSearchString() {
                return hasLength(this.userSearchString)
            },
            resetInput() {
                this.userSearchString = null;
            },
            isVideo() {
                return this.$router.currentRoute.name == videochat_name
            },

            getGraphQlSubscriptionQuery() {
                return `
                subscription {
                    userOnlineEvents(userIds:[${this.participantsDto.participants.map((p)=> p.id ).join(", ")}]) {
                        id
                        online
                    }
                }`
            },
            onNextSubscriptionElement(items) {
                this.onUserOnlineChanged(items);
            },
        },
        watch: {
            userSearchString (searchString) {
              this.doSearch();
            },
            participantsPage(newValue) {
                if (this.show) {
                    console.debug("SettingNewPage", newValue);
                    this.participantsDto = participantsDtoFactory();
                    this.loadParticipantsData();
                }
            },
            show(newValue) {
                if (!newValue) {
                    this.closeModal();
                }
            },
            participantsDto(newValue, oldValue) {
                const oldArr = oldValue?.participants.map((p)=> p.id );
                const newArr = newValue?.participants.map((p)=> p.id );
                if (newArr == null || newArr.length == 0) {
                    this.graphQlUnsubscribe();
                } else {
                    if (!isArrEqual(oldArr, newArr)) {
                        this.graphQlSubscribe();
                    }
                }
            }
        },

        created() {
            this.doSearch = debounce(this.doSearch, 700);
            bus.$on(OPEN_PARTICIPANTS_DIALOG, this.showModal);
            bus.$on(PARTICIPANT_ADDED, this.onParticipantAdded);
            bus.$on(PARTICIPANT_DELETED, this.onParticipantDeleted);
            bus.$on(PARTICIPANT_EDITED, this.onParticipantEdited);
            bus.$on(CHAT_DELETED, this.onChatDelete);
            bus.$on(CHAT_EDITED, this.onChatEdit);
            bus.$on(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
        },
        beforeDestroy() {
            this.graphQlUnsubscribe();
        },
        destroyed() {
            bus.$off(OPEN_PARTICIPANTS_DIALOG, this.showModal);
            bus.$off(PARTICIPANT_ADDED, this.onParticipantAdded);
            bus.$off(PARTICIPANT_DELETED, this.onParticipantDeleted);
            bus.$off(PARTICIPANT_EDITED, this.onParticipantEdited);
            bus.$off(CHAT_DELETED, this.onChatDelete);
            bus.$off(CHAT_EDITED, this.onChatEdit);
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
