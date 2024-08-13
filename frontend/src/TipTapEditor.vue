<template>
  <div class="richText">
    <input id="file-input" type="file" style="display: none;" accept="image/*,video/*,audio/*" multiple="multiple" />
    <div :class="editorContainer()">
      <editor-content :editor="editor" class="editorContent" />
    </div>
  </div>
</template>

<script>
import "prosemirror-view/style/prosemirror.css";
import "./messageBody.styl";
import { Editor, EditorContent } from "@tiptap/vue-3";
import { Extension } from "@tiptap/core";
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
import CodeBlock from '@tiptap/extension-code-block';
import BulletList from '@tiptap/extension-bullet-list';
import OrderedList from '@tiptap/extension-ordered-list';
import ListItem from '@tiptap/extension-list-item';
import {buildImageHandler} from '@/TipTapImage';
import suggestion from './suggestion';
import {hasLength, media_audio, media_image, media_video} from "@/utils";
import bus, {
    FILE_UPLOAD_MODAL_START_UPLOADING,
    PREVIEW_CREATED,
    OPEN_FILE_UPLOAD_MODAL,
    MEDIA_LINK_SET,
    EMBED_LINK_SET,
} from "./bus/bus";
import Video from "@/TipTapVideo";
import Audio from "@/TipTapAudio";
import Iframe from '@/TipTapIframe';
import { v4 as uuidv4 } from 'uuid';
import {getStoredMessageEditNormalizeText, getTreatNewlinesAsInHtml} from "@/store/localStore.js";
import {mapStores} from "pinia";
import {fileUploadingSessionTypeMedia, useChatStore} from "@/store/chatStore.js";
import {mergeAttributes} from "@tiptap/core";
import {profile} from "@/router/routes.js";

const empty = "";

