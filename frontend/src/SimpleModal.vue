<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="400" persistent>
            <v-card>
                <v-card-title>{{title}}</v-card-title>

                <v-card-text>{{text}}</v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn color="error" class="mr-4" @click="actionFunction()">{{buttonName}}</v-btn>
                    <v-btn class="mr-4" @click="lightClose()">Close</v-btn>
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
                actionFunction: ()=>{}
            }
        },
        methods: {
            showModal(newData) {
                this.$data.title = newData.title;
                this.$data.text = newData.text;
                this.$data.actionFunction = newData.actionFunction;
                this.$data.buttonName = newData.buttonName;
                this.$data.show = true;
            },
            lightClose() {
                this.$data.show = false;
                this.$data.actionFunction = ()=>{};
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