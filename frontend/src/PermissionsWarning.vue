<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title>Browser permissions</v-card-title>

                <v-card-text>Please enable audio auto-play permissions for this site in your browser preferences. For Safari they resides in Safari -> Settings -> Web Sites -> Auto Play. See https://browserhow.com/how-to-allow-or-block-auto-play-sound-access-in-safari-mac/#how-to-allow-autoplay-sound-on-safari-macos for details</v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn class="mr-4" @click="show=false">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_PERMISSIONS_WARNING_MODAL} from "./bus";

    export default {
        data () {
            return {
                show: false,
            }
        },
        methods: {
            showModal() {
                this.$data.show = true;
            },
        },
        created() {
            bus.$on(OPEN_PERMISSIONS_WARNING_MODAL, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_PERMISSIONS_WARNING_MODAL, this.showModal);
        },
    }
</script>