const embedUploadFunction = (chatId, files, correlationId, fileItemId, shouldAddDateToTheFilename) => {
    bus.emit(OPEN_FILE_UPLOAD_MODAL, {showFileInput: true, fileItemUuid: fileItemId, shouldSetFileUuidToMessage: true, predefinedFiles: files, correlationId: correlationId, shouldAddDateToTheFilename: shouldAddDateToTheFilename, fileUploadingSessionType: fileUploadingSessionTypeMedia});
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
      fileItemUuid: null,
    };
  },

  computed: {
    chatId() {
      return this.$route.params.id
    },
    ...mapStores(useChatStore),
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
      if (this.messageHasMeaningfulContent(value)) {
          return value
      } else {
          return empty
      }
    },
    messageHasMeaningfulContent(htmlString) {
        const htmlDoc = domParser.parseFromString(htmlString, 'text/html');

        const videos = htmlDoc.getElementsByTagName('video');
        const audios = htmlDoc.getElementsByTagName('audio');
        const iframes = htmlDoc.getElementsByTagName('iframe');
        const images = htmlDoc.getElementsByTagName('img');
        const textContent = htmlDoc.documentElement.textContent.trim();

        return htmlDoc && (
            textContent !== ""
            || videos.length > 0
            || audios.length > 0
            || iframes.length > 0
            || images.length > 0
        )
    },
    onUpdateContent() {
      const value = this.getContent();
      this.$emit("myinput", value);
    },
    setCursorToEnd() {
        this.$nextTick(()=>{
            setTimeout(()=>{
                this.editor.commands.focus('end')
            }, 1)
        })
    },
    addImage() {
        this.fileInput.click();
    },
    setImage(src, previewUrl) {
        if (hasLength(src) && hasLength(previewUrl)) {
            this.editor.chain().focus().setImage({ src: previewUrl, original: src }).run()
        } else {
            this.editor.chain().focus().setImage({ src: src }).run()
        }
    },
    addVideo() {
        this.fileInput.click();
    },
    addAudio() {
        this.fileInput.click();
    },
    setVideo(src, previewUrl) {
        if (hasLength(src) && hasLength(previewUrl)) {
            this.editor.chain().focus().setVideo({src: previewUrl, original: src}).run()
        } else {
            this.editor.chain().focus().setVideo({src: src}).run()
        }
    },
    setAudio(src) {
        this.editor.chain().focus().setAudio({ src: src }).run();
    },
    setIframe(obj) {
      if (obj.src) {
          this.editor.chain().focus().setIframe(obj).run()
      }
    },
    resetFileItemUuid() {
        this.fileItemUuid = null;
    },
    setCorrelationId(newCorrelationId) {
      this.chatStore.correlationId = newCorrelationId;
    },
    onPreviewCreated(dto) {
        if (hasLength(this.chatStore.correlationId) && this.chatStore.correlationId == dto.correlationId) {
            if (dto.aType == media_video) {
                this.setVideo(dto.url, dto.previewUrl)
            } else if (dto.aType == media_image) {
                this.setImage(dto.url, dto.previewUrl)
            } else if (dto.aType == media_audio) {
                this.setAudio(dto.url)
            }
            if (this.chatStore.sendMessageAfterMediaInsert && this.chatStore.fileUploadingSessionType == fileUploadingSessionTypeMedia) {
                this.$emit("sendMessage");
                this.chatStore.resetSendMessageAfterMediaInsertRoutine();
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

                this.setIframe({src: iframe.src, width: iframe.width, height: iframe.height, allowfullscreen: iframe.getAttribute('allowfullscreen')});
                return
            }
        }

        this.setIframe({src: link});
    },
    addText(text) {
      this.editor.commands.insertContent(text)
    },
    setFileItemUuid(fileItemUuid) {
        this.fileItemUuid = fileItemUuid;
    },
    editorContainer() {
      const ret = ["richText__content"];
      if (this.isMobile()) {
        ret.push("richText__content__mobile")
      }
      return ret;
    },
    getEditor() {
      return this.editor
    },
  },
  mounted() {
    bus.on(PREVIEW_CREATED, this.onPreviewCreated);
    bus.on(MEDIA_LINK_SET, this.onMediaLinkSet);
    bus.on(EMBED_LINK_SET, this.onEmbedLinkSet);

    const imagePluginInstance = buildImageHandler(
    (image, shouldAddDateToTheFilename) => {
        this.setCorrelationId(uuidv4());
        embedUploadFunction(this.chatId, [image], this.chatStore.correlationId, this.fileItemUuid, shouldAddDateToTheFilename);
    })
        .configure({
            inline: true,
            HTMLAttributes: {
                class: 'image-custom-class',
            },
        });

    // We use Ctrl+Enter for sending a message
    // https://github.com/ueberdosis/tiptap/issues/2195#issuecomment-979171024
    // https://github.com/ueberdosis/tiptap/discussions/2948
    const DisableCtrlEnter = Extension.create({
      addKeyboardShortcuts() {
          return {
              'Mod-Enter': () => true,
          };
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
                  if (this.chatStore.shouldShowSendMessageButtons) {
                      return this.$vuetify.locale.t('$vuetify.message_edit_placeholder')
                  } else {
                    return this.$vuetify.locale.t('$vuetify.message_edit_placeholder_short')
                  }
              },
          }),
          Text,
          Video,
          imagePluginInstance,
          Audio,
          Italic,
          Bold,
          Strike,
          Underline,
          Link.configure({
              openOnClick: false,
              linkOnPaste: true
          }),
          TextStyle,
          Color,
          Highlight.configure({ multicolor: true }),
          Mention.configure({
              HTMLAttributes: {
                  class: 'mention',
              },
              suggestion: suggestion(this),
              renderHTML({ options, node }) {
                  if (node.attrs.id > 0) { // real users have id > 0, all and here have < 0
                      return [
                          "a",
                          mergeAttributes({ href: `${profile}/${node.attrs.id}` }, options.HTMLAttributes),
                          `${options.suggestion.char}${node.attrs.label}`,
                      ];
                  } else {
                      return [
                          "span",
                          options.HTMLAttributes,
                          `${options.suggestion.char}${node.attrs.label}`,
                      ];
                  }
              },
          }),
          Code,
          CodeBlock,
          Iframe,
          ListItem,
          BulletList,
          OrderedList,
          DisableCtrlEnter,
      ],
      editorProps: {
          // Preserves newline on text paste.
          // Combined from
          //  https://github.com/ueberdosis/tiptap/issues/775#issuecomment-762971612
          //  and https://discuss.prosemirror.net/t/how-to-preserve-hard-breaks-when-pasting-html-into-a-plain-text-schema/4202/5
          //  and prosemirror-view/src/clipboard.ts parseFromClipboard()
          transformPastedHTML(html) {
            const normalize = (input) => {
                return domParser.parseFromString(input, 'text/html').documentElement.getElementsByTagName('body')[0].innerHTML
            }

            if (!getStoredMessageEditNormalizeText()) {
                return html
            }

            let str;
            if (getTreatNewlinesAsInHtml()) {
              str = html.replace(/(\r\n\r\n|\r\r|\n\n)/g, "<p>");
              str = str.replace(/(\r\n|\r|\n)/g, " ");
            } else {
              str = html.replace(/(\r\n|\r|\n)/g, "<p>");
            }
            const fixedSpaces = str.replace(/&#32;/g, "&nbsp;");
            const withP = fixedSpaces.replace(/<br[^>]*>/g, "<p>");
            const rmDuplicatedP = withP.replace(/<p[^>]*><\/p>/gi, '');
            const normalized = normalize(rmDuplicatedP)
            console.debug("html=", html, ", str=", str, "fixedSpaces=", fixedSpaces, ", withP=", withP, ", rmDuplicatedP=", rmDuplicatedP, "normalized=", normalized)
            return normalized;
          },
      },
      content: empty,
      onUpdate: () => this.onUpdateContent(),
    });

    this.$nextTick(()=>{
        this.fileInput = document.getElementById('file-input');
        // triggered when we upload image or video after this.fileInput.click()
        this.fileInput.onchange = e => {
          this.setCorrelationId(uuidv4());
          if (e.target.files.length) {
              const files = Array.from(e.target.files);
              embedUploadFunction(this.chatId, files, this.chatStore.correlationId, this.fileItemUuid)
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

<style lang="stylus">
@import "constants.styl"

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
  border-bottom: 1px solid;
  height: 100%;
  overflow-y: auto;
  border-color: $borderColor;
  line-height: $lineHeight;
}

.richText__content {
  padding: 6px 6px;
  flex: 1 1 auto;
  overflow-x: hidden;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
}

.richText__content__mobile {
  margin: 12px 12px 18px 12px;
}

.richText__content p {
    margin-bottom: unset
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
