<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480" scrollable>
            <v-card :title="$vuetify.locale.t('$vuetify.users_read')">
                <v-card-text class="ma-0 pa-0">
                    <v-container class="px-6 pt-0" v-html="participantsDto.text"></v-container>
                    <v-list v-if="participantsDto.participants && participantsDto.participants.length > 0">
                        <template v-for="(item, index) in participantsDto.participants">
                            <v-list-item class="pl-2 ml-1 pr-0 mr-3 mb-1 mt-1" :prepend-avatar="item.avatar" @click.prevent="onParticipantClick(item)" :href="getLink(item)">
                              <v-list-item-title :class="!isMobile() ? 'mr-2' : ''">
                                {{item.login + (item.id == chatStore.currentUser.id ? $vuetify.locale.t('$vuetify.you_brackets') : '' )}}
                              </v-list-item-title>
                            </v-list-item>
                            <v-divider></v-divider>
                        </template>
                    </v-list>
                    <template v-else-if="!loading">
                        <v-card-text>{{ $vuetify.locale.t('$vuetify.participants_not_found') }}</v-card-text>
                    </template>

                    <v-progress-circular
                        class="ma-4 pa-4"
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
                    <v-btn color="error" class="my-1" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
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
            participantsPage: firstPage,
            loading: false,
            messageDto: null,
        }
    },

    methods: {
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

            return axios.get('/api/chat/'+this.messageDto.chatId+'/message/read/'+this.messageDto.messageId, {
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
            return this.participantsPage - 1;
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
    computed: {
        ...mapStores(useChatStore),
        chatId() {
            return this.$route.params.id
        },
        participantsPagesCount() {
            const count = Math.ceil(this.participantsDto.participantsCount / pageSize);
            console.debug("Calc pages count", count);
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
    },
    created() {
        bus.on(OPEN_MESSAGE_READ_USERS_DIALOG, this.showModal);
    },
    destroyed() {
        bus.off(OPEN_MESSAGE_READ_USERS_DIALOG, this.showModal);
    },
}
</script>

<style lang="stylus">
.white-colored {
    color white !important
}
</style>
