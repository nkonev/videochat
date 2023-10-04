<template>
  <v-app>
    <v-main>
      <v-container fluid class="ma-0 pa-0" style="height: 100%; width: 100%">
        header
        <router-view />
      </v-container>

    </v-main>

  </v-app>
</template>

<script>
import 'typeface-roboto'; // More modern versions turn out into almost non-bold font in Firefox
import BlogList from "@/BlogList.vue";

export default {
  components: {
    BlogList
  }
}

</script>
