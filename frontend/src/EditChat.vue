<template>
    <v-row justify="center">
        <v-dialog v-model="show" persistent max-width="290">
            <v-card>
                <v-card-title class="headline">Create chat</v-card-title>
                <v-card-text>Let Google help apps determine location. This means sending anonymous location data to Google, even when no apps are running.</v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn color="green darken-1" text @click="show=false">Disagree</v-btn>
                    <v-btn color="green darken-1" text @click="show=false">Agree</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
export default {
    props: {
        value: Boolean
    },
    computed: {
        show: {
            get () {
                return this.value
            },
            set (value) {
                this.$emit('input', value)
            }
        }
    }
}
</script>