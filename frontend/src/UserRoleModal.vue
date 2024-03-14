<template>
        <v-dialog v-model="show" max-width="440" persistent>
            <v-card v-if="show" :title="$vuetify.locale.t('$vuetify.change_role')">

                <v-card-text class="pb-0">
                    <v-select v-if="!loading"
                        :items="allPossibleRoles"
                        label="Select video device"
                        v-model="chosenRole"
                        variant="outlined"
                        density="compact"
                    ></v-select>
                    <v-progress-circular
                        class="ma-4"
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                </v-card-text>

                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="flat" v-if="chosenRole != null" color="primary" @click="changeRole()">{{ $vuetify.locale.t('$vuetify.ok') }}</v-btn>
                    <v-btn variant="flat" color="red" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>

            </v-card>

        </v-dialog>
</template>

<script>
import bus, {
    CHANGE_ROLE_DIALOG,
} from "./bus/bus";
    import axios from "axios";

    export default {
        data () {
            return {
                show: false,
                user: null,
                allPossibleRoles: [],
                chosenRole: null,
                loading: false,
            }
        },
        methods: {
            showModal(user) {
                this.show = true;
                this.user = user;
                this.chosenRole = user.additionalData.roles[0];
                this.requestAllPossibleRolesIfNeed()
            },
            closeModal() {
                this.show = false;
                this.user = null;
                this.chosenRole = null;
            },
            requestAllPossibleRolesIfNeed() {
                if (!this.allPossibleRoles.length) {
                    this.loading = true;
                    axios.get('/api/aaa/user/role').then((response) => {
                        this.allPossibleRoles = response.data;
                    }).finally(() => {
                        this.loading = false;
                    })
                }
            },
            changeRole() {
                this.loading = true;
                axios.put('/api/aaa/user/role', null, { params: {
                        userId: this.user.id,
                        role: this.chosenRole,
                    }}).then(()=>{
                        this.closeModal();
                    }).finally(()=>{
                        this.loading = false;
                    })
            }
        },
        created() {
            bus.on(CHANGE_ROLE_DIALOG, this.showModal);
        },
        destroyed() {
            bus.off(CHANGE_ROLE_DIALOG, this.showModal);
        },
    }
</script>
