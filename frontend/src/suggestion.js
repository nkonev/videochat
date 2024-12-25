import { VueRenderer } from '@tiptap/vue-3'
import tippy from 'tippy.js'
import axios from "axios";

import MentionList from './MentionList.vue'

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
            let popup

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

                    popup = tippy('body', {
                        getReferenceClientRect: props.clientRect,
                        appendTo: () => document.body,
                        content: component.element,
                        showOnCreate: true,
                        interactive: true,
                        trigger: 'manual',
                        placement: 'bottom-start',
                        hideOnClick: 'toggle'
                    })
                },

                onUpdate(props) {
                    component.updateProps(props)

                    if (!props.clientRect) {
                        return
                    }

                    popup[0].setProps({
                        getReferenceClientRect: props.clientRect,
                    })
                },

                onKeyDown(props) {
                    if (props.event.key === 'Escape') {
                        popup[0].hide()

                        return true
                    }

                    return component.ref?.onKeyDown(props)
                },

                onExit() {
                    popup[0].destroy()
                    component.destroy()
                },
            }
        },
    }
}
