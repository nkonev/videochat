<template>
    <splitpanes class="default-theme">
        <pane>
            <ChatList/>
        </pane>
        <pane>
            <MessageList/>
        </pane>
    </splitpanes>
</template>

<script>
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import ChatList from "@/ChatList.vue";
import MessageList from "@/MessageList.vue";

export default {
    components: { Splitpanes, Pane, ChatList, MessageList },
}
</script>


<style>

</style>
