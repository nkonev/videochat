import { Node, mergeAttributes } from '@tiptap/core';

// https://www.codemzy.com/blog/tiptap-video-embed-extension
const Audio = Node.create({
    name: 'audio', // unique name for the Node
    group: 'inline',
    selectable: true, // so we can select the video
    draggable: true, // so we can drag the video
    atom: true, // is a single unit
    inline: true,

    parseHTML() {
        return [
            {
                tag: 'audio',
            },
            {
                tag: 'img[class="audio-custom-class"]',
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
        return [
            'span', {"class": "media-in-message-wrapper media-in-message-wrapper-audio"},
            ['img', mergeAttributes({"class": "audio-custom-class"}, HTMLAttributes)],
            ['span', {"class": "media-in-message-button-open mdi mdi-arrow-expand-all", "title": "Open in player"}],
            ['span', {"class": "media-in-message-button-replace mdi mdi-play-box-outline", "title": "Play in-place"}],
        ];
    },
    addCommands() {
        return {
            setAudio: options => ({ commands }) => {
                return commands.insertContent({
                    type: this.name,
                    attrs: options,
                })
            },
        }
    },
});

export default Audio;
