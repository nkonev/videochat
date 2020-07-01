<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="800" persistent>
            <v-card>
                <v-card-title>Create chat</v-card-title>

                <v-container fluid>
                <v-autocomplete
                        v-model="friends"
                        :disabled="isUpdating"
                        :items="people"
                        filled
                        chips
                        color="blue-grey lighten-2"
                        label="Select"
                        item-text="name"
                        item-value="id"
                        multiple
                        :hide-selected="true"
                >
                    <template v-slot:selection="data">
                        <v-chip
                                v-bind="data.attrs"
                                :input-value="data.selected"
                                close
                                @click="data.select"
                                @click:close="removeSelected(data.item)"
                        >
                            <v-avatar left>
                                <v-img :src="data.item.avatar"></v-img>
                            </v-avatar>
                            {{ data.item.name }}
                        </v-chip>
                    </template>
                    <template v-slot:item="data">
                        <v-list-item-avatar>
                            <img :src="data.item.avatar">
                        </v-list-item-avatar>
                        <v-list-item-content>
                            <v-list-item-title v-html="data.item.name"></v-list-item-title>
                        </v-list-item-content>
                    </template>
                </v-autocomplete>
                </v-container>

                <v-card-actions class="pa-4">
                    <v-btn color="primary" class="mr-4" @click="show=false">Create</v-btn>
                    <v-btn color="error" class="mr-4" @click="show=false">Close</v-btn>
                    <v-spacer/>
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
                get() {
                    return this.value
                },
                set(value) {
                    this.$emit('input', value)
                }
            }
        },
        data () {
            const srcs = {
                1: 'https://cdn.vuetifyjs.com/images/lists/1.jpg',
                2: 'https://cdn.vuetifyjs.com/images/lists/2.jpg',
                3: 'https://cdn.vuetifyjs.com/images/lists/3.jpg',
                4: 'https://cdn.vuetifyjs.com/images/lists/4.jpg',
                5: 'https://cdn.vuetifyjs.com/images/lists/5.jpg',
            };

            return {
                friends: [1, 3],
                isUpdating: false,
                people: [
                    { id:1, name: 'Sandra Adams', avatar: srcs[1] },
                    { id:2, name: 'Ali Connors', avatar: srcs[2] },
                    { id:3, name: 'Trevor Hansen', avatar: srcs[3] },
                    { id:4, name: 'Tucker Smith', avatar: srcs[2] },
                    { id:5, name: 'Britta Holt', avatar: srcs[4] },
                    { id:6, name: 'Jane Smith ', avatar: srcs[5] },
                    { id:7, name: 'John Smith', avatar: srcs[1] },
                    { id:8, name: 'Sandra Williams', avatar: srcs[3] },
                ],
            }
        },


        watch: {
            isUpdating (val) {
                if (val) {
                    setTimeout(() => (this.isUpdating = false), 3000)
                }
            },
        },

        methods: {
            removeSelected (item) {
                const index = this.friends.indexOf(item.name)
                if (index >= 0) this.friends.splice(index, 1)
            },
        },
    }
</script>