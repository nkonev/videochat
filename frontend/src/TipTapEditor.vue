<template>
  <div class="richText">
    <input id="file-input" type="file" style="display: none;" accept="image/*,video/*,audio/*" multiple="multiple" />
    <div :class="editorContainer()">
      <!--
       we can add !chatStore.shouldShowSendMessageButtons
       it will lead to the redrawing the editor on opening video and
       changing the placeholder (gray text) to the shorter version,
       but it also shows the bubble menu on the bottom of the page,
       breaking the markup by adding an excess scroll
       -->
      <div v-if="editor">
        <bubble-menu
            :should-show="shouldShowBubbleMenu"
            class="bubble-menu"
            :updateDelay="0"
            :resizeDelay="0"
            :editor="editor"
        >
            <button @click="boldClick" :class="{ 'is-active': boldValue() }">
              {{ $vuetify.locale.t('$vuetify.message_edit_bold_short') }}
            </button>
            <button @click="italicClick" :class="{ 'is-active': italicValue() }">
              {{ $vuetify.locale.t('$vuetify.message_edit_italic_short') }}
            </button>
            <button @click="underlineClick" :class="{ 'is-active': underlineValue() }">
              {{ $vuetify.locale.t('$vuetify.message_edit_underline_short') }}
            </button>
        </bubble-menu>

        <floating-menu
            :editor="editor"
            class="floating-menu"
            :should-show="shouldShowFloatingMenu"
        >
            <button @click="bulletListClick" :class="{ 'is-active': bulletListValue() }">
              {{ $vuetify.locale.t('$vuetify.message_edit_bullet_list_short') }}
            </button>
            <button @click="orderedListClick" :class="{ 'is-active': orderedListValue() }">
              {{ $vuetify.locale.t('$vuetify.message_edit_ordered_list_short') }}
            </button>

            <button @click="imageClick">
              {{ $vuetify.locale.t('$vuetify.message_edit_image_short') }}
            </button>
            <button @click="videoClick">
              {{ $vuetify.locale.t('$vuetify.message_edit_video_short') }}
            </button>
            <button @click="embedClick">
              {{ $vuetify.locale.t('$vuetify.message_edit_embed_short') }}
            </button>
        </floating-menu>
      </div>

      <editor-content :editor="editor" class="editorContent" :title="$vuetify.locale.t('$vuetify.message_edit_placeholder')" />
    </div>
  </div>
</template>

<script>
import "prosemirror-view/style/prosemirror.css";
import "./messageBody.styl";
// https://github.com/ueberdosis/tiptap/pull/5398/files
// https://github.com/ueberdosis/tiptap/blob/next/demos/src/Examples/Menus/Vue/index.vue
import {Editor, EditorContent, isTextSelection} from "@tiptap/vue-3";
import { BubbleMenu, FloatingMenu } from '@tiptap/vue-3/menus'
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
import { TextStyle, Color } from '@tiptap/extension-text-style';
import Highlight from "@tiptap/extension-highlight";
import Mention from '@tiptap/extension-mention';
import Code from '@tiptap/extension-code';
import CodeBlock from '@tiptap/extension-code-block';
import BulletList from '@tiptap/extension-bullet-list';
import OrderedList from '@tiptap/extension-ordered-list';
import ListItem from '@tiptap/extension-list-item';
import Blockquote from '@tiptap/extension-blockquote';
import {buildImageHandler} from '@/TipTapImage';
import suggestion from './suggestion';
import {
  defaultAudioPreviewUrl, defaultIframePreviewUrl,
  defaultVideoPreviewUrl,
  embed,
  hasLength, isMobileBrowser,
  link_dialog_type_add_media_embed,
  media_audio,
  media_image,
  media_video
} from "@/utils";
import bus, {
  FILE_UPLOAD_MODAL_START_UPLOADING,
  PREVIEW_CREATED,
  OPEN_FILE_UPLOAD_MODAL,
  MEDIA_LINK_SET,
  EMBED_LINK_SET, OPEN_MESSAGE_EDIT_MEDIA, OPEN_MESSAGE_EDIT_LINK, FILE_CREATED, SET_FILE_CORRELATION_ID,
} from "./bus/bus";
import Video from "@/TipTapVideo";
import Audio from "@/TipTapAudio";
import Iframe from '@/TipTapIframe';
import { v4 as uuidv4 } from 'uuid';
import {getStoredMessageEditNormalizeText, getTreatNewlinesAsInHtml} from "@/store/localStore.js";
import {mapStores} from "pinia";
import {fileUploadingSessionTypeMedia, fileUploadingSessionTypeMessageEdit, useChatStore} from "@/store/chatStore.js";
import {mergeAttributes} from "@tiptap/core";
import {profile} from "@/router/routes.js";

