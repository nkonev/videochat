<template>
    <v-container class="grey lighten-5">
                <v-row
                    :align="'stretch'"
                    class="grey lighten-5"
                    style="height: 100%; display: flex"
                >
                    <v-card
                        v-for="n in 3"
                        :key="n"
                        class="ma-3 pa-6"
                        outlined
                        tile
                    >
                        Column
                    </v-card>
                    <textarea></textarea>
                </v-row>
    </v-container>
</template>