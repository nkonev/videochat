<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title>Choose avatar</v-card-title>

                <v-container fluid>
                    <v-file-input counter :rules="rules" accept="image/*" label="File input" @change="onFileChange"></v-file-input>
                </v-container>

                <v-card-actions class="pa-4">
                    <v-btn color="primary" class="mr-4" @click="saveAvatar()">Choose</v-btn>
                    <v-btn color="error" class="mr-4" @click="show=false">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";
    import bus, {OPEN_CHOOSE_AVATAR} from "./bus";
    import {FETCH_USER_PROFILE} from "./store";
    import Vue from 'vue'

    export default {
        data() {
            return {
                show: false,
                rules: [
                    value => !value || value.size < 10000000 || 'Avatar size should be less than 10 MB!',
                ],
                formData: null,
            }
        },

        methods: {
            saveAvatar() {
                const config = {
                    headers: { 'content-type': 'multipart/form-data' }
                }
                axios
                    .post(`/api/storage/avatar`, this.formData, config)
                    .then(({ data }) => {
                        return axios.patch(`/api/profile`, {avatar: data.relativeUrl})
                    }).then(value => {
                        console.log("PATCH result", value);
                        this.$store.dispatch(FETCH_USER_PROFILE);
                        this.show = false;
                    })
            },
            getFormData(files){
                const data = new FormData();
                [...files].forEach(file => {
                    data.append('data', file, file.name); // currently only one file at a time
                });
                return data;
            },
            onFileChange(file) {
                console.log("On change", file);
                this.formData = this.getFormData([file]);
            },
            showModal() {
                console.log("Reseiving open avatar");
                this.$data.show = true;
            },
        },
        created() {
            bus.$on(OPEN_CHOOSE_AVATAR, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_CHOOSE_AVATAR, this.showModal);
        },
    }
</script>