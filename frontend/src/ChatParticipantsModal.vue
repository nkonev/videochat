<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable :persistent="hasSearchString()">
            <v-card>
                <v-card-title class="d-flex align-center ml-2">
                    <template v-if="showSearchButton">
                      {{ $vuetify.locale.t('$vuetify.participants_modal_title') }}
                    </template>
                    <v-spacer/>
                    <CollapsedSearch :provider="{
                        getModelValue: this.getModelValue,
                        setModelValue: this.setModelValue,
                        getShowSearchButton: this.getShowSearchButton,
                        setShowSearchButton: this.setShowSearchButton,
                        searchName: this.searchName,
                        textFieldVariant: 'outlined',
                    }"/>

                </v-card-title>

                <v-card-text class="ma-0 pa-0 participants-list">
                    <v-list v-if="participantsDto.participants && participantsDto.participants.length > 0">
                        <template v-for="(item, index) in participantsDto.participants">
                            <v-list-item
                              class="list-item-prepend-spacer-16"
                              @contextmenu.stop="onShowContextMenu($event, item)"
                            >
                                <template v-slot:prepend v-if="hasLength(item.avatar)">
                                    <v-badge
                                        v-if="item.avatar"
                                        :color="getUserBadgeColor(item)"
                                        dot
                                        location="right bottom"
                                        overlap
                                        bordered
                                        :model-value="item.online"
                                    >
                                        <v-avatar :image="item.avatar"></v-avatar>
                                    </v-badge>

                                </template>

                                <v-row no-gutters align="center" class="d-flex flex-row">
                                    <v-col class="flex-grow-0 flex-shrink-0">
                                        <v-list-item-title><a class="colored-link" @click.prevent="onParticipantClick(item)" :href="getLink(item)">{{getUserNameWrapper(item)}}</a></v-list-item-title>
                                    </v-col>
                                    <v-col v-if="!isMobile()" class="ml-4 flex-grow-1 flex-shrink-0">
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

                                <template v-slot:append>
                                    <template v-if="item.admin || dto.canChangeChatAdmins">
                                        <template v-if="dto.canChangeChatAdmins && item.id != chatStore.currentUser.id && !isMobile()">
                                            <v-btn
                                                variant="flat"
                                                :loading="item.adminLoading ? true : false"
                                                @click="changeChatAdmin(item)"
                                                icon
                                                :title="item.admin ? $vuetify.locale.t('$vuetify.revoke_chat_admin') : $vuetify.locale.t('$vuetify.grant_chat_admin')"
                                            >
                                                <v-icon :color="item.admin ? 'primary' : 'disabled'">mdi-crown</v-icon>
                                            </v-btn>
                                        </template>
                                        <template v-else-if="item.admin">
                                              <span class="pl-1 pr-1" :title="$vuetify.locale.t('$vuetify.chat_admin')">
                                                  <v-icon color="primary">mdi-crown</v-icon>
                                              </span>
                                        </template>
                                    </template>
                                    <template v-if="!isMobile()">
                                        <template v-if="dto.canEdit && item.id != chatStore.currentUser.id">
                                            <v-btn variant="flat" icon @click="deleteParticipant(item)" :title="$vuetify.locale.t('$vuetify.delete_from_chat')"><v-icon color="red">mdi-delete</v-icon></v-btn>
                                        </template>
                                        <template v-if="dto.canVideoKick && item.id != chatStore.currentUser.id && isVideo()">
                                            <v-btn variant="flat" icon @click="kickFromVideoCall(item)" :title="$vuetify.locale.t('$vuetify.kick')"><v-icon color="red">mdi-block-helper</v-icon></v-btn>
                                        </template>
                                        <template v-if="dto.canAudioMute && item.id != chatStore.currentUser.id && isVideo()">
                                            <v-btn variant="flat" icon @click="forceMute(item)" :title="$vuetify.locale.t('$vuetify.force_mute')"><v-icon color="red">mdi-microphone-off</v-icon></v-btn>
                                        </template>
                                    </template>

                                    <template v-if="item.id != chatStore.currentUser.id">
                                        <v-btn variant="flat" icon @click="startCalling(item)" :title="item.callingTo ? $vuetify.locale.t('$vuetify.stop_call') : $vuetify.locale.t('$vuetify.call')"><v-icon :class="{'call-blink': item.callingTo}" color="success">mdi-phone</v-icon></v-btn>
                                    </template>
                                </template>
                            </v-list-item>
                            <v-divider></v-divider>
                        </template>
                    </v-list>
                    <template v-else-if="!loading">
                        <v-card-text>{{ $vuetify.locale.t('$vuetify.participants_not_found') }}</v-card-text>
                    </template>
                    <ChatParticipantsContextMenu
                      ref="contextMenuRef"
                      @deleteParticipantFromChat="this.deleteParticipant"
                      @kickParticipantFromChat="this.kickFromVideoCall"
                      @forceMuteParticipantInChat="this.forceMute"
                      @changeChatAdmin="this.changeChatAdmin"
                    />

                    <v-progress-circular
                        class="ma-4"
                        v-if="loading"
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">

                    <!-- Pagination is shuddering / flickering on the second page without this wrapper -->
                    <v-row no-gutters class="ma-0 pa-0 d-flex flex-row">
                        <v-col class="ma-0 pa-0 flex-grow-1 flex-shrink-0" :class="isMobile() ? 'mb-2' : ''">
                            <v-pagination
                                variant="elevated"
                                active-color="primary"
                                :density="isMobile() ? 'compact' : 'comfortable'"
                                v-if="shouldShowPagination"
                                v-model="page"
                                :length="pagesCount"
                                :total-visible="isMobile() ? 3 : 7"
                            ></v-pagination>
                        </v-col>
                        <v-col class="ma-0 pa-0 d-flex flex-row flex-grow-1 flex-shrink-0 align-self-end justify-end">
                            <v-btn v-if="dto.canEdit" color="primary" variant="flat" @click="addParticipants()">
                                {{ $vuetify.locale.t('$vuetify.add') }}
                            </v-btn>
                            <v-btn color="red" variant="flat" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                        </v-col>
                    </v-row>
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
      OPEN_CHAT_EDIT,
      OPEN_PARTICIPANTS_DIALOG,
      OPEN_SIMPLE_MODAL,
      PARTICIPANT_ADDED,
      PARTICIPANT_DELETED,
      PARTICIPANT_EDITED,
      USER_PROFILE_CHANGED,
      VIDEO_DIAL_STATUS_CHANGED,
    } from "./bus/bus";
    import {profile, profile_name, videochat_name} from "./router/routes";
    import userStatusMixin from "@/mixins/userStatusMixin";
    import {deepCopy, findIndex, hasLength, isArrEqual, moveToFirstPosition, replaceInArray} from "@/utils";
    import debounce from "lodash/debounce";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import Mark from "mark.js";
    import CollapsedSearch from "@/CollapsedSearch.vue";
    import ChatParticipantsContextMenu from "@/ChatParticipantsContextMenu.vue";
    import ChatListContextMenu from "@/ChatListContextMenu.vue";

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
        mixins: [userStatusMixin('userOnlineInChatParticipants')],
        data () {
            return {
                show: false,
                dto: dtoFactory(),
                participantsDto: participantsDtoFactory(),
                chatId: null,
                userSearchString: null,
                page: firstPage,
                loading: false,
                showSearchButton: true,
                markInstance: null,
            }
        },
        computed: {
            pagesCount() {
                const count = Math.ceil(this.participantsDto.participantsCount / pageSize);
                // console.debug("Calc pages count", count);
                return count;
            },
            shouldShowPagination() {
                return this.participantsDto != null && this.participantsDto.participantsCount > pageSize
            },
            ...mapStores(useChatStore),
        },

        methods: {
            hasLength,
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
                return this.page - 1;
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
                            const tmp = deepCopy(response.data);
                            this.transformParticipantsWrapper(tmp.participants);
                            this.participantsDto = tmp;
                        }).finally(() => {
                            this.loading = false;
                            axios.put('/api/video/' + this.chatId + '/dial/request-for-is-calling');
                            this.performMarking();
                    })
            },
            changeChatAdmin(item) {
                item.adminLoading = true;
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
            kickFromVideoCall(item) {
                axios.put(`/api/video/${this.dto.id}/kick?userId=${item.id}`)
            },
            forceMute(item) {
                axios.put(`/api/video/${this.dto.id}/mute?userId=${item.id}`)
            },
            deleteParticipant(participant) {
                bus.emit(OPEN_SIMPLE_MODAL, {
                    buttonName: this.$vuetify.locale.t('$vuetify.delete_btn'),
                    title: this.$vuetify.locale.t('$vuetify.delete_participant', participant.id),
                    text: this.$vuetify.locale.t('$vuetify.delete_participant_text', participant.id, participant.login),
                    actionFunction: (that)=> {
                        that.loading = true;
                        axios.delete(`/api/chat/${this.dto.id}/user/${participant.id}`, {
                                params: {
                                    page: this.translatePage(),
                                    size: pageSize,
                                },
                            })
                            .then(() => {
                                bus.emit(CLOSE_SIMPLE_MODAL);
                            }).finally(()=>{
                                that.loading = false;
                            })
                    }
                });
            },
            closeModal() {
                console.debug("Closing ChatParticipantsModal");
                this.graphQlUnsubscribe();

                this.loading = false;
                this.show = false;
                this.chatId = null;
                this.dto = dtoFactory();
                this.participantsDto = participantsDtoFactory();
                this.userSearchString = null;
                this.page = firstPage;
                this.showSearchButton = true;
            },
            addParticipants() {
                bus.emit(OPEN_CHAT_EDIT, this.chatId);
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
            getUserIdsSubscribeTo() {
                if (this.participantsDto?.participants){
                    return this.participantsDto.participants.map(item => item.id);
                } else {
                    return []
                }
            },
            onUserOnlineChanged(rawData) {
                const dtos = rawData?.data?.userOnlineEvents;
                if (this.participantsDto?.participants && dtos) {
                    this.participantsDto.participants.forEach(item => {
                        dtos.forEach(dtoItem => {
                            if (dtoItem.id == item.id) {
                                item.online = dtoItem.online;
                            }
                        })
                    })
                }
            },
            onUserVideoStatusChanged(rawData) {
                const dtos = rawData?.data?.userVideoStatusEvents;
                if (this.participantsDto?.participants && dtos) {
                    this.participantsDto.participants.forEach(item => {
                        dtos.forEach(dtoItem => {
                            if (item.id == dtoItem.userId) {
                                item.isInVideo = dtoItem.isInVideo;
                            }
                        })
                    })
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
                    this.page = firstPage;
                    this.loadParticipantsData();
                }
            },
            transformParticipantsWrapper(participants) {
                if (participants != null) {
                    participants.forEach(item => {
                        item.adminLoading = false;
                        item.callingTo = false;
                        this.transformItem(item);
                    });
                }
            },
            getUserNameWrapper(item) {
                let bldr = this.getUserName(item);
                if (item.id == this.chatStore.currentUser.id) {
                    bldr += " ";
                    bldr += this.$vuetify.locale.t('$vuetify.you_brackets');
                    bldr += " ";
                }

                return bldr;
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

                const tmp = deepCopy(users);
                this.transformParticipantsWrapper(tmp);
                for (const user of tmp) {
                    this.addItem(user);
                }
                this.performMarking();
            },
            onParticipantDeleted(users) {
                if (!this.show) {
                    return
                }

                const tmp = deepCopy(users);
                this.transformParticipantsWrapper(tmp);
                for (const user of tmp) {
                    this.removeItem(user);
                }
            },
            onParticipantEdited(users) {
                if (!this.show) return

                const tmp = deepCopy(users);
                this.transformParticipantsWrapper(tmp);
                for (const user of tmp) {
                    this.changeItem(user);
                }
                this.performMarking();
            },
            onUserProfileChanged(user) {
              this.participantsDto.participants.forEach(item => {
                if (item.id == user.id) { // replaces content of tet-a-tet. It's better to move it to chat
                  item.avatar = user.avatar;
                  item.login = user.login;
                }
              });
              this.performMarking();
            },
            hasSearchString() {
                return hasLength(this.userSearchString)
            },
            isVideo() {
                return this.$route.name == videochat_name
            },

            getGraphQlSubscriptionQuery() {
                return `
                    subscription {
                        userOnlineEvents(userIds:[${this.participantsDto.participants.map((p)=> p.id ).join(", ")}]) {
                            id
                            online
                        }
                    }
                `
            },
            onNextSubscriptionElement(items) {
                this.onUserOnlineChanged(items);
            },
            performMarking() {
              this.$nextTick(() => {
                this.markInstance.unmark();
                if (hasLength(this.userSearchString)) {
                  this.markInstance.mark(this.userSearchString);
                }
              })
            },
            getModelValue() {
              return this.userSearchString
            },
            setModelValue(v) {
              this.userSearchString = v
            },
            getShowSearchButton() {
              return this.showSearchButton
            },
            setShowSearchButton(v) {
              this.showSearchButton = v
            },
            searchName() {
              return this.$vuetify.locale.t('$vuetify.search_by_participants')
            },
            onShowContextMenu(e, menuableItem) {
              this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem, this.dto);
            },

        },
        watch: {
            userSearchString (searchString) {
              this.doSearch();
            },
            page(newValue) {
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
        components: {
          ChatListContextMenu,
          ChatParticipantsContextMenu,
          CollapsedSearch
        },

        created() {
            this.doSearch = debounce(this.doSearch, 700);
        },
        mounted() {
          bus.on(OPEN_PARTICIPANTS_DIALOG, this.showModal);
          bus.on(PARTICIPANT_ADDED, this.onParticipantAdded);
          bus.on(PARTICIPANT_DELETED, this.onParticipantDeleted);
          bus.on(PARTICIPANT_EDITED, this.onParticipantEdited);
          bus.on(CHAT_DELETED, this.onChatDelete);
          bus.on(CHAT_EDITED, this.onChatEdit);
          bus.on(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
          bus.on(USER_PROFILE_CHANGED, this.onUserProfileChanged);

          this.markInstance = new Mark(".participants-list");
        },
        beforeUnmount() {
            bus.off(OPEN_PARTICIPANTS_DIALOG, this.showModal);
            bus.off(PARTICIPANT_ADDED, this.onParticipantAdded);
            bus.off(PARTICIPANT_DELETED, this.onParticipantDeleted);
            bus.off(PARTICIPANT_EDITED, this.onParticipantEdited);
            bus.off(CHAT_DELETED, this.onChatDelete);
            bus.off(CHAT_EDITED, this.onChatEdit);
            bus.off(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
            bus.off(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            this.markInstance.unmark();
            this.markInstance = null;
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
