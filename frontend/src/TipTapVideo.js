import { Node, mergeAttributes } from '@tiptap/core';

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
        return [
            'span', {"class": "video-in-message-wrapper"},
            ['img', mergeAttributes({"class": "video-custom-class", "controls": true}, HTMLAttributes)],
            ['span', {"class": "video-in-message-button mdi mdi-fullscreen"}],
            ['span', {"class": "video-in-message-button-replace mdi mdi-play-box-outline"}],
        ];
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
