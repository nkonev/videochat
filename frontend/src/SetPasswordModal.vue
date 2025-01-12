<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480" persistent>
            <v-card :title="$vuetify.locale.t('$vuetify.set_password_for', userName)" :disabled="loading">
                <v-progress-linear
                  :active="loading"
                  :indeterminate="loading"
                  absolute
                  bottom
                  color="primary"
                ></v-progress-linear>

                <v-card-text>
                  <v-text-field
                      v-model="password"
                      :append-icon="showInputablePassword ? 'mdi-eye' : 'mdi-eye-off'"
                      @click:append="showInputablePassword = !showInputablePassword"
                      :label="$vuetify.locale.t('$vuetify.password')"
                      required
                      :type="showInputablePassword ? 'text' : 'password'"
                      :disabled="loading"
                      variant="underlined"
                  ></v-text-field>

                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-spacer></v-spacer>
                    <v-btn color="primary" variant="flat" @click="onSet()">{{ $vuetify.locale.t('$vuetify.ok') }}</v-btn>
                    <v-btn color="red" variant="flat" @click="onClose()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_SET_PASSWORD_MODAL} from "./bus/bus";
    import axios from "axios";

    export default {
        data () {
            return {
                show: false,
                loading: false,
                password: null,
                showInputablePassword: false,
                userId: null,
                userName: null,
            }
        },
        methods: {
            showModal(newData) {
                this.$data.show = true;
                this.$data.userId = newData.userId;
                this.$data.userName = newData.userName;
            },
            onClose() {
                this.$data.show = false;
                this.$data.loading = false;
                this.$data.showInputablePassword = false;
                this.$data.loading = false;
                this.$data.password = null;
                this.$data.userId = null;
                this.$data.userName = null;
            },
            onSet() {
              this.$data.loading = true;
              axios.put(`/api/aaa/user/${this.$data.userId}/password`, {password: this.password})
                  .finally(()=>{
                    this.onClose();
                  })
            },
        },
        mounted() {
            bus.on(OPEN_SET_PASSWORD_MODAL, this.showModal);
        },
        beforeUnmount() {
            bus.off(OPEN_SET_PASSWORD_MODAL, this.showModal);
        },
    }
</script>
