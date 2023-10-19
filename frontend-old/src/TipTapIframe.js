import { Node, mergeAttributes } from '@tiptap/core';

export default Node.create({
    name: 'iframe',
    group: 'inline',
    selectable: true, // so we can select the video
    draggable: true, // so we can drag the video
    atom: true, // is a single unit
    inline: true,

    addAttributes() {
        return {
            src: {
                default: null,
            },
        }
    },

    parseHTML() {
        return [{
            tag: 'iframe',
        }]
    },

    renderHTML({ HTMLAttributes }) {
        return ['iframe', mergeAttributes({"class": "iframe-custom-class"}, HTMLAttributes)];
    },

    addCommands() {
        return {
            setIframe: options => ({ commands }) => {
                return commands.insertContent({
                    type: this.name,
                    attrs: options,
                })
            },
        }
    },
})
