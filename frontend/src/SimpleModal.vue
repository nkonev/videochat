<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480" persistent>
            <v-card :title="title" :disabled="loading">
                <v-progress-linear
                  :active="loading"
                  :indeterminate="loading"
                  absolute
                  bottom
                  color="primary"
                ></v-progress-linear>

                <v-card-text v-html="text"></v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-spacer></v-spacer>
                    <v-btn color="red" variant="flat" @click="actionFunction(this)">{{buttonName}}</v-btn>
                    <v-btn variant="outlined" @click="lightClose()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_SIMPLE_MODAL, CLOSE_SIMPLE_MODAL} from "./bus/bus";

    export default {
        data () {
            return {
                show: false,
                loading: false,
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
                this.$data.loading = false;
            },
            hideModal() {
                this.$data.show = false;
                this.$data.actionFunction = ()=>{};
                this.$data.cancelFunction = null;
                this.$data.loading = false;

                this.$data.title = "";
                this.$data.text = "";
                this.$data.buttonName = "";
            },
        },
        mounted() {
            bus.on(OPEN_SIMPLE_MODAL, this.showModal);
            bus.on(CLOSE_SIMPLE_MODAL, this.hideModal)
        },
        beforeUnmount() {
            bus.off(OPEN_SIMPLE_MODAL, this.showModal);
            bus.off(CLOSE_SIMPLE_MODAL, this.hideModal)
        },
    }
</script>
