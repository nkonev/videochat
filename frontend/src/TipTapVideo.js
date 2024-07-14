import { Node, mergeAttributes } from '@tiptap/core';

// https://www.codemzy.com/blog/tiptap-video-embed-extension
const Video = Node.create({
    name: 'video', // unique name for the Node
    group: 'block',
    selectable: true, // so we can select the video
    draggable: true, // so we can drag the video
    atom: true, // is a single unit
    // inline: true,

    parseHTML() {
        return [
            {
                tag: 'video',
            },
        ]
    },
    addAttributes() {
        return {
            "src": {
                default: null
            },
            "poster": {
                default: null
            },
        }
    },
    renderHTML({ HTMLAttributes }) {
        return [
            'div', {"class": "video-in-message-wrapper"},
            ['video', mergeAttributes({"class": "video-custom-class", "controls": true}, HTMLAttributes)],
            ['span', {"class": "video-in-message-button mdi mdi-play-box-outline"}]
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
