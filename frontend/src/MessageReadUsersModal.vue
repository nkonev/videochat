<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="520" scrollable>
            <v-card :title="$vuetify.locale.t('$vuetify.users_read') + ' #' + this.messageDto?.messageId">
                <v-card-text class="ma-0 pa-0">
                    <div v-if="shouldShowMessageText()" :class="messageWrapperClass()">
                        <v-container :class="messageClass()" v-html="participantsDto.text"></v-container>
                    </div>
                    <v-list v-if="participantsDto.participants && participantsDto.participants.length > 0" class="pb-0">
                        <template v-for="(item, index) in participantsDto.participants">
                            <v-list-item @click.prevent="onParticipantClick(item)" :href="getLink(item)">
                                <template v-slot:prepend v-if="hasLength(item.avatar)">
                                    <v-avatar :image="item.avatar"></v-avatar>
                                </template>
                                <v-list-item-title>
                                {{item.login + (item.id == chatStore.currentUser.id ? $vuetify.locale.t('$vuetify.you_brackets') : '' )}}
                              </v-list-item-title>
                            </v-list-item>
                        </template>
                    </v-list>
                    <template v-else-if="!loading">
                        <v-card-text>{{ $vuetify.locale.t('$vuetify.participants_not_found') }}</v-card-text>
                    </template>

                    <v-progress-circular
                        class="ma-4"
                        v-if="loading"
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                </v-card-text>

                <v-card-actions class="my-actions d-flex flex-wrap flex-row">

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
                      <v-btn
                        variant="elevated"
                        color="red"
                        @click="closeModal()"
                      >
                        {{ $vuetify.locale.t('$vuetify.close') }}
                      </v-btn>
                    </v-col>
                  </v-row>
                </v-card-actions>

            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    OPEN_MESSAGE_READ_USERS_DIALOG,
} from "./bus/bus";
import axios from "axios";
import {profile, profile_name} from "@/router/routes";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {hasLength} from "@/utils";
import "./messageWrapper.styl";

const firstPage = 1;
const pageSize = 20;

const participantsDtoFactory = () => {
    return {
        participants: [],
        participantsCount: 0,
        text: ""
    }
}

export default {
    data () {
        return {
            show: false,
            participantsDto: participantsDtoFactory(),
            page: firstPage,
            loading: false,
            messageDto: null,
        }
    },

    methods: {
        hasLength,
        showModal(messageDto) {
            this.show = true;
            this.messageDto = messageDto;
            this.loadData();
        },
        closeModal() {
            this.show = false;
            this.participantsDto = participantsDtoFactory();
            this.loading = false;
            this.messageDto = null;
        },
        loadData() {
            if (!this.show) {
                return
            }
            this.loading = true;

            return axios.get(`/api/chat/${this.chatId}/message/read/${this.messageDto.messageId}`, {
                    params: {
                        page: this.translatePage(),
                        size: pageSize,
                    },
                })
                .then((response) => {
                    this.participantsDto = response.data;
                }).finally(() => {
                    this.loading = false;
                })

        },
        translatePage() {
            return this.page - 1;
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
        getTotalVisible() {
            if (!this.isMobile()) {
                return 7
            } else if (this.page == firstPage || this.page == this.pagesCount) {
                return 3
            } else {
                return 1
            }
        },
        messageClass() {
            let classes = ['message-item-text'];
            if (this.isMobile()) {
                classes.push('message-item-text-mobile');
            }

            return classes
        },
        messageWrapperClass() {
            let classes = ['pa-0', 'mb-0', 'mt-2', 'mx-4', 'message-item-wrapper'];
            if (this.messageDto?.ownerId && this.messageDto?.ownerId == this.chatStore.currentUser?.id) {
                classes.push('my');
            }
            return classes
        },
        shouldShowMessageText() {
            return hasLength(this.participantsDto.text)
        },
    },
    computed: {
        ...mapStores(useChatStore),
        chatId() {
            return this.$route.params.id
        },
        pagesCount() {
            const count = Math.ceil(this.participantsDto.participantsCount / pageSize);
            // console.debug("Calc pages count", count);
            return count;
        },
        shouldShowPagination() {
            return this.participantsDto != null && this.participantsDto.participantsCount > pageSize
        }
    },

    watch: {
        show(newValue) {
            if (!newValue) {
                this.closeModal();
            }
        },
        page(newValue) {
            if (this.show) {
                console.debug("SettingNewPage", newValue);
                this.participantsDto = participantsDtoFactory();
                this.loadData();
            }
        },
    },
    mounted() {
        bus.on(OPEN_MESSAGE_READ_USERS_DIALOG, this.showModal);
    },
    beforeUnmount() {
        bus.off(OPEN_MESSAGE_READ_USERS_DIALOG, this.showModal);
    },
}
</script>

