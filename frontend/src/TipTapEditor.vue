<template>
  <div class="richText">
    <input id="file-input" type="file" style="display: none;" accept="image/*,video/*,audio/*" multiple="multiple" />
    <div class="richText__content">
      <editor-content :editor="editor" class="editorContent" />
    </div>
  </div>
</template>

<script>
import "prosemirror-view/style/prosemirror.css";
import "./messageBody.styl";
import { Editor, EditorContent } from "@tiptap/vue-3";
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
import {hasLength, media_audio, media_image, media_video} from "@/utils";
import bus, {
    FILE_UPLOAD_MODAL_START_UPLOADING,
    PREVIEW_CREATED,
    OPEN_FILE_UPLOAD_MODAL,
    MEDIA_LINK_SET,
    EMBED_LINK_SET
} from "./bus/bus";
import Video from "@/TipTapVideo";
import Audio from "@/TipTapAudio";
import Iframe from '@/TipTapIframe';
import { v4 as uuidv4 } from 'uuid';

const empty = "";

const embedUploadFunction = (chatId, files, correlationId, fileItemId, shouldAddDateToTheFilename) => {
    bus.emit(OPEN_FILE_UPLOAD_MODAL, {showFileInput: true, fileItemUuid: fileItemId, shouldSetFileUuidToMessage: true, predefinedFiles: files, correlationId: correlationId, shouldAddDateToTheFilename: shouldAddDateToTheFilename});
    bus.emit(FILE_UPLOAD_MODAL_START_UPLOADING);
}

const domParser = new DOMParser();

export default {
  components: {
    EditorContent,
  },

  data() {
    return {
      editor: null,
      fileInput: null,
      correlationId: null,
      preallocatedCandidateFileItemId: null,
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
      this.$emit("myinput", value);
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
    addAudio() {
        this.fileInput.click();
    },
    setVideo(src, previewUrl) {
        this.editor.chain().focus().setVideo({ src: src, poster: previewUrl }).run();
    },
    setAudio(src) {
        this.editor.chain().focus().setAudio({ src: src }).run();
    },
    setIframe(url) {
      if (url) {
          this.editor.chain().focus().setIframe({ src: url }).run()
      }
    },
    regenerateNewFileItemUuid() {
      this.preallocatedCandidateFileItemId = uuidv4();
    },
    resetFileItemUuid() {
      this.preallocatedCandidateFileItemId = null;
    },
    onPreviewCreated(dto) {
        if (hasLength(this.correlationId) && this.correlationId == dto.correlationId) {
            if (dto.aType == media_video) {
                this.setVideo(dto.url, dto.previewUrl)
            } else if (dto.aType == media_image) {
                this.setImage(dto.url)
            } else if (dto.aType == media_audio) {
                this.setAudio(dto.url)
            }
        }
    },
    onMediaLinkSet({link, mediaType}) {
        if (mediaType == media_video) {
            this.setVideo(link)
        } else if (mediaType == media_image) {
            this.setImage(link)
        } else if (mediaType == media_audio) {
            this.setAudio(link)
        }
    },
    onEmbedLinkSet(link) {
        console.log("onEmbedLinkSet", link);
        if (link && !link.startsWith("http")) {
            const htmlDoc = domParser.parseFromString(link, 'text/html');
            const iframes = htmlDoc.getElementsByTagName('iframe');
            if (iframes.length == 1) {
                const iframe = iframes[0];
                link = iframe.src;
            }
        }

        this.setIframe(link);
    },
    addText(text) {
        this.editor.commands.insertContent(text)
    },
  },
  mounted() {
    bus.on(PREVIEW_CREATED, this.onPreviewCreated);
    bus.on(MEDIA_LINK_SET, this.onMediaLinkSet);
    bus.on(EMBED_LINK_SET, this.onEmbedLinkSet);
    this.regenerateNewFileItemUuid();

    const imagePluginInstance = buildImageHandler(
    (image, shouldAddDateToTheFilename) => {
        this.correlationId = uuidv4();
        embedUploadFunction(this.chatId, [image], this.correlationId, this.preallocatedCandidateFileItemId, shouldAddDateToTheFilename);
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
                return this.$vuetify.locale.t('$vuetify.message_edit_placeholder')
              },
          }),
          Text,
          imagePluginInstance,
          Video,
          Audio,
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
          Iframe,
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
          },
      },
      content: empty,
      onUpdate: () => this.onUpdateContent(),
    });

    this.$nextTick(()=>{
        this.fileInput = document.getElementById('file-input');
        // triggered when we upload image or video after this.fileInput.click()
        this.fileInput.onchange = e => {
          this.correlationId = uuidv4();
          if (e.target.files.length) {
              const files = Array.from(e.target.files);
              embedUploadFunction(this.chatId, files, this.correlationId, this.preallocatedCandidateFileItemId)
          }
        }
    })
  },

  beforeUnmount() {
    bus.off(PREVIEW_CREATED, this.onPreviewCreated);
    bus.off(MEDIA_LINK_SET, this.onMediaLinkSet);
    bus.off(EMBED_LINK_SET, this.onEmbedLinkSet);
    this.resetFileItemUuid();

    this.editor.destroy();
    if (this.fileInput) {
      this.fileInput.onchange = null;
    }
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

</style>
