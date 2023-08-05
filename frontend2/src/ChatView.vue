<template>
    <splitpanes class="default-theme">
        <pane>
            <ChatList/>
        </pane>
        <pane>
            <div> right </div>
        </pane>
    </splitpanes>
</template>

<script>
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import ChatList from "@/ChatList.vue";

export default {
    components: { Splitpanes, Pane, ChatList },
}
</script>


<style>

</style>
