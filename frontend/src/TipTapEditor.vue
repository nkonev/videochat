<template>
  <div class="richText">
    <input id="file-input" type="file" style="display: none;" accept="image/*,video/*" />
    <div class="richText__content">
      <editor-content :editor="editor" class="editorContent" />
    </div>
  </div>
</template>

<script>
import {Slice, Fragment, Node, DOMParser} from 'prosemirror-model';
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
import Code from '@tiptap/extension-code';
import {buildImageHandler} from '@/TipTapImage';
import suggestion from './suggestion';
import {hasLength, media_image, media_video} from "@/utils";
import bus, {FILE_UPLOAD_MODAL_START_UPLOADING, PREVIEW_CREATED, OPEN_FILE_UPLOAD_MODAL} from "./bus";
import Video from "@/TipTapVideo";
import { v4 as uuidv4 } from 'uuid';

const empty = "";

const embedUploadFunction = (chatId, fileObj, correlationId) => {
    bus.$emit(OPEN_FILE_UPLOAD_MODAL, null, false, [fileObj], correlationId);
    bus.$emit(FILE_UPLOAD_MODAL_START_UPLOADING);
}

export default {
  components: {
    EditorContent,
  },

  data() {
    return {
      editor: null,
      fileInput: null,
      correlationId: null,
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
    messageTextIsNotEmpty(text) {
       return text && text !== "" && text !== '<p><br></p>' && text !== '<p></p>'
    },
    onUpdateContent() {
      const value = this.getContent();
      this.$emit("input", value);
    },
    setCursorToEnd() {
      this.editor.commands.focus('end')
    },
    addImage() {
        this.fileInput.click();
    },
    setImage(src) {
        this.editor.chain().focus().setImage({ src: src }).run()
    },
    addVideo() {
        this.fileInput.click();
    },
    setVideo(src, previewUrl) {
        this.editor.chain().focus().setVideo({ src: src, poster: previewUrl }).run();
    },
    onPreviewCreated(dto) {
        if (hasLength(this.correlationId) && this.correlationId == dto.correlationId) {
            if (dto.aType == media_video) {
                this.setVideo(dto.url, dto.previewUrl)
            } if (dto.aType == media_image) {
                this.setImage(dto.url)
            }
        }
    }
  },
  mounted() {
    bus.$on(PREVIEW_CREATED, this.onPreviewCreated);

    const imagePluginInstance = buildImageHandler(
    (image) => {
        this.correlationId = uuidv4();
        embedUploadFunction(this.chatId, image, this.correlationId);
    })
        .configure({
            inline: true,
            HTMLAttributes: {
                class: 'image-custom-class',
            },
        });

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
          imagePluginInstance,
          Video,
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
          Code,
      ],
      editorProps: {
          // Preserves newline on text paste.
          // Combined from
          //  https://github.com/ueberdosis/tiptap/issues/775#issuecomment-762971612
          //  and https://discuss.prosemirror.net/t/how-to-preserve-hard-breaks-when-pasting-html-into-a-plain-text-schema/4202/5
          //  and prosemirror-view/src/clipboard.ts parseFromClipboard()
          transformPastedHTML(html) {
              const withP = html.replace(/<br>\\*/g, "</p><p>");
              const rmDuplicatedP = withP.replace(/<p><\/p>/gi, '');
              return rmDuplicatedP;
          }
      },
      content: empty,
      onUpdate: () => this.onUpdateContent(),
    });

    this.fileInput = document.getElementById('file-input');
    this.fileInput.onchange = e => {
      this.correlationId = uuidv4();
      if (e.target.files.length) {
          const file = e.target.files[0];
          embedUploadFunction(this.chatId, file, this.correlationId)
      }
    }
  },

  beforeDestroy() {
    bus.$off(PREVIEW_CREATED, this.onPreviewCreated);
    this.editor.destroy();
    this.fileInput = null;
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
