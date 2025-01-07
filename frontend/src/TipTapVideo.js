import { Node, mergeAttributes } from '@tiptap/core';
import {hasLength} from "@/utils.js";

// https://www.codemzy.com/blog/tiptap-video-embed-extension
const Video = Node.create({
    name: 'video', // unique name for the Node
    group: 'inline',
    selectable: true, // so we can select the video
    draggable: true, // so we can drag the video
    atom: true, // is a single unit
    inline: true,

    parseHTML() {
        return [
            {
                tag: 'video',
            },
            {
                tag: 'img[class="video-custom-class"]',
            },
        ]
    },
    addAttributes() {
        return {
            "src": {
                default: null
            },
            original: {
                default: null,
                parseHTML: element => element.getAttribute('data-original'),
                renderHTML: attributes => {
                    if (!attributes.original) {
                        return {};
                    }
                    return {
                        'data-original': attributes.original,
                    };
                },
            },
        }
    },
    renderHTML({ HTMLAttributes }) {
        if (hasLength(HTMLAttributes["data-original"])) {
            return [
                'span', {"class": "media-in-message-wrapper media-in-message-wrapper-video"},
                ['img', mergeAttributes({"class": "video-custom-class"}, HTMLAttributes)],
                ['span', {"class": "media-in-message-button-open mdi mdi-arrow-expand-all", "title": "Open in player"}],
                ['span', {
                    "class": "media-in-message-button-replace mdi mdi-play-box-outline",
                    "title": "Play in-place"
                }],
            ];
        } else {
            return [
                'span', {"class": "media-in-message-wrapper media-in-message-wrapper-video"},
                ['video', mergeAttributes({"class": "video-custom-class", "controls": true}, HTMLAttributes)],
            ];
        }
    },
    addCommands() {
        return {
            setVideo: options => ({ commands }) => {
                return commands.insertContent({
                    type: this.name,
                    attrs: options,
                })
            },
        }
    },
});

export default Video;
