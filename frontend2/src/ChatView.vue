<template>
    <splitpanes class="default-theme" :dbl-click-splitter="false" :style="heightWithoutAppBar">
      <pane size="20">
        <ChatList/>
      </pane>
      <pane>
        <splitpanes class="default-theme" :dbl-click-splitter="false" horizontal>
            <pane>
                <MessageList/>
            </pane>
          <pane size="25">
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
import heightMixin from "@/mixins/heightMixin";

export default {
    mixins: [
      heightMixin(),
    ],
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
