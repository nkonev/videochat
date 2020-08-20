<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title>Choose avatar</v-card-title>

                <v-container fluid>
                    <v-file-input accept="image/*" label="File input"></v-file-input>
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

    export default {
        data () {
            return {
                show: false,
            }
        },

        methods: {
            saveAvatar() {
                axios.put(`/api/chat`, {})
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