<template>
  <v-app>
    <v-app-bar>
      <v-app-bar-title>
        <v-icon icon="mdi-circle-slice-6" />
        My app bar {{ appStore.count }}
      </v-app-bar-title>
      <v-btn
        color="primary"
        min-width="228"
        size="x-large"
        variant="flat"
        :to="{ name: 'list'}"
      >
        <v-icon
          icon="mdi-speedometer"
          size="large"
          start
        />
        To list
      </v-btn>
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
