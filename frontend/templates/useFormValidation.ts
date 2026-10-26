// copy verbatim; edit only the marked regions
import { reactive, ref } from 'vue'

export function useFormValidation() {
  const submitted = ref(false)
  const touched = reactive(new Set<string>())

  function blur(key: string) {
    touched.add(key)
  }

  function show(key: string, err: string): string {
    return submitted.value || touched.has(key) ? err : ''
  }

  function reset() {
    submitted.value = false
    touched.clear()
  }

  return { submitted, blur, show, reset }
}
