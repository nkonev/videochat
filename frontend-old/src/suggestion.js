import { VueRenderer } from '@tiptap/vue-2'
import tippy from 'tippy.js'
import axios from "axios";

import MentionList from './MentionList.vue'

export default (chatId) => {

    return {
        items: ({query}) => {
            return axios.get(`/api/chat/${chatId}/suggest-participants`, {
                params: {
                    searchString: query,
                },
            }).then(({data}) => {
                return data.map(item => item.login)
            })
        },

        render: () => {
            let component
            let popup

            return {
                onStart: props => {
                    component = new VueRenderer(MentionList, {
                        // using vue 2:
                        parent: this,
                        propsData: props,
                        // using vue 3:
                        // props,
                        // editor: props.editor,
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