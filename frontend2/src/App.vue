<template>
  <v-app>
    <v-app-bar>
      <v-app-bar-title>
        <v-icon icon="mdi-circle-slice-6" />
        My app bar {{ appStore.count }}
      </v-app-bar-title>
    </v-app-bar>

    <v-main>
      <v-container fluid class="ma-0 pa-0" style="height: 100%">
        <router-view />
      </v-container>
    </v-main>
  </v-app>
</template>

<script setup>
import '@fontsource/roboto';
import {useAppStore} from "@/store/app";
const appStore = useAppStore();

</script>
