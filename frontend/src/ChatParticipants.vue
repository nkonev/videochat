<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title>Participants</v-card-title>

                <v-container fluid>
                    <v-list v-if="dto.participants.length > 0">
                        <template v-for="(item, index) in dto.participants">
                            <v-list-item >
                                <v-list-item-avatar v-if="item.avatar">
                                    <v-img :src="item.avatar"></v-img>
                                </v-list-item-avatar>
                                <v-list-item-content>
                                    <v-list-item-title>{{item.login}}<template v-if="item.id == currentUser.id"> (you)</template></v-list-item-title>
                                </v-list-item-content>
                                <v-tooltip bottom v-if="isAdmin() && item.id != currentUser.id">
                                    <template v-slot:activator="{ on, attrs }">
                                        <v-btn v-bind="attrs" v-on="on" icon @click="kickFromVideoCall(item.id)"><v-icon color="error">mdi-block-helper</v-icon></v-btn>
                                    </template>
                                    <span>Kick</span>
                                </v-tooltip>
                                <v-tooltip bottom v-if="item.admin">
                                    <template v-slot:activator="{ on, attrs }">
                                        <span class="pl-1 pr-1">
                                            <v-icon v-bind="attrs" v-on="on">mdi-crown</v-icon>
                                        </span>
                                    </template>
                                    <span>Admin</span>
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
                </v-container>

                <v-card-actions class="pa-4">
                    <v-btn color="error" class="mr-4" @click="show=false">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";
    import bus, {OPEN_INFO_DIALOG} from "./bus";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import {chat_name, videochat_name} from "./routes";

    const dtoFactory = ()=>{
        return {
            id: null,
            name: "",
            participantIds: [ ],
            participants: [ ],
        }
    };

    export default {
        data () {
            return {
                show: false,
                dto: dtoFactory(),
                isLoading: false,
                people: [  ],
            }
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        },

        methods: {
            showModal(chatId) {
                this.$data.show = true;
                const val = chatId;
                if (val) {
                    console.log("Getting info about chat id", val);
                    axios.get('/api/chat/'+val)
                        .then((response) => {
                            this.dto = response.data;
                        });
                } else {
                    this.dto = dtoFactory();
                }

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
            isAdmin(){
                return this.dto.participants.filter(value => value.admin).filter(value => value.id == this.currentUser.id).length != 0
            }
        },
        created() {
            bus.$on(OPEN_INFO_DIALOG, this.showModal);
        },
        destroyed() {
            this.dto = dtoFactory();
            bus.$off(OPEN_INFO_DIALOG, this.showModal);
        },
    }
</script>