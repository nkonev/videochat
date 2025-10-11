import {
    computePosition,
    flip,
    shift,
} from '@floating-ui/dom'
import { posToDOMRect, VueRenderer } from '@tiptap/vue-3'
import axios from "axios";

import MentionList from './MentionList.vue'

// https://github.com/ueberdosis/tiptap/pull/5398/files
const updatePosition = (editor, element) => {
    const virtualElement = {
        getBoundingClientRect: () => posToDOMRect(editor.view, editor.state.selection.from, editor.state.selection.to),
    }

    computePosition(virtualElement, element, {
        placement: 'bottom-start',
        strategy: 'absolute',
        middleware: [shift(), flip()],
    }).then(({ x, y, strategy }) => {
        element.style.width = 'max-content'
        element.style.position = strategy
        element.style.left = `${x}px`
        element.style.top = `${y}px`
    })
}

export default (tipTapEditorVue) => {

    return {
        items: ({query}) => {
            const chatId = tipTapEditorVue.$route.params.id;
            return axios.get(`/api/chat/${chatId}/mention/suggest`, {
                params: {
                    searchString: query,
                },
            }).then(({data}) => {
                return data.map((item) => {return {id: item.id, label: item.login}})
            })
        },

        render: () => {
            let component

            return {
                onStart: props => {
                    component = new VueRenderer(MentionList, {
                        // using vue 3:
                        props,
                        editor: props.editor,
                    })

                    if (!props.clientRect) {
                        return
                    }

                    component.element.style.position = 'absolute'
                    document.body.appendChild(component.element)
                    updatePosition(props.editor, component.element)

                },

                onUpdate(props) {
                    component.updateProps(props)

                    if (!props.clientRect) {
                        return
                    }

                    updatePosition(props.editor, component.element)
                },

                onKeyDown(props) {
                    if (props.event.key === 'Escape') {

                        component.destroy()
                        component.element.remove()

                        return true
                    }

                    return component.ref?.onKeyDown(props)
                },

                onExit() {
                    component.destroy()
                    component.element.remove()
                },
            }
        },
    }
}
