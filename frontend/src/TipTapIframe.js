import { Node, mergeAttributes } from '@tiptap/core';

export default Node.create({
    name: 'iframe',
    group: 'inline',
    selectable: true, // so we can select the video
    draggable: true, // so we can drag the video
    atom: true, // is a single unit
    inline: true,

    parseHTML() {
        return [
            {
                tag: 'iframe',
            },
            {
                tag: 'img[class="iframe-custom-class"]',
            },
        ]
    },

    addAttributes() {
        return {
            src: {
                default: null,
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

            width: {
                default: null,
                parseHTML: element => element.getAttribute('data-width'),
                renderHTML: attributes => {
                    if (!attributes.width) {
                        return {};
                    }
                    return {
                        'data-width': attributes.width,
                    };
                },
            },
            height: {
                default: null,
                parseHTML: element => element.getAttribute('data-height'),
                renderHTML: attributes => {
                    if (!attributes.height) {
                        return {};
                    }
                    return {
                        'data-height': attributes.height,
                    };
                },
            },
            allowfullscreen: {
                default: null,
                parseHTML: element => element.getAttribute('data-allowfullscreen'),
                renderHTML: attributes => {
                    if (!attributes.allowfullscreen) {
                        return {};
                    }
                    return {
                        'data-allowfullscreen': attributes.allowfullscreen,
                    };
                },
            },
        }
    },

    renderHTML({ HTMLAttributes }) {
        return [
            'span', {"class": "media-in-message-wrapper media-in-message-wrapper-iframe"},
            ['img', mergeAttributes({"class": "iframe-custom-class"}, HTMLAttributes)],
            ['span', { "class": "media-in-message-button-replace media-in-message-button-replace-first mdi mdi-play-box-outline", "title": "Play in-place"}],
        ];
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
