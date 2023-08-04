<template>
  <v-list>
    <v-list-item
      v-for="n in 5"
      :key="n"
      :title="`Item ${ n }`"
      prepend-icon="mdi-forum"
      link
    >
    </v-list-item>
  </v-list>

</template>
