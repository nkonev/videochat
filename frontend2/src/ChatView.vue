<template>
    <splitpanes class="default-theme" :dbl-click-splitter="false">
      <pane size="30">
        <ChatList/>
      </pane>
      <pane>
        <splitpanes class="default-theme" :dbl-click-splitter="false" horizontal>
            <pane>
                <MessageList/>
            </pane>
          <pane>
            <MessageEdit :chatId="this.chatId"/>
          </pane>
        </splitpanes>
      </pane>
    </splitpanes>
</template>

<script>
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import ChatList from "@/ChatList.vue";
import MessageList from "@/MessageList.vue";
import MessageEdit from "@/MessageEdit.vue";

export default {
    components: {
      Splitpanes,
      Pane,
      ChatList,
      MessageList,
      MessageEdit,
    },
    computed: {
      chatId() {
        return this.$route.params.id
      },
    }
}
</script>


<style>

</style>
