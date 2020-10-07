<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title>Participants</v-card-title>

                <v-container fluid>
                    <v-list>
                        <template v-for="(item, index) in people">
                            <v-list-item>
                                <v-list-item-avatar v-if="item.avatar">
                                    <v-img :src="item.avatar"></v-img>
                                </v-list-item-avatar>
                                <v-list-item-content>
                                    <v-list-item-title>{{item.login}}</v-list-item-title>
                                </v-list-item-content>
                            </v-list-item>
                            <v-divider></v-divider>
                        </template>
                    </v-list>
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
                        }).then(()=>{
                        axios.get('/api/user/list', {
                            params: {userId: [...this.dto.participantIds] + ''}
                        }).then((response) => {
                            this.people = response.data;
                        })
                    })
                } else {
                    this.dto = dtoFactory();
                }

            },

        },
        created() {
            bus.$on(OPEN_INFO_DIALOG, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_INFO_DIALOG, this.showModal);
        },
    }
</script>