<template>
        <v-dialog v-model="show" max-width="440" persistent>
            <v-card v-if="show" :title="$vuetify.locale.t('$vuetify.override_permissions_for', user?.login)">

                <v-card-text class="pb-0">
                    <v-select v-if="!loading"
                        :items="allPossiblePermissions"
                        label="Override add permissions"
                        v-model="addPermissions"
                        variant="outlined"
                        density="compact"
                        color="primary"
                        multiple
                    ></v-select>
                    <v-select v-if="!loading"
                              :items="allPossiblePermissions"
                              label="Override remove permissions"
                              v-model="removePermissions"
                              variant="outlined"
                              density="compact"
                              color="primary"
                              multiple
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
                    <v-btn variant="flat" color="primary" @click="changePermissions()">{{ $vuetify.locale.t('$vuetify.ok') }}</v-btn>
                    <v-btn variant="flat" color="red" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>

            </v-card>

        </v-dialog>
</template>

<script>
    import bus, {
      CHANGE_PERMISSIONS_DIALOG,
    } from "./bus/bus";
    import axios from "axios";

    export default {
        data () {
            return {
                show: false,
                user: null,
                allPossiblePermissions: [],
                addPermissions: [],
                removePermissions: [],
                loading: false,
            }
        },
        methods: {
            showModal(user) {
                this.show = true;
                this.user = user;

                this.loading = true;
                axios.get(`/api/aaa/user/permission/${user.id}`).then((response) => {
                  this.addPermissions = response.data.addPermissions;
                  this.removePermissions = response.data.removePermissions;
                }).finally(() => {
                  this.loading = false;

                  this.requestAllPossiblePermissionsIfNeed()
                })
            },
            closeModal() {
                this.show = false;
                this.user = null;
                this.addPermissions = [];
                this.removePermissions = [];
            },
            requestAllPossiblePermissionsIfNeed() {
                if (!this.allPossiblePermissions.length) {
                    this.loading = true;
                    axios.get('/api/aaa/user/permission').then((response) => {
                        this.allPossiblePermissions = response.data;
                    }).finally(() => {
                        this.loading = false;
                    })
                }
            },
            changePermissions() {
                this.loading = true;
                axios.put('/api/aaa/user/permission', {
                    userId: this.user.id,
                    addPermissions: this.addPermissions,
                    removePermissions: this.removePermissions,
                }).then(()=>{
                        this.closeModal();
                    }).finally(()=>{
                        this.loading = false;
                    })
            }
        },
        mounted() {
            bus.on(CHANGE_PERMISSIONS_DIALOG, this.showModal);
        },
        beforeUnmount() {
            bus.off(CHANGE_PERMISSIONS_DIALOG, this.showModal);
        },
    }
</script>
