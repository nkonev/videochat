<template>
    <v-container fill-height fluid :style="$vuetify.breakpoint.lgAndUp ? 'max-width: 420px' : ''">
        <v-row align="center" justify="center">
            <v-col>
                <v-card>
                    <v-card-title class="d-flex justify-space-around">{{$vuetify.lang.t('$vuetify.welcome_participant', currentUser.login)}}</v-card-title>
                    <v-card-actions  class="d-flex justify-space-around">
                        <v-btn color="primary" @click="createChat()" text>
                            <v-icon>mdi-plus</v-icon>
                            {{ $vuetify.lang.t('$vuetify.new_chat') }}
                        </v-btn>
                        <v-btn @click="findUser()" text>
                            <v-icon>mdi-magnify</v-icon>
                            {{ $vuetify.lang.t('$vuetify.find_user') }}
                        </v-btn>
                    </v-card-actions>
                </v-card>

            </v-col>
        </v-row>
    </v-container>
</template>

<script>
    import {GET_USER} from "@/store";
    import {mapGetters} from "vuex";
    import bus, {OPEN_CHAT_EDIT, OPEN_FIND_USER} from "@/bus";

    export default {
        computed: {
            ...mapGetters({currentUser: GET_USER}),
        },
        methods: {
            createChat() {
                bus.$emit(OPEN_CHAT_EDIT, null);
            },
            findUser() {
                bus.$emit(OPEN_FIND_USER)
            },
        }
    }
</script>
