<template>
  <v-card
    class="mx-auto"
    max-width="344"
    variant="outlined"
  >
    <v-card-item>
      <div>
        <div class="text-overline mb-1">
          OVERLINE
        </div>
        <div class="text-h6 mb-1">
          Headline
        </div>
        <div class="text-caption">Greyhound divisely hello coldly fonwderfully</div>
      </div>
      <div>Current Count: {{ appStore.count }}</div>
    </v-card-item>

    <v-card-actions>
      <v-btn variant="outlined" @click="onClick()">
        Button
      </v-btn>
      <v-btn variant="outlined" @click="onClick2()">
        Decrement
      </v-btn>
      <v-btn variant="outlined" @click="onClick3()">
        Increment
      </v-btn>

    </v-card-actions>
  </v-card>
</template>

<script setup>
import router from "@/router/index";
import {useAppStore} from "@/store/app";
const appStore = useAppStore();

const onClick = () => {
  router.push(({ name: 'root'}))
}

function onClick2() {
  appStore.count--
}

function onClick3() {
  appStore.increment()
}
</script>
