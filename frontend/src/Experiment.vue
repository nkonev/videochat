<template>
    <v-container class="grey lighten-5">
                <v-row
                    :align="'center'"
                    :justify="'center'"
                    class="grey lighten-5"
                    style="height: 300px;"
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
                </v-row>
    </v-container>
</template>