const empty = "";

const embedUploadFunction = (chatId, files, correlationId, fileItemId, shouldAddDateToTheFilename) => {
    bus.emit(OPEN_FILE_UPLOAD_MODAL, {
      showFileInput: true,
      fileItemUuid: fileItemId,
      shouldSetFileUuidToMessage: true,
      predefinedFiles: files,
      correlationId: correlationId,
      shouldAddDateToTheFilename: shouldAddDateToTheFilename,
      fileUploadingSessionType: fileUploadingSessionTypeMedia,
    });
    bus.emit(FILE_UPLOAD_MODAL_START_UPLOADING);
}

const domParser = new DOMParser();

export default {
  components: {
    EditorContent,
    BubbleMenu,
    FloatingMenu,
  },

  data() {
    return {
      editor: null,
      fileInput: null,
      fileItemUuid: null,
      receivedPreviews: 0,
      receivedFiles: 0,
      fileCorrelationId: null,
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
        this.receivedPreviews = 0;
        this.receivedFiles = 0;
    },
    clearContent() {
        this.editor.commands.setContent(empty, false);
        this.receivedPreviews = 0;
        this.receivedFiles = 0;
        this.resetFileCorrelationId();
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
        return this.editor.commands.focus('end')
      }).then(()=>{
        this.editor.commands.scrollIntoView() // fix an issue related to long text and mobile virtual keyboard
      })
    },
    scrollToCursor() {
       this.editor.commands.scrollIntoView() // fix an issue related to long text and mobile virtual keyboard
    },
    resetFileCorrelationId() {
      this.setFileCorrelationId(null);
    },
    setFileCorrelationId(c) {
      this.fileCorrelationId = c;
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
            this.editor.chain().focus().setVideo({src: defaultVideoPreviewUrl, original: src}).run()
        }
    },
    setAudio(src) {
        this.editor.chain().focus().setAudio({src: defaultAudioPreviewUrl, original: src}).run()
    },
    setIframe(obj) {
      if (obj.src) {
          this.editor.chain().focus().setIframe(obj).run()
      }
    },
    onPreviewCreated(dto) {
        if (hasLength(this.fileCorrelationId) && this.fileCorrelationId == dto.correlationId) {
            if (dto.aType == media_video) {
                this.setVideo(dto.url, dto.previewUrl)
            } else if (dto.aType == media_image) {
                this.setImage(dto.url, dto.previewUrl)
            }
        }

        if (this.chatStore.sendMessageAfterUploadsUploaded && this.chatStore.fileUploadingSessionType == fileUploadingSessionTypeMedia && this.chatStore.sendMessageAfterUploadFileItemUuid == dto?.fileItemUuid) {
          this.receivedPreviews++;
          console.log("Got previews ", this.receivedPreviews, "expected", this.chatStore.sendMessageAfterMediaNumFiles)

          if (this.chatStore.sendMessageAfterMediaNumFiles <= this.receivedPreviews) {
            this.$emit("sendMessage"); // TODO introduce sendMessageTask in chatStore, before sending the message we delete the task, if the message in the same chat as tick was clicked we send the message, otherwise we just delete the task
            this.chatStore.resetSendMessageAfterMediaInsertRoutine();
            this.receivedPreviews = 0;
          }
        }
    },
    onFileCreatedEvent(dto) {
      // it's for embedded audio. for embedded (image, video, record) - see onPreviewCreated()
      if (hasLength(this.fileCorrelationId) && this.fileCorrelationId == dto?.correlationId) {
        if (dto?.fileInfoDto != null && !dto.fileInfoDto.previewable && dto.fileInfoDto.aType == media_audio) {
          this.setAudio(dto?.fileInfoDto.url)
        }
      }

      // for any files loaded via MessageEdit.
      if (this.chatStore.sendMessageAfterUploadsUploaded && this.chatStore.fileUploadingSessionType == fileUploadingSessionTypeMessageEdit && this.chatStore.sendMessageAfterUploadFileItemUuid == dto?.fileInfoDto?.fileItemUuid) {
        this.receivedFiles++;
        console.log("Got files ", this.receivedFiles, "expected", this.chatStore.sendMessageAfterNumFiles)

        if (this.chatStore.sendMessageAfterNumFiles <= this.receivedFiles) {
          this.$emit("sendMessage");
          this.chatStore.resetSendMessageAfterFileInsertRoutine();
          this.receivedFiles = 0;
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
        console.debug("onEmbedLinkSet", link);
        if (link && !link.startsWith("http")) {
            const htmlDoc = domParser.parseFromString(link, 'text/html');
            const iframes = htmlDoc.getElementsByTagName('iframe');
            if (iframes.length == 1) {
                const iframe = iframes[0];
                this.setIframe({src: defaultIframePreviewUrl, original: iframe.src, width: iframe.width, height: iframe.height, allowfullscreen: iframe.getAttribute('allowfullscreen') != null});
                return
            }
        }

        this.setIframe({src: defaultIframePreviewUrl, original: link});
    },
    addText(text) {
      this.editor.commands.insertContent(text)
    },
    setFileItemUuid(fileItemUuid) {
        this.fileItemUuid = fileItemUuid;
    },
    resetFileItemUuid() {
      this.setFileItemUuid(null);
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
    boldValue() {
      return this.editor.isActive('bold')
    },
    boldClick() {
      this.editor.chain().focus().toggleBold().run()
    },
    italicValue() {
      return this.editor.isActive('italic')
    },
    italicClick() {
      this.editor.chain().focus().toggleItalic().run()
    },
    underlineValue() {
      return this.editor.isActive('underline')
    },
    underlineClick() {
      this.editor.chain().focus().toggleUnderline().run()
    },
    bulletListClick() {
      this.editor.chain().focus().toggleBulletList().run()
    },
    orderedListClick() {
      this.editor.chain().focus().toggleOrderedList().run()
    },
    bulletListValue() {
      return this.editor.isActive('bulletList')
    },
    orderedListValue() {
      return this.editor.isActive('orderedList')
    },
    imageClick() {
      bus.emit(
          OPEN_MESSAGE_EDIT_MEDIA,
          {
            chatId: this.chatId,
            type: media_image,
            fromDiskCallback: () => this.addImage(),
            setExistingMediaCallback: this.setImage
          }
      );
    },
    videoClick() {
      bus.emit(
          OPEN_MESSAGE_EDIT_MEDIA,
          {
            chatId: this.chatId,
            type: media_video,
            fromDiskCallback: () => this.addVideo(),
            setExistingMediaCallback: this.setVideo
          },
      );
    },
    embedClick() {
      bus.emit(OPEN_MESSAGE_EDIT_LINK, {dialogType: link_dialog_type_add_media_embed, mediaType: embed});
    },
    // patched version of frontend/node_modules/@tiptap/extension-bubble-menu/src/bubble-menu-plugin.ts
    shouldShowBubbleMenu({
      editor,
      view,
      state,
      from,
      to,
    }) {
      // patch start
      if (this.chatStore.shouldShowSendMessageButtons) {
        return false
      }
      // patch end

      const { doc, selection } = state
      const { empty } = selection

      // Sometime check for `empty` is not enough.
      // Doubleclick an empty paragraph returns a node size of 2.
      // So we check also for an empty text size.
      const isEmptyTextBlock = !doc.textBetween(from, to).length && isTextSelection(state.selection)

      // When clicking on a element inside the bubble menu the editor "blur" event
      // is called and the bubble menu item is focussed. In this case we should
      // consider the menu as part of the editor and keep showing the menu
      const isChildOfMenu = false; // patch because there is no element

      const hasEditorFocus = view.hasFocus() || isChildOfMenu

      if (!hasEditorFocus || empty || isEmptyTextBlock || !editor.isEditable) {
        return false
      }

      return true
    },

    // patched version of frontend/node_modules/@tiptap/extension-floating-menu/src/floating-menu-plugin.ts
    shouldShowFloatingMenu({ editor, view, state }) {
      // patch start
      if (this.chatStore.shouldShowSendMessageButtons) {
          return false
      }
      // patch end

      const { selection } = state
      const { $anchor, empty } = selection
      const isRootDepth = $anchor.depth === 1
      const isEmptyTextBlock = $anchor.parent.isTextblock && !$anchor.parent.type.spec.code && !$anchor.parent.textContent

      if (
          !view.hasFocus()
          || !empty
          || !isRootDepth
          || !isEmptyTextBlock
          || !editor.isEditable
      ) {
        return false
      }

      return true
    },
    documentBody() {
      return document.body
    },
  },
  mounted() {
    bus.on(PREVIEW_CREATED, this.onPreviewCreated);
    bus.on(FILE_CREATED, this.onFileCreatedEvent);
    bus.on(MEDIA_LINK_SET, this.onMediaLinkSet);
    bus.on(EMBED_LINK_SET, this.onEmbedLinkSet);
    bus.on(SET_FILE_CORRELATION_ID, this.setFileCorrelationId)

    const imagePluginInstance = buildImageHandler(
    (image, shouldAddDateToTheFilename) => {
        const correlationId = uuidv4();
        this.setFileCorrelationId(correlationId);
        embedUploadFunction(this.chatId, [image], correlationId, this.fileItemUuid, shouldAddDateToTheFilename);
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
                return this.$vuetify.locale.t('$vuetify.message_edit_placeholder_short')
              },
          }),
          Text,
          Blockquote,
          Iframe,
          Audio,
          Video,
          imagePluginInstance,
          Italic,
          Bold,
          Strike,
          Underline,
          Link.configure({
              openOnClick: false,
              linkOnPaste: true,
              autolink: true,
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
                          `@${node.attrs.label}`,
                      ];
                  } else {
                      return [
                          "span",
                          options.HTMLAttributes,
                          `@${node.attrs.label}`,
                      ];
                  }
              },
          }),
          Code,
          CodeBlock,
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
          if (e.target.files.length) {
              const files = Array.from(e.target.files);
              const correlationId = uuidv4();
              this.setFileCorrelationId(correlationId);
              embedUploadFunction(this.chatId, files, correlationId, this.fileItemUuid)
          }
        }
    })

    this.$emit("editorIsReady")
  },

  beforeUnmount() {
    bus.off(PREVIEW_CREATED, this.onPreviewCreated);
    bus.off(FILE_CREATED, this.onFileCreatedEvent);
    bus.off(MEDIA_LINK_SET, this.onMediaLinkSet);
    bus.off(EMBED_LINK_SET, this.onEmbedLinkSet);
    bus.off(SET_FILE_CORRELATION_ID, this.setFileCorrelationId)
    this.resetFileItemUuid();
    this.resetFileCorrelationId();

    this.editor.destroy();
    if (this.fileInput) {
      this.fileInput.onchange = null;
    }
    this.fileInput = null;
    this.receivedPreviews = 0;
    this.receivedFiles = 0;
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

  blockquote {
    border-left: 3px solid gray;
    margin: 0.6rem 0;
    padding-left: 1rem;
  }
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

.bubble-menu {
  display: flex;
  background-color: #0D0D0D;
  padding: 0.2rem;
  border-radius: 0.5rem;

  button {
    border: none;
    background: none;
    color: #FFF;
    font-size: 0.85rem;
    font-weight: 500;
    padding: 0 0.2rem;
    opacity: 0.6;

    &:hover,
    &.is-active {
      opacity: 1;
    }
  }
}

.floating-menu {
  display: flex;
  background-color: #f1f1f1;
  padding: 0.2rem;
  border-radius: 0.5rem;

  button {
    border: none;
    background: none;
    font-size: 0.85rem;
    font-weight: 500;
    padding: 0 0.2rem;
    opacity: 0.6;

    &:hover,
    &.is-active {
      opacity: 1;
    }
  }
}

</style>
