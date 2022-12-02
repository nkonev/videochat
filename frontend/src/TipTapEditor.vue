<template>
  <div class="richText">
    <input id="image-file-input" type="file" style="display: none;" accept="image/*" />
    <div class="richText__content">
      <editor-content :editor="editor" class="editorContent" />
    </div>
  </div>
</template>

<script>
import "prosemirror-view/style/prosemirror.css";
import "./message.styl";
import { Editor, EditorContent } from "@tiptap/vue-2";
import Document from "@tiptap/extension-document";
import Paragraph from "@tiptap/extension-paragraph";
import Italic from "@tiptap/extension-italic";
import Bold from "@tiptap/extension-bold";
import Strike from '@tiptap/extension-strike';
import Underline from '@tiptap/extension-underline';
import Link from '@tiptap/extension-link'
import Text from "@tiptap/extension-text";
import History from '@tiptap/extension-history';
import Placeholder from '@tiptap/extension-placeholder';
import TextStyle from "@tiptap/extension-text-style";
import Color from '@tiptap/extension-color';
import Highlight from "@tiptap/extension-highlight";
import Mention from '@tiptap/extension-mention';
import axios from "axios";
import {buildImageHandler} from '@/TipTapImage';
import suggestion from './suggestion';

const empty = "";

const embedUploadFunction = (chatId, fileObj) => {
    const formData = new FormData();
    formData.append('embed_file_header', fileObj);
    return axios.post('/api/storage/'+chatId+'/embed', formData)
        .then((result) => {
            let url = result.data.relativeUrl; // Get url from response
            console.debug("got embed url", url);
            return url;
        })
}

export default {
  components: {
    EditorContent,
  },

  data() {
    return {
      editor: null,
      imageFileInput: null,
    };
  },

  computed: {
    chatId() {
      return this.$route.params.id
    },
  },
  methods: {
    setContent(value) {
        // https://tiptap.dev/api/commands/set-content
        this.editor.commands.setContent(value, false, { preserveWhitespace: "full" });
    },
    clearContent() {
      this.editor.commands.setContent(empty, false);
    },
    getContent() {
      const value = this.editor.getHTML();
      if (this.messageTextIsNotEmpty(value)) {
        return value
      } else {
        return empty
      }
    },
    addImage() {
      this.imageFileInput.click();
    },
    messageTextIsNotEmpty(text) {
       return text && text !== "" && text !== '<p><br></p>' && text !== '<p></p>'
    },
    onUpdateContent() {
      const value = this.getContent();
      this.$emit("input", value);
    },
    setCursorToEnd() {
      this.editor.commands.focus('end')
    }
  },
  mounted() {
    this.editor = new Editor({
      // https://github.com/ueberdosis/tiptap/issues/873#issuecomment-730147217
      parseOptions: {
        preserveWhitespace: "full",
      },
      autofocus: true,
      enablePasteRules: false,
      injectCSS: false,
      enableInputRules: false,
      extensions: [
          Document,
          Paragraph,
          History,
          Placeholder.configure({
              placeholder: ({ node }) => {
                return this.$vuetify.lang.t('$vuetify.message_edit_placeholder')
              },
          }),
          Text,
          buildImageHandler((image) => embedUploadFunction(this.chatId, image)).configure({
              inline: true,
              HTMLAttributes: {
                  class: 'image-custom-class',
              },
          }),
          Italic,
          Bold,
          Strike,
          Underline,
          Link.configure({
              openOnClick: false,
              linkOnPaste: false
          }),
          TextStyle,
          Color,
          Highlight.configure({ multicolor: true }),
          Mention.configure({
              HTMLAttributes: {
                  class: 'mention',
              },
              suggestion: suggestion(this.chatId),
          }),
      ],
      content: empty,
      onUpdate: () => this.onUpdateContent(),
    });

    this.imageFileInput = document.getElementById('image-file-input');
    this.imageFileInput.onchange = e => {
      if (e.target.files.length) {
          const file = e.target.files[0];
          embedUploadFunction(this.chatId, file)
              .then(url => {
                  this.editor.chain().focus().setImage({ src: url }).run()
              })
      }
    }
  },

  beforeDestroy() {
    this.editor.destroy();
    this.imageFileInput = null;
  },
};
</script>
<style>
.editorContent {
    height: 100%;
}
.ProseMirror {
    height: 100%;
}
.richText {
  display: flex;
  flex-direction: column;
  color: #0d0d0d;
  background-color: #fff;
  border: 1px dashed #0D0D0D;
  height: 100%;
  overflow-y: auto;
}
.richText__header {
  display: flex;
  align-items: center;
  flex: 0 0 auto;
  flex-wrap: wrap;
  padding: 0.25rem;
  border-bottom: 3px solid #0D0D0D;
}

.richText__content {
  padding: 6px 6px;
  flex: 1 1 auto;
  overflow-x: hidden;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
}

.richText__content p {
    margin-bottom: unset
}

.richText__footer {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  border-top: 3px solid #0D0D0D;
  font-size: 12px;
  font-weight: 600;
  color: #0d0d0d;
  white-space: nowrap;
  padding: 0.25rem 0.75rem;
}

.richText__content :focus-visible {
  outline: none;
}

.ProseMirror p.is-editor-empty:first-child::before {
    content: attr(data-placeholder);
    float: left;
    color: #a9a9a9;
    pointer-events: none;
    height: 0;
}

.ProseMirror img {
    max-width: 100%;
    height: auto;
}

</style>