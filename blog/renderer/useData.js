// https://vike.dev/useData
export { useData, getData }

import { computed } from 'vue'
import { usePageContext } from './usePageContext'

/** https://vike.dev/useData */
function useData() {
  const data = computed(() => usePageContext().data)
  return data
}

function getData() {
    return usePageContext().data;
}
