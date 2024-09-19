<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="740" scrollable :persistent="true">
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
                    <template v-if="!loading">
                        <v-list v-if="itemsDto.items && itemsDto.count > 0">
                            <template v-for="(item, index) in itemsDto.items">
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
                                            <v-list-item-title><a class="nodecorated-link" @click.prevent="onParticipantClick(item)" :href="getLink(item)" :style="getLoginColoredStyle(item, true)">{{getUserNameWrapper(item)}}</a></v-list-item-title>
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
                                        <template v-if="item.admin || chatStore.chatDto.canChangeChatAdmins">
                                            <template v-if="chatStore.chatDto.canChangeChatAdmins && item.id != chatStore.currentUser.id && !isMobile()">
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
                                            <template v-if="chatStore.chatDto.canEdit && item.id != chatStore.currentUser.id">
                                                <v-btn variant="flat" icon @click="deleteParticipant(item)" :title="$vuetify.locale.t('$vuetify.delete_from_chat')"><v-icon color="red">mdi-delete</v-icon></v-btn>
                                            </template>
                                            <template v-if="chatStore.chatDto.canVideoKick && item.id != chatStore.currentUser.id && isVideo()">
                                                <v-btn variant="flat" icon @click="kickFromVideoCall(item)" :title="$vuetify.locale.t('$vuetify.kick')"><v-icon color="red">mdi-block-helper</v-icon></v-btn>
                                            </template>
                                            <template v-if="chatStore.chatDto.canAudioMute && item.id != chatStore.currentUser.id && isVideo()">
                                                <v-btn variant="flat" icon @click="forceMute(item)" :title="$vuetify.locale.t('$vuetify.force_mute')"><v-icon color="red">mdi-microphone-off</v-icon></v-btn>
                                            </template>
                                        </template>

                                        <template v-if="item.id != chatStore.currentUser.id">
                                            <v-btn variant="flat" icon @click="inviteToCall(item)" :title="item.callingTo ? $vuetify.locale.t('$vuetify.stop_call') : $vuetify.locale.t('$vuetify.call')"><v-icon :class="{'call-blink': item.callingTo}" color="success">mdi-phone</v-icon></v-btn>
                                        </template>
                                    </template>
                                </v-list-item>
                                <v-divider></v-divider>
                            </template>
                        </v-list>
                        <template v-else-if="!loading">
                            <v-card-text>{{ $vuetify.locale.t('$vuetify.participants_not_found') }}</v-card-text>
                        </template>
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
                                density="comfortable"
                                v-if="shouldShowPagination"
                                v-model="page"
                                :length="pagesCount"
                                :total-visible="getTotalVisible()"
                            ></v-pagination>
                        </v-col>
                        <v-col class="ma-0 pa-0 d-flex flex-row flex-grow-1 flex-shrink-0 align-self-end justify-end">
                            <v-btn v-if="chatStore.chatDto.canEdit" color="primary" variant="flat" @click="addParticipants()">
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
        CLOSE_SIMPLE_MODAL,
        OPEN_CHAT_EDIT,
        OPEN_PARTICIPANTS_DIALOG,
        OPEN_SIMPLE_MODAL,
        PARTICIPANT_ADDED,
        PARTICIPANT_DELETED,
        PARTICIPANT_EDITED,
        CO_CHATTED_PARTICIPANT_CHANGED,
        VIDEO_DIAL_STATUS_CHANGED, LOGGED_OUT,
    } from "./bus/bus";
    import {profile, profile_name, videochat_name} from "./router/routes";
    import userStatusMixin from "@/mixins/userStatusMixin";
    import {
        deepCopy,
        findIndex,
        getLoginColoredStyle,
        hasLength,
        isCalling,
        isSetEqual,
        replaceInArray
    } from "@/utils";
    import debounce from "lodash/debounce";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import Mark from "mark.js";
    import CollapsedSearch from "@/CollapsedSearch.vue";
    import ChatParticipantsContextMenu from "@/ChatParticipantsContextMenu.vue";
    import ChatListContextMenu from "@/ChatListContextMenu.vue";
    import pageableModalMixin, {firstPage, pageSize} from "@/mixins/pageableModalMixin.js";

    export default {
        mixins: [
            userStatusMixin('chatParticipants'),
            pageableModalMixin(),
        ],
        data () {
            return {
                chatId: null,
                userSearchString: null,
                showSearchButton: true,
                markInstance: null,
            }
        },
        computed: {
            ...mapStores(useChatStore),
            participantIds() {
                const tmps = deepCopy(this.itemsDto?.items || []);
                return tmps.map((item) => item.id);
            },
        },

        methods: {
            getLoginColoredStyle,
            hasLength,
            isCachedRelevantToArguments({chatId}) {
                return this.chatId == chatId
            },
            initializeWithArguments({chatId}) {
                this.chatId = chatId;
            },
            initiateRequest() {
                return axios.get(`/api/chat/${this.chatId}/participant`, {
                    params: {
                        page: this.translatePage(),
                        size: pageSize,
                        searchString: this.userSearchString
                    },
                })
            },
            afterFirstDrawItems() {
                this.$nextTick(()=>{
                    const userIds = this.itemsDto?.items.map(item => item.id);
                    const joined = userIds.join(",");

                    axios.put("/api/video/user/request-in-video-status", null, {
                        params: {
                            userId: joined
                        },
                    });
                })
            },
            changeChatAdmin(item) {
                item.adminLoading = true;
                axios.put(`/api/chat/${this.chatId}/participant/${item.id}`, null, {
                    params: {
                        admin: !item.admin,
                    },
                });
            },
            inviteToCall(dto) {
                const call = !dto.callingTo;
                axios.put(`/api/video/${this.chatId}/dial/invite`, null, {
                    params: {
                        userId: dto.id,
                        call: call,
                        tokenId: this.chatStore.videoTokenId,
                    }
                }).then(resp => {
                    // if we are't in call we receive the token in advance
                    this.chatStore.videoTokenId = resp.data.tokenId;

                    // console.log("Inviting to video chat", call);
                    if (this.$route.name != videochat_name && call) {
                        const routerNewState = { name: videochat_name};
                        this.$router.push(routerNewState);
                    }
                }).catch((e) => {
                  if (e.response.status == 409) {
                    this.setWarning(this.$vuetify.locale.t('$vuetify.user_is_already_in_other_call', this.getUserNameWrapper(dto)))
                  } else {
                    throw e
                  }
                })
            },
            kickFromVideoCall(item) {
                axios.put(`/api/video/${this.chatId}/kick?userId=${item.id}`)
            },
            forceMute(item) {
                axios.put(`/api/video/${this.chatId}/mute?userId=${item.id}`)
            },
            deleteParticipant(participant) {
                bus.emit(OPEN_SIMPLE_MODAL, {
                    buttonName: this.$vuetify.locale.t('$vuetify.delete_btn'),
                    title: this.$vuetify.locale.t('$vuetify.delete_participant', participant.id),
                    text: this.$vuetify.locale.t('$vuetify.delete_participant_text', participant.id, participant.login),
                    actionFunction: (that)=> {
                        that.loading = true;
                        axios.delete(`/api/chat/${this.chatId}/participant/${participant.id}`)
                            .then(() => {
                                bus.emit(CLOSE_SIMPLE_MODAL);
                            }).finally(()=>{
                                that.loading = false;
                            })
                    }
                });
            },
            addParticipants() {
                bus.emit(OPEN_CHAT_EDIT, this.chatStore.chatDto);
            },
            onChatDelete(dto) {
                if (this.show && dto.id == this.chatId) {
                    this.closeModal();
                }
                if (dto.id == this.chatId) {
                    this.reset();
                }
            },
            getUserIdsSubscribeTo() {
                if (this.itemsDto?.items){
                    return this.itemsDto.items.map(item => item.id);
                } else {
                    return []
                }
            },
            onUserStatusChanged(dtos) {
                if (this.itemsDto?.items && dtos) {
                    this.itemsDto.items.forEach(item => {
                        dtos.forEach(dtoItem => {
                            if (dtoItem.online !== null && item.id == dtoItem.userId) {
                                item.online = dtoItem.online;
                            }
                            if (dtoItem.isInVideo !== null && item.id == dtoItem.userId) {
                              item.isInVideo = dtoItem.isInVideo;
                            }
                        })
                    })
                }
            },
            onChatDialStatusChange(dto) {
                if (dto.chatId != this.chatId || !this.itemsDto.items) {
                    return;
                }

                for (const participant of this.itemsDto.items) {
                    innerLoop:
                    for (const videoDialChanged of dto.dials) {
                        if (participant.id == videoDialChanged.userId) {
                            this.$nextTick(()=>{
                              participant.callingTo = isCalling(videoDialChanged.status);
                            })
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
                if (!this.dataLoaded) {
                    return
                }

                this.page = firstPage;
                this.updateItems();
            },
            transformItems(items) {
                if (items != null) {
                    items.forEach(item => {
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

            // does should change items list (new item added to visible part or not for example)
            hasItem(item) {
                let idxOf = findIndex(this.itemsDto.items, item);
                return idxOf !== -1;
            },

            extractDtoFromEventDto(eventDto) {
                return eventDto
            },

            onUserProfileChanged(user) {
                const tmp = deepCopy(user);
                const arrTmp = [tmp];
                this.transformItems(arrTmp);

                replaceInArray(this.itemsDto.items, arrTmp[0]);

                this.performMarking();
            },
            hasSearchString() {
                return hasLength(this.userSearchString)
            },
            isVideo() {
                return this.$route.name == videochat_name
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
            initiateFilteredRequest(eventDto) {
                return axios.post(`/api/chat/${this.chatId}/participant/filter`, {
                    searchString: this.userSearchString,
                    userId: eventDto.map(p => p.id) // participantIds
                })
            },
            initiateCountRequest() {
                return axios.post(`/api/chat/${this.chatId}/participant/count`,{
                    searchString: this.userSearchString,
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
              this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem, this.chatStore.chatDto);
            },
            clearOnClose() {
                this.showSearchButton = true;
            },
            clearOnReset() {
                this.chatId = null;
                // because resubscription happens on change computed participantIds
                // we will hold subscription on status change because we still have participants across opening and closing this modal
                this.graphQlUserStatusUnsubscribe();

                this.userSearchString = null;
            },
            resetOnRouteIdChange(){
                return true
            },
            shouldReactOnPageChange() {
                return this.show
            },
        },
        watch: {
            userSearchString (searchString) {
              this.doSearch();
            },
            participantIds(newArr, oldArr) {
                if (oldArr.length !== 0 && newArr.length === 0) {
                    this.graphQlUserStatusUnsubscribe();
                } else {
                    if (!isSetEqual(oldArr, newArr)) {
                        this.graphQlUserStatusSubscribe();
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
          bus.on(PARTICIPANT_ADDED, this.onItemCreatedEvent);
          bus.on(PARTICIPANT_DELETED, this.onItemRemovedEvent);
          bus.on(PARTICIPANT_EDITED, this.onItemUpdatedEvent);
          bus.on(CHAT_DELETED, this.onChatDelete);
          bus.on(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
          bus.on(CO_CHATTED_PARTICIPANT_CHANGED, this.onUserProfileChanged);
          bus.on(LOGGED_OUT, this.onLogout);

          this.markInstance = new Mark(".participants-list");
        },
        beforeUnmount() {
            bus.off(OPEN_PARTICIPANTS_DIALOG, this.showModal);
            bus.off(PARTICIPANT_ADDED, this.onItemCreatedEvent);
            bus.off(PARTICIPANT_DELETED, this.onItemRemovedEvent);
            bus.off(PARTICIPANT_EDITED, this.onItemUpdatedEvent);
            bus.off(CHAT_DELETED, this.onChatDelete);
            bus.off(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
            bus.off(CO_CHATTED_PARTICIPANT_CHANGED, this.onUserProfileChanged);
            bus.off(LOGGED_OUT, this.onLogout);
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
