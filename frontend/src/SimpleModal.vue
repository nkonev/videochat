<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="400" persistent>
            <v-card>
                <v-card-title>{{title}}</v-card-title>

                <v-card-text>{{text}}</v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn color="error" class="mr-4" @click="actionFunction()">{{buttonName}}</v-btn>
                    <v-btn class="mr-4" @click="lightClose()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_SIMPLE_MODAL, CLOSE_SIMPLE_MODAL} from "./bus";

    export default {
        data () {
            return {
                show: false,
                title: "",
                text: "",
                buttonName: "",
                actionFunction: ()=>{},
                cancelFunction: null,
            }
        },
        methods: {
            showModal(newData) {
                this.$data.title = newData.title;
                this.$data.text = newData.text;
                this.$data.actionFunction = newData.actionFunction;
                this.$data.cancelFunction = newData.cancelFunction;
                this.$data.buttonName = newData.buttonName;
                this.$data.show = true;
            },
            lightClose() {
                if (this.$data.cancelFunction) {
                    this.$data.cancelFunction();
                }
                this.$data.show = false;
                this.$data.actionFunction = ()=>{};
                this.$data.cancelFunction = null;
            },
            hideModal() {
                this.lightClose();
                this.$data.title = "";
                this.$data.text = "";
                this.$data.buttonName = "";
            },
        },
        created() {
            bus.$on(OPEN_SIMPLE_MODAL, this.showModal);
            bus.$on(CLOSE_SIMPLE_MODAL, this.hideModal)
        },
        destroyed() {
            bus.$off(OPEN_SIMPLE_MODAL, this.showModal);
            bus.$off(CLOSE_SIMPLE_MODAL, this.hideModal)
        },
    }
</script>