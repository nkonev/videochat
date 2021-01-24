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
                                    <v-list-item-title>{{item.login}}</v-list-item-title>
                                </v-list-item-content>
                                <v-tooltip bottom v-if="item.admin">
                                    <template v-slot:activator="{ on, attrs }">
                                        <v-icon v-bind="attrs" v-on="on">mdi-crown</v-icon>
                                    </template>
                                    <span>Admin</span>
                                </v-tooltip>
                                <v-list-item-action>
                                    <v-btn icon @click="inviteToVideoCall(item.id)"><v-icon color="success">mdi-phone</v-icon></v-btn>
                                </v-list-item-action>
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
                search: null,
                dto: dtoFactory(),
                isLoading: false,
                people: [  ],
            }
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
                axios.post(`/api/chat/${this.dto.id}/video/invite?userId=${userId}`)
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