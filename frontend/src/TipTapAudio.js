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
        ]
    },
    addAttributes() {
        return {
            "src": {
                default: null
            },
        }
    },
    renderHTML({ HTMLAttributes }) {
        return ['audio', mergeAttributes({"class": "audio-custom-class", "controls": true}, HTMLAttributes)];
